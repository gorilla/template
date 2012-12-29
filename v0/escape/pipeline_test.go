// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package escape

import (
	"fmt"
	"testing"

	"github.com/gorilla/template/v0/parse"
)

func TestensurePipelineContains(t *testing.T) {
	tests := []struct {
		input, output string
		ids           []string
	}{
		{
			"{{.X}}",
			".X",
			[]string{},
		},
		{
			"{{.X | html}}",
			".X | html",
			[]string{},
		},
		{
			"{{.X}}",
			".X | html",
			[]string{"html"},
		},
		{
			"{{.X | html}}",
			".X | html | urlquery",
			[]string{"urlquery"},
		},
		{
			"{{.X | html | urlquery}}",
			".X | html | urlquery",
			[]string{"urlquery"},
		},
		{
			"{{.X | html | urlquery}}",
			".X | html | urlquery",
			[]string{"html", "urlquery"},
		},
		{
			"{{.X | html | urlquery}}",
			".X | html | urlquery",
			[]string{"html"},
		},
		{
			"{{.X | urlquery}}",
			".X | html | urlquery",
			[]string{"html", "urlquery"},
		},
		{
			"{{.X | html | print}}",
			".X | urlquery | html | print",
			[]string{"urlquery", "html"},
		},
		{
			"{{($).X | html | print}}",
			"($).X | urlquery | html | print",
			[]string{"urlquery", "html"},
		},
	}
	for i, test := range tests {
		text := fmt.Sprintf(`{{define "t"}}%s{{end}}`, test.input)
		// fake funcs just for the test.
		funcs := map[string]interface{}{
			"html":     true,
			"print":    true,
			"urlquery": true,
		}
		tree, err := parse.Parse("", text, "", "", funcs, FuncMap)
		if err != nil {
			t.Errorf("#%d: parsing error: %v", i, err)
			continue
		}
		action, ok := (tree["t"].List.Nodes[0].(*parse.ActionNode))
		if !ok {
			t.Errorf("#%d: First node is not an action: %s", i, text)
			continue
		}
		pipe := action.Pipe
		ensurePipelineContains(pipe, test.ids)
		got := pipe.String()
		if got != test.output {
			t.Errorf("#%d: %s, %v: want\n\t%s\ngot\n\t%s", i, text, test.ids, test.output, got)
		}
	}
}

func TestRedundantFuncs(t *testing.T) {
	inputs := []interface{}{
		"\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f" +
			"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
			` !"#$%&'()*+,-./` +
			`0123456789:;<=>?` +
			`@ABCDEFGHIJKLMNO` +
			`PQRSTUVWXYZ[\]^_` +
			"`abcdefghijklmno" +
			"pqrstuvwxyz{|}~\x7f" +
			"\u00A0\u0100\u2028\u2029\ufeff\ufdec\ufffd\uffff\U0001D11E" +
			"&amp;%22\\",
		CSS(`a[href =~ "//example.com"]#foo`),
		HTML(`Hello, <b>World</b> &amp;tc!`),
		HTMLAttr(` dir="ltr"`),
		JS(`c && alert("Hello, World!");`),
		JSStr(`Hello, World & O'Reilly\x21`),
		URL(`greeting=H%69&addressee=(World)`),
	}

	for n0, m := range redundantFuncs {
		f0 := FuncMap[n0].(func(...interface{}) string)
		for n1 := range m {
			f1 := FuncMap[n1].(func(...interface{}) string)
			for _, input := range inputs {
				want := f0(input)
				if got := f1(want); want != got {
					t.Errorf("%s %s with %T %q: want\n\t%q,\ngot\n\t%q", n0, n1, input, input, want, got)
				}
			}
		}
	}
}
