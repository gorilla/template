// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"strings"
	"testing"
)

type slotTest struct {
	name   string
	input  string
	ok     bool
	result string
}

func TestSlot(t *testing.T) {
	// Some deep inheritance.
	tpl1 := `
	{{define "tpl1"}}
		A
		{{slot "header"}}
			-h1-
		{{end}}
		B
		{{slot "footer"}}
			-f1-
		{{end}}
		C
	{{end}}

	{{define "tpl2" "tpl1"}}
		xxx
	{{end}}

	{{define "tpl3" "tpl2"}}
		xxx
		{{fill "header"}}
			-h3-
		{{end}}
		xxx
	{{end}}

	{{define "tpl4" "tpl3"}}
		xxx
		{{fill "header"}}
			-h4-
		{{end}}
		xxx
		{{fill "footer"}}
			-f4-
		{{end}}
		xxx
	{{end}}

	{{define "tpl5" "tpl4"}}
		xxx
		{{fill "footer"}}
			-f5-
		{{end}}
		xxx
	{{end}}`
	// Recursive inheritance.
	tpl2 := `
	{{define "tpl1" "tpl2"}}
		{{fill "header"}}
			-h1-
		{{end}}
	{{end}}

	{{define "tpl2" "tpl3"}}
		{{fill "header"}}
			-h2-
		{{end}}
	{{end}}

	{{define "tpl3" "tpl1"}}
		{{fill "header"}}
			-h3-
		{{end}}
	{{end}}`

	tests := []slotTest{
		// the base template itself
		{"tpl1", tpl1, true, "A-h1-B-f1-C"},
		// default slot value
		{"tpl2", tpl1, true, "A-h1-B-f1-C"},
		// override only one slot
		{"tpl3", tpl1, true, "A-h3-B-f1-C"},
		// override both slots
		{"tpl4", tpl1, true, "A-h4-B-f4-C"},
		// override only one slot, higher level override both
		{"tpl5", tpl1, true, "A-h4-B-f5-C"},
		// impossible recursion
		{"tpl1", tpl2, false, "impossible recursion"},
	}
	for _, test := range tests {
		set, err := new(Set).Parse(test.input)
		if err != nil {
			t.Errorf("%s: unexpected parse error: %s", test.name, err)
			continue
		}
		b := new(bytes.Buffer)
		err = set.Execute(b, test.name, nil)
		if test.ok {
			if err != nil {
				t.Errorf("%s: unexpected exec error: %s", test.name, err)
				continue
			}
			result := b.String()
			result = strings.Replace(result, " ", "", -1)
			result = strings.Replace(result, "\n", "", -1)
			result = strings.Replace(result, "\t", "", -1)
			if test.result != result {
				t.Errorf("%s: expected %q, got %q", test.name, test.result, result)
			}
		} else {
			if err == nil {
				t.Errorf("%s: expected exec error", test.name)
				continue
			}
			if !strings.Contains(err.Error(), test.result) {
				t.Errorf("%s: expected exec error %q, got %q", test.name, test.result, err.Error())
			}
		}
	}
}
