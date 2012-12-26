// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/template is a template engine to generate textual or HTML
output.

It is based on the standard text/template and html/template packages.

Here's the simplest example to render a template:

	set, err := new(Set).Parse(`{{define "hello"}}Hello, World.{{end}}`)
	if err != nil {
		// do something with the parsing error...
	}
	err = set.Execute(os.Stderr, "hello", nil)
	if err != nil {
		// do something with the execution error...
	}

First we create a Set, which stores a collection of templates. Then we call
Parse() to parse a string and add the templates defined there to the set.
And finally we call Execute() to render the template "hello" using the given
data (in this case, nil), and write the output to an io.Writer (in this case,
os.Stderr).

Parse() can be called multiple times to fill the set with as many template
definitions as needed. There are also ParseFiles() and ParseGlob() methods
to read and parse the contents from the specified files.
*/
package template
