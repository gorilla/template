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
