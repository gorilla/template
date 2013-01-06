// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/gorilla/template/v0/escape"
	"github.com/gorilla/template/v0/parse"
)

// Set stores a collection of parsed templates.
//
// To add templates call Set.Parse (or other parse methods):
//
//     set, err := new(Set).Parse(`{{define "hello"}}Hello, World.{{end}}`)
//     if err != nil {
//         // do something with the parsing error...
//     }
//
// To execute a template call Set.Execute passing an io.Writer, the name of
// the template to execute and related data:
//
//     err = set.Execute(os.Stderr, "hello", nil)
//     if err != nil {
//         // do something with the execution error...
//     }
type Set struct {
	mutex      sync.Mutex
	tree       parse.Tree
	leftDelim  string
	rightDelim string
	escape     bool // compilation flag to perform contextual escaping
	compiled   bool // compilation flag to lock the set after first execution
	// We use two maps, one for parsing and one for execution.
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

// init initializes the set fields to default values.
func (s *Set) init() {
	if s.tree == nil {
		s.tree = make(parse.Tree)
	}
	if s.execFuncs == nil {
		s.execFuncs = make(map[string]reflect.Value)
	}
	if s.parseFuncs == nil {
		s.parseFuncs = make(FuncMap)
	}
}

// Delims sets the action delimiters to the specified strings, to be used in
// subsequent calls to Parse. An empty delimiter stands for the corresponding
// default: "{{" or "}}".
// The return value is the set, so calls can be chained.
func (s *Set) Delims(left, right string) *Set {
	s.leftDelim = left
	s.rightDelim = right
	return s
}

// Funcs adds the elements of the argument map to the template's function map.
// It panics if a value in the map is not a function with appropriate return
// type. However, it is legal to overwrite elements of the map. The return
// value is the set, so calls can be chained.
func (s *Set) Funcs(funcMap FuncMap) *Set {
	s.init()
	addValueFuncs(s.execFuncs, funcMap)
	addFuncs(s.parseFuncs, funcMap)
	return s
}

// Escape turns on contextual escaping in all templates in the set, rewriting
// them to guarantee that the output is safe. The return value is the set,
// so calls can be chained.
func (s *Set) Escape() *Set {
	s.escape = true
	return s
}

// Clone returns a duplicate of the template, including all associated
// templates. The actual representation is not copied, but the name space of
// associated templates is, so further calls to Parse in the copy will add
// templates to the copy but not to the original. Clone can be used to prepare
// common templates and use them with variant definitions for other templates
// by adding the variants after the clone is made.
func (s *Set) Clone() (*Set, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	ns := new(Set).Delims(s.leftDelim, s.rightDelim)
	ns.init()
	for k, v := range s.parseFuncs {
		ns.parseFuncs[k] = v
	}
	for k, v := range s.execFuncs {
		ns.execFuncs[k] = v
	}
	err := ns.tree.AddTree(s.tree.Copy())
	if err != nil {
		return nil, err
	}
	ns.escape = s.escape
	ns.compiled = s.compiled
	return ns, nil
}

// Compile performs inlining and contextual escaping in all templates in the
// set. This doesn't need to be called manually because the set is compiled
// automatically when executed, but it can be used to force compilation and
// catch errors earlier.
func (s *Set) Compile() (*Set, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.compiled {
		// Inlining.
		if err := inlineTree(s.tree); err != nil {
			return nil, err
		}
		// Contextual escaping.
		if s.escape {
			if err := escape.EscapeTree(s.tree); err != nil {
				return nil, err
			}
			s.Funcs(escape.FuncMap)
		}
		s.compiled = true
	}
	return s, nil
}

// Parse ----------------------------------------------------------------------

// parse parses the given text and adds the resulting templates to the set.
// The name is only used for debugging purposes: when parsing files or glob,
// it can show which file caused an error.
//
// Parsing templates after the set executed results in an error.
func (s *Set) parse(text, name string) (*Set, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.compiled {
		return nil, fmt.Errorf(
			"template: new templates can't be added after execution")
	}
	s.init()
	if tree, err := parse.Parse(name, text, s.leftDelim, s.rightDelim,
		builtins, s.parseFuncs); err != nil {
		return nil, err
	} else if err = s.tree.AddTree(tree); err != nil {
		return nil, err
	}
	return s, nil
}

// Parse parses the given text and adds the resulting templates to the set.
// If an error occurs, parsing stops and the returned set is nil; otherwise
// it is s.
func (s *Set) Parse(text string) (*Set, error) {
	return s.parse(text, "template string")
}

// ParseFiles parses the named files and adds the resulting templates to the
// set. There must be at least one file. If an error occurs, parsing stops and
// the returned set is nil; otherwise it is s.
func (s *Set) ParseFiles(filenames ...string) (*Set, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf(
			"template: ParseFiles must be called with at least one filename")
	}
	for _, filename := range filenames {
		if b, err := ioutil.ReadFile(filename); err != nil {
			return nil, err
		} else if _, err = s.parse(string(b), filename); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ParseGlob parses the template definitions in the files identified by the
// pattern and adds the resulting templates to the set. The pattern is
// processed by filepath.Glob and must match at least one file. ParseGlob is
// equivalent to calling s.ParseFiles with the list of files matched by the
// pattern. If an error occurs, parsing stops and the returned set is nil;
// otherwise it is s.
func (s *Set) ParseGlob(pattern string) (*Set, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf(
			"template: pattern doesn't match any files: %#q", pattern)
	}
	return s.ParseFiles(filenames...)
}

// Convenience parsing wrappers -----------------------------------------------

// Must is a helper that wraps a call to a function that returns (*Set, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as:
//
//     var set = Must(new(Set).Parse(`{{define "hello"}}Hello, World.{{end}}`))
func Must(s *Set, err error) *Set {
	if err != nil {
		panic(err)
	}
	return s
}

// This redundant API is probably not needed...

// Parse creates a new Set with the template definitions from the given text.
// If an error occurs, parsing stops and the returned set is nil.
//func Parse(text string) (*Set, error) {
//	return new(Set).Parse(text)
//}

// ParseFiles creates a new Set with the template definitions from the named
// files. There must be at least one file. If an error occurs, parsing stops
// and the returned set is nil.
//func ParseFiles(filenames ...string) (*Set, error) {
//	return new(Set).ParseFiles(filenames...)
//}

// ParseGlob creates a new Set with the template definitions from the
// files identified by the pattern. The pattern is processed by filepath.Glob
// and must match at least one file. ParseGlob is equivalent to calling
// ParseFiles with the list of files matched by the pattern. If an error
// occurs, parsing stops and the returned set is nil.
//func ParseGlob(pattern string) (*Set, error) {
//	return new(Set).ParseGlob(pattern)
//}
