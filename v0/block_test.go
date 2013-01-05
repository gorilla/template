// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"strings"
	"testing"
)

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

	tests := []execTest{
		// the base template itself
		{"tpl1", tpl1, "A-h1-B-f1-C", nil, true},
		// default slot value
		{"tpl2", tpl1, "A-h1-B-f1-C", nil, true},
		// override only one slot
		{"tpl3", tpl1, "A-h3-B-f1-C", nil, true},
		// override both slots
		{"tpl4", tpl1, "A-h4-B-f4-C", nil, true},
		// override only one slot, higher level override both
		{"tpl5", tpl1, "A-h4-B-f5-C", nil, true},
		// impossible recursion
		{"tpl1", tpl2, "impossible recursion", nil, false},
	}
	for _, test := range tests {
		set, err := new(Set).Parse(test.input)
		if err != nil {
			t.Errorf("%s: unexpected parse error: %s", test.name, err)
			continue
		}
		b := new(bytes.Buffer)
		err = set.Execute(b, test.name, test.data)
		if test.ok {
			if err != nil {
				t.Errorf("%s: unexpected exec error: %s", test.name, err)
				continue
			}
			output := b.String()
			output = strings.Replace(output, " ", "", -1)
			output = strings.Replace(output, "\n", "", -1)
			output = strings.Replace(output, "\t", "", -1)
			if test.output != output {
				t.Errorf("%s: expected %q, got %q", test.name, test.output, output)
			}
		} else {
			if err == nil {
				t.Errorf("%s: expected exec error", test.name)
				continue
			}
			if !strings.Contains(err.Error(), test.output) {
				t.Errorf("%s: expected exec error %q, got %q", test.name, test.output, err.Error())
			}
		}
	}
}
