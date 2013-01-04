// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"strings"
	"testing"
)

type blockTest struct {
	name   string
	input  string
	tpl    string
	values interface{}
	ok     bool
	result string
}

func TestBlock(t *testing.T) {
	tests := []blockTest{
		{"default block value", `
		{{define "base"}}
			{{block "header"}}
				base header
			{{end}}
		{{end}}
		{{define "t" "base"}}
		{{end}}`, "t", nil, true, "base header"},

		{"override block value", `
		{{define "base"}}
			{{block "header"}}
				base header
			{{end}}
		{{end}}
		{{define "t" "base"}}
			{{fill "header"}}
				t header
			{{end}}
		{{end}}`, "t", nil, true, "t header"},

		{"override block value, deeper", `
		{{define "base"}}
			{{block "header"}}
				base header
			{{end}}
		{{end}}
		{{define "t" "base"}}
			{{fill "header"}}
				t header
			{{end}}
		{{end}}
		{{define "x" "t"}}
			{{fill "header"}}
				x header
			{{end}}
		{{end}}`, "x", nil, true, "x header"},
	}
	for _, test := range tests {
		set, err := new(Set).Parse(test.input)
		if test.ok && err != nil {
			t.Errorf("%s: unexpected parse error: %s", test.name, err)
			continue
		} else if !test.ok && err == nil {
			t.Errorf("%s: expected parse error", test.name)
			continue
		}
		b := new(bytes.Buffer)
		err = set.Execute(b, test.tpl, test.values)
		if err != nil {
			t.Errorf("%s: unexpected exec error: %s", test.name, err)
			continue
		}
		result := strings.TrimSpace(b.String())
		if test.result != result {
			t.Errorf("%s: expected %q, got %q", test.name, test.result, result)
			continue
		}
	}
}
