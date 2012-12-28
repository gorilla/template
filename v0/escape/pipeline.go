// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package escape

import (
	"github.com/gorilla/template/v0/parse"
)

// redundantFuncs[a][b] implies that FuncMap[b](FuncMap[a](x)) == FuncMap[a](x)
// for all x.
var redundantFuncs = map[string]map[string]bool{
	"html_template_commentescaper": {
		"html_template_attrescaper":    true,
		"html_template_nospaceescaper": true,
		"html_template_htmlescaper":    true,
	},
	"html_template_cssescaper": {
		"html_template_attrescaper": true,
	},
	"html_template_jsregexpescaper": {
		"html_template_attrescaper": true,
	},
	"html_template_jsstrescaper": {
		"html_template_attrescaper": true,
	},
	"html_template_urlescaper": {
		"html_template_urlnormalizer": true,
	},
}

// equivEscapers matches contextual escapers to equivalent template builtins.
var equivEscapers = map[string]string{
	"html_template_attrescaper":    "html",
	"html_template_htmlescaper":    "html",
	"html_template_nospaceescaper": "html",
	"html_template_rcdataescaper":  "html",
	"html_template_urlescaper":     "urlquery",
	"html_template_urlnormalizer":  "urlquery",
}

// ensurePipelineContains ensures that the pipeline has commands with
// the identifiers in s in order.
// If the pipeline already has some of the sanitizers, do not interfere.
// For example, if p is (.X | html) and s is ["escapeJSVal", "html"] then it
// has one matching, "html", and one to insert, "escapeJSVal", to produce
// (.X | escapeJSVal | html).
//
// TODO: should not be public but this is a hard one.
func ensurePipelineContains(p *parse.PipeNode, s []string) {
	if len(s) == 0 {
		return
	}
	n := len(p.Cmds)
	// Find the identifiers at the end of the command chain.
	idents := p.Cmds
	for i := n - 1; i >= 0; i-- {
		if cmd := p.Cmds[i]; len(cmd.Args) != 0 {
			if id, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
				if id.Ident == "noescape" {
					return
				}
				continue
			}
		}
		idents = p.Cmds[i+1:]
	}
	dups := 0
	for _, id := range idents {
		if escFnsEq(s[dups], (id.Args[0].(*parse.IdentifierNode)).Ident) {
			dups++
			if dups == len(s) {
				return
			}
		}
	}
	newCmds := make([]*parse.CommandNode, n-len(idents), n+len(s)-dups)
	copy(newCmds, p.Cmds)
	// Merge existing identifier commands with the sanitizers needed.
	for _, id := range idents {
		pos := id.Args[0].Position()
		i := indexOfStr((id.Args[0].(*parse.IdentifierNode)).Ident, s, escFnsEq)
		if i != -1 {
			for _, name := range s[:i] {
				newCmds = appendCmd(newCmds, newIdentCmd(name, pos))
			}
			s = s[i+1:]
		}
		newCmds = appendCmd(newCmds, id)
	}
	// Create any remaining sanitizers.
	for _, name := range s {
		newCmds = appendCmd(newCmds, newIdentCmd(name, p.Position()))
	}
	p.Cmds = newCmds
}

// escFnsEq returns whether the two escaping functions are equivalent.
func escFnsEq(a, b string) bool {
	if e := equivEscapers[a]; e != "" {
		a = e
	}
	if e := equivEscapers[b]; e != "" {
		b = e
	}
	return a == b
}

// indexOfStr is the first i such that eq(s, strs[i]) or -1 if s was not found.
func indexOfStr(s string, strs []string, eq func(a, b string) bool) int {
	for i, t := range strs {
		if eq(s, t) {
			return i
		}
	}
	return -1
}

// appendCmd appends the given command to the end of the command pipeline
// unless it is redundant with the last command.
func appendCmd(cmds []*parse.CommandNode, cmd *parse.CommandNode) []*parse.CommandNode {
	if n := len(cmds); n != 0 {
		last, ok := cmds[n-1].Args[0].(*parse.IdentifierNode)
		next, _ := cmd.Args[0].(*parse.IdentifierNode)
		if ok && redundantFuncs[last.Ident][next.Ident] {
			return cmds
		}
	}
	return append(cmds, cmd)
}

// newIdentCmd produces a command containing a single identifier node.
func newIdentCmd(identifier string, pos parse.Pos) *parse.CommandNode {
	return &parse.CommandNode{
		NodeType: parse.NodeCommand,
		Args:     []parse.Node{parse.NewIdentifier(identifier).SetPos(pos)},
	}
}
