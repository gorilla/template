// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"

	"github.com/gorilla/template/v0/parse"
)

// parentList returns the list of parent templates for a given template name.
// It returns an error if a template is not found or recursive dependency
// is detected.
func parentList(tree parse.Tree, name string) (parents []string, err error) {
	for {
		define := tree[name]
		if define == nil {
			return nil, fmt.Errorf("template: template not found: %q", name)
		}
		for _, v := range parents {
			if v == name {
				parents = append(parents, name)
				return nil, fmt.Errorf("template: impossible recursion: %#v",
					parents)
			}
		}
		parents = append(parents, name)
		if define.Parent == "" {
			break
		}
		name = define.Parent
	}
	return
}

// compilationOrder returns the order in which templates must be compiled in a
// set. Templates with more dependencies are compiled first; those without a
// parent are compiled only after all its dependents were compiled.
func compilationOrder(tree parse.Tree) ([]string, error) {
	var parents [][]string
	for name, _ := range tree {
		p, err := parentList(tree, name)
		if err != nil {
			return nil, err
		}
		parents = append(parents, p)
	}
	order := make([]string, len(parents))
	for len(parents) > 0 {
		i := 0
		for i < len(parents) {
			if len(parents[i]) == 1 {
				name := parents[i][0]
				order[len(parents)-1] = name
				parents = append(parents[:i], parents[i+1:]...)
				for k, v := range parents {
					var s []string
					for _, v2 := range v {
						if v2 != name {
							s = append(s, v2)
						}
					}
					parents[k] = s
				}
			} else {
				i++
			}
		}
	}
	return order, nil
}

// inlineTree expands all {{define}} actions from a tree.
func inlineTree(tree parse.Tree) error {
	order, err := compilationOrder(tree)
	if err != nil {
		return err
	}
	for _, name := range order {
		if err := inlineDefine(tree, name); err != nil {
			return err
		}
	}
	return nil
}

// inlineDefine expands a {{define}} action.
func inlineDefine(tree parse.Tree, name string) error {
	// TODO: expand {{template}} before anything else?
	return inlineExtendedDefine(tree, name)
}

// inlineSimpleDefine expands a parentless {{define}} action.
func inlineSimpleDefine(tree parse.Tree, name string) error {
	// Expand {{block}}, remove {{fill}}.
	cleanupBlock(tree[name].List)
	return nil
}

// inlineExtendedDefine expands an extended {{define}} action.
func inlineExtendedDefine(tree parse.Tree, name string) error {
	define := tree[name]
	parent := tree[define.Parent]
	if define.Parent == "" {
		return inlineSimpleDefine(tree, name)
	} else if parent == nil {
		return fmt.Errorf("template: define extends undefined parent %q",
			define.Parent)
	}
	// Get all FillNode's from current define.
	fillers := map[string]*parse.FillNode{}
	unused := map[string]bool{}
	for _, n := range define.List.Nodes {
		if f, ok := n.(*parse.FillNode); ok {
			fillers[f.Name] = f
			unused[f.Name] = true
		}
	}
	// Update nodes and parent.
	// TODO: must review debugging system because updating like this will
	// report wrong positions and context.
	define.List = parent.List.CopyList()
	define.Parent = parent.Parent
	// Replace FillNode's and BlockNode's from parent.
	applyFillers(define.List, fillers, unused)
	// Add extra fillers.
	for k, v := range unused {
		if v {
			define.List.Nodes = append(define.List.Nodes, fillers[k].CopyFill())
		}
	}
	// Do it again until parent is empty.
	return inlineExtendedDefine(tree, name)
}

// applyFillers replaces block and fill nodes by their filler counterparts.
func applyFillers(n parse.Node, fillers map[string]*parse.FillNode, unused map[string]bool) {
	switch n := n.(type) {
	case *parse.IfNode:
		applyFillers(n.List, fillers, unused)
		applyFillers(n.ElseList, fillers, unused)
	case *parse.ListNode:
		for k, v := range n.Nodes {
			switch v := v.(type) {
			case *parse.BlockNode:
				// Replace the block by the list of nodes from the filler.
				if filler := fillers[v.Name]; filler != nil {
					n.Nodes[k] = filler.List.CopyList()
				}
			case *parse.FillNode:
				// Replace the fill by the new filler.
				if filler := fillers[v.Name]; filler != nil {
					n.Nodes[k] = filler.CopyFill()
					unused[v.Name] = false
				}
			default:
				applyFillers(v, fillers, unused)
			}
		}
	case *parse.RangeNode:
		applyFillers(n.List, fillers, unused)
		applyFillers(n.ElseList, fillers, unused)
	case *parse.WithNode:
		applyFillers(n.List, fillers, unused)
		applyFillers(n.ElseList, fillers, unused)
	}
}

// cleanupBlock removes block and fill nodes.
//
// May contain child actions:
// BlockNode:  n.List
// DefineNode: n.List
// FillNode:   n.List
// IfNode:     n.List, n.ElseList
// ListNode:   n.Nodes
// RangeNode:  n.List, n.ElseList
// WithNode:   n.List, n.ElseList
func cleanupBlock(n parse.Node) {
	switch n := n.(type) {
	case *parse.IfNode:
		cleanupBlock(n.List)
		cleanupBlock(n.ElseList)
	case *parse.ListNode:
		k := 0
		for k < len(n.Nodes) {
			v := n.Nodes[k]
			switch v := v.(type) {
			case *parse.BlockNode:
				// Replace the block by its list of nodes.
				n.Nodes[k] = v.List
				continue
			case *parse.FillNode:
				// Remove the filler.
				n.Nodes = append(n.Nodes[:k], n.Nodes[k+1:]...)
				continue
			default:
				cleanupBlock(v)
			}
			k++
		}
	case *parse.RangeNode:
		cleanupBlock(n.List)
		cleanupBlock(n.ElseList)
	case *parse.WithNode:
		cleanupBlock(n.List)
		cleanupBlock(n.ElseList)
	}
}
