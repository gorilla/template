// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package parse

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

// Parse returns a map from template name to parse.Tree, created by parsing the
// templates described in the argument string. The top-level template will be
// given the specified name. If an error is encountered, parsing stops and an
// empty map is returned with the error.
func Parse(name, text, leftDelim, rightDelim string, funcs ...map[string]interface{}) (Tree, error) {
	return new(parser).parse(name, text, leftDelim, rightDelim, funcs...)
}

// parser parses a single template into a tree.
type parser struct {
	name      string // template being parsed, for error messages.
	text      string
	lex       *lexer
	tree      Tree // tree being built.
	funcs     []map[string]interface{}
	vars      []string // variables defined at the moment.
	token     [3]item  // three-token lookahead for parser.
	peekCount int
}

// next returns the next token.
func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}
	return p.token[p.peekCount]
}

// backup backs the input stream up one token.
func (p *parser) backup() {
	p.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (p *parser) backup2(t1 item) {
	p.token[1] = t1
	p.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (p *parser) backup3(t2, t1 item) { // Reverse order: we're pushing back.
	p.token[1] = t1
	p.token[2] = t2
	p.peekCount = 3
}

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}

// nextNonSpace returns the next non-space token.
func (p *parser) nextNonSpace() (token item) {
	for {
		token = p.next()
		if token.typ != itemSpace {
			break
		}
	}
	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (p *parser) peekNonSpace() (token item) {
	for {
		token = p.next()
		if token.typ != itemSpace {
			break
		}
	}
	p.backup()
	return token
}

// Parsing.

// ErrorContext returns a textual representation of the location of the node in the input text.
func (p *parser) ErrorContext(n Node) (location, context string) {
	pos := int(n.Position())
	text := p.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	context = n.String()
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}
	return fmt.Sprintf("%s:%d:%d", p.name, lineNum, byteNum), context
}

// errorf formats the error and terminates processing.
func (p *parser) errorf(format string, args ...interface{}) {
	format = fmt.Sprintf("template: %s:%d: %s", p.name, p.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (p *parser) error(err error) {
	p.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (p *parser) expect(expected itemType, context string) item {
	token := p.nextNonSpace()
	if token.typ != expected {
		p.errorf("expected %s in %s; got %s", expected, context, token)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (p *parser) expectOneOf(expected1, expected2 itemType, context string) item {
	token := p.nextNonSpace()
	if token.typ != expected1 && token.typ != expected2 {
		p.errorf("expected %s or %s in %s; got %s", expected1, expected2, context, token)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (p *parser) unexpected(token item, context string) {
	p.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (p *parser) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
	return
}

// atEOF returns true if, possibly after spaces, we're at EOF.
func (p *parser) atEOF() bool {
	for {
		token := p.peek()
		switch token.typ {
		case itemEOF:
			return true
		case itemText:
			for _, r := range token.val {
				if !unicode.IsSpace(r) {
					return false
				}
			}
			p.next() // skip spaces.
			continue
		}
		break
	}
	return false
}

// parse is the top-level parser for a template: it parses {{define}} actions
// and add the define nodes to the tree. It runs to EOF.
func (p *parser) parse(name, text, leftDelim, rightDelim string, funcs ...map[string]interface{}) (tree Tree, err error) {
	defer p.recover(&err)
	p.name = name
	p.text = text
	p.lex = lex(name, text, leftDelim, rightDelim)
	p.tree = make(Tree)
	p.funcs = funcs
	p.vars = []string{"$"}
	for {
		switch p.next().typ {
		case itemEOF:
			return p.tree, nil
		case itemLeftDelim:
			token := p.expect(itemDefine, "template root")
			if err = p.tree.Add(p.parseDefinition(token.pos)); err != nil {
				p.error(err)
			}
		}
	}
	return p.tree, nil
}

// parseDefinition parses a {{define}} ... {{end}} template definition and
// returns a defineNode. The "define" keyword has already been scanned.
//
//	{{define stringValue}} itemList {{end}}
//	{{define stringValue stringValue}} itemList {{end}}
func (p *parser) parseDefinition(pos Pos) *DefineNode {
	const context = "define clause"
	defer p.popVars(1)
	line := p.lex.lineNumber()
	var name, parent string
	token := p.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		name = s
	default:
		p.unexpected(token, context)
	}
	token = p.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		parent = s
		p.expect(itemRightDelim, context)
	case itemRightDelim:
	default:
		p.unexpected(token, context)
	}
	list, end := p.itemList()
	if end.Type() != nodeEnd {
		p.errorf("unexpected %s in %s", end, context)
	}
	return newDefine(pos, line, name, parent, list, p.text)
}

// itemList:
//	textOrAction*
// Terminates at {{end}} or {{else}}, returned separately.
func (p *parser) itemList() (list *ListNode, next Node) {
	list = newList(p.peekNonSpace().pos)
	for p.peekNonSpace().typ != itemEOF {
		n := p.textOrAction()
		switch n.Type() {
		case nodeEnd, nodeElse:
			return list, n
		}
		list.append(n)
	}
	p.errorf("unexpected EOF")
	return
}

// textOrAction:
//	text | action
func (p *parser) textOrAction() Node {
	switch token := p.nextNonSpace(); token.typ {
	case itemText:
		return newText(token.pos, token.val)
	case itemLeftDelim:
		return p.action()
	default:
		p.unexpected(token, "input")
	}
	return nil
}

// Action:
//	control
//	command ("|" command)*
// Left delim is past. Now get actions.
// First word could be a keyword such as range.
func (p *parser) action() (n Node) {
	switch token := p.nextNonSpace(); token.typ {
	case itemElse:
		return p.elseControl()
	case itemEnd:
		return p.endControl()
	case itemIf:
		return p.ifControl()
	case itemRange:
		return p.rangeControl()
	case itemTemplate:
		return p.templateControl()
	case itemWith:
		return p.withControl()
	case itemBlock:
		return p.blockControl()
	case itemFill:
		return p.fillControl()
	}
	p.backup()
	// Do not pop variables; they persist until "end".
	return newAction(p.peek().pos, p.lex.lineNumber(), p.pipeline("command"))
}

// Pipeline:
//	declarations? command ('|' command)*
func (p *parser) pipeline(context string) (pipe *PipeNode) {
	var decl []*VariableNode
	pos := p.peekNonSpace().pos
	// Are there declarations?
	for {
		if v := p.peekNonSpace(); v.typ == itemVariable {
			p.next()
			// Since space is a token, we need 3-token look-ahead here in the worst case:
			// in "$x foo" we need to read "foo" (as opposed to ":=") to know that $x is an
			// argument variable rather than a declaration. So remember the token
			// adjacent to the variable so we can push it back if necessary.
			tokenAfterVariable := p.peek()
			if next := p.peekNonSpace(); next.typ == itemColonEquals || (next.typ == itemChar && next.val == ",") {
				p.nextNonSpace()
				variable := newVariable(v.pos, v.val)
				decl = append(decl, variable)
				p.vars = append(p.vars, v.val)
				if next.typ == itemChar && next.val == "," {
					if context == "range" && len(decl) < 2 {
						continue
					}
					p.errorf("too many declarations in %s", context)
				}
			} else if tokenAfterVariable.typ == itemSpace {
				p.backup3(v, tokenAfterVariable)
			} else {
				p.backup2(v)
			}
		}
		break
	}
	pipe = newPipeline(pos, p.lex.lineNumber(), decl)
	for {
		switch token := p.nextNonSpace(); token.typ {
		case itemRightDelim, itemRightParen:
			if len(pipe.Cmds) == 0 {
				p.errorf("missing value for %s", context)
			}
			if token.typ == itemRightParen {
				p.backup()
			}
			return
		case itemBool, itemCharConstant, itemComplex, itemDot, itemField, itemIdentifier,
			itemNumber, itemNil, itemRawString, itemString, itemVariable, itemLeftParen:
			p.backup()
			pipe.append(p.command())
		default:
			p.unexpected(token, context)
		}
	}
	return
}

func (p *parser) parseControl(context string) (pos Pos, line int, pipe *PipeNode, list, elseList *ListNode) {
	defer p.popVars(len(p.vars))
	line = p.lex.lineNumber()
	pipe = p.pipeline(context)
	var next Node
	list, next = p.itemList()
	switch next.Type() {
	case nodeEnd: //done
	case nodeElse:
		elseList, next = p.itemList()
		if next.Type() != nodeEnd {
			p.errorf("expected end; found %s", next)
		}
		elseList = elseList
	}
	return pipe.Position(), line, pipe, list, elseList
}

// If:
//	{{if pipeline}} itemList {{end}}
//	{{if pipeline}} itemList {{else}} itemList {{end}}
// If keyword is past.
func (p *parser) ifControl() Node {
	return newIf(p.parseControl("if"))
}

// Range:
//	{{range pipeline}} itemList {{end}}
//	{{range pipeline}} itemList {{else}} itemList {{end}}
// Range keyword is past.
func (p *parser) rangeControl() Node {
	return newRange(p.parseControl("range"))
}

// With:
//	{{with pipeline}} itemList {{end}}
//	{{with pipeline}} itemList {{else}} itemList {{end}}
// If keyword is past.
func (p *parser) withControl() Node {
	return newWith(p.parseControl("with"))
}

// End:
//	{{end}}
// End keyword is past.
func (p *parser) endControl() Node {
	return newEnd(p.expect(itemRightDelim, "end").pos)
}

// Else:
//	{{else}}
// Else keyword is past.
func (p *parser) elseControl() Node {
	return newElse(p.expect(itemRightDelim, "else").pos, p.lex.lineNumber())
}

// Template:
//	{{template stringValue pipeline}}
// Template keyword is past.  The name must be something that can evaluate
// to a string.
func (p *parser) templateControl() Node {
	var name string
	token := p.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		name = s
	default:
		p.unexpected(token, "template invocation")
	}
	var pipe *PipeNode
	if p.nextNonSpace().typ != itemRightDelim {
		p.backup()
		// Do not pop variables; they persist until "end".
		pipe = p.pipeline("template")
	}
	return newTemplate(token.pos, p.lex.lineNumber(), name, pipe)
}

// Block:
//	{{block stringValue}}
// Block keyword is past.
func (p *parser) blockControl() Node {
	const context = "block definition"
	var name string
	token := p.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		name = s
	default:
		p.unexpected(token, context)
	}
	p.expect(itemRightDelim, context)
	list, end := p.itemList()
	if end.Type() != nodeEnd {
		p.errorf("unexpected %s in %s", end, context)
	}
	return newBlock(token.pos, p.lex.lineNumber(), name, list)
}

// Fill:
//	{{fill stringValue}} itemList {{end}}
// Fill keyword is past.
func (p *parser) fillControl() Node {
	const context = "fill definition"
	var name string
	token := p.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		name = s
	default:
		p.unexpected(token, context)
	}
	p.expect(itemRightDelim, context)
	list, end := p.itemList()
	if end.Type() != nodeEnd {
		p.errorf("unexpected %s in %s", end, context)
	}
	return newFill(token.pos, p.lex.lineNumber(), name, list)
}

// command:
//	operand (space operand)*
// space-separated arguments up to a pipeline character or right delimiter.
// we consume the pipe character but leave the right delim to terminate the action.
func (p *parser) command() *CommandNode {
	cmd := newCommand(p.peekNonSpace().pos)
	for {
		p.peekNonSpace() // skip leading spaces.
		operand := p.operand()
		if operand != nil {
			cmd.append(operand)
		}
		switch token := p.next(); token.typ {
		case itemSpace:
			continue
		case itemError:
			p.errorf("%s", token.val)
		case itemRightDelim, itemRightParen:
			p.backup()
		case itemPipe:
		default:
			p.errorf("unexpected %s in operand; missing space?", token)
		}
		break
	}
	if len(cmd.Args) == 0 {
		p.errorf("empty command")
	}
	return cmd
}

// operand:
//	term .Field*
// An operand is a space-separated component of a command,
// a term possibly followed by field accesses.
// A nil return means the next item is not an operand.
func (p *parser) operand() Node {
	node := p.term()
	if node == nil {
		return nil
	}
	if p.peek().typ == itemField {
		chain := newChain(p.peek().pos, node)
		for p.peek().typ == itemField {
			chain.Add(p.next().val)
		}
		// Compatibility with original API: If the term is of type NodeField
		// or NodeVariable, just put more fields on the original.
		// Otherwise, keep the Chain node.
		// TODO: Switch to Chains always when we can.
		switch node.Type() {
		case NodeField:
			node = newField(chain.Position(), chain.String())
		case NodeVariable:
			node = newVariable(chain.Position(), chain.String())
		default:
			node = chain
		}
	}
	return node
}

// term:
//	literal (number, string, nil, boolean)
//	function (identifier)
//	.
//	.Field
//	$
//	'(' pipeline ')'
// A term is a simple "expression".
// A nil return means the next item is not a term.
func (p *parser) term() Node {
	switch token := p.nextNonSpace(); token.typ {
	case itemError:
		p.errorf("%s", token.val)
	case itemIdentifier:
		if !p.hasFunction(token.val) {
			p.errorf("function %q not defined", token.val)
		}
		return NewIdentifier(token.val).SetPos(token.pos)
	case itemDot:
		return newDot(token.pos)
	case itemNil:
		return newNil(token.pos)
	case itemVariable:
		return p.useVar(token.pos, token.val)
	case itemField:
		return newField(token.pos, token.val)
	case itemBool:
		return newBool(token.pos, token.val == "true")
	case itemCharConstant, itemComplex, itemNumber:
		number, err := newNumber(token.pos, token.val, token.typ)
		if err != nil {
			p.error(err)
		}
		return number
	case itemLeftParen:
		pipe := p.pipeline("parenthesized pipeline")
		if token := p.next(); token.typ != itemRightParen {
			p.errorf("unclosed right paren: unexpected %s", token)
		}
		return pipe
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		return newString(token.pos, token.val, s)
	}
	p.backup()
	return nil
}

// hasFunction reports if a function name exists in the Tree's maps.
func (p *parser) hasFunction(name string) bool {
	for _, funcMap := range p.funcs {
		if funcMap == nil {
			continue
		}
		if funcMap[name] != nil {
			return true
		}
	}
	return false
}

// popVars trims the variable list to the specified length
func (p *parser) popVars(n int) {
	p.vars = p.vars[:n]
}

// useVar returns a node for a variable reference. It errors if the
// variable is not defined.
func (p *parser) useVar(pos Pos, name string) Node {
	v := newVariable(pos, name)
	for _, varName := range p.vars {
		if varName == v.Ident[0] {
			return v
		}
	}
	p.errorf("undefined variable %q", v.Ident[0])
	return nil
}
