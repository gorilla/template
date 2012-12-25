// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package escape

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// HTML escaping.

var (
	htmlQuot = []byte("&#34;") // shorter than "&quot;"
	htmlApos = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp  = []byte("&amp;")
	htmlLt   = []byte("&lt;")
	htmlGt   = []byte("&gt;")
)

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func HTMLEscape(w io.Writer, b []byte) {
	last := 0
	for i, c := range b {
		var html []byte
		switch c {
		case '"':
			html = htmlQuot
		case '\'':
			html = htmlApos
		case '&':
			html = htmlAmp
		case '<':
			html = htmlLt
		case '>':
			html = htmlGt
		default:
			continue
		}
		w.Write(b[last:i])
		w.Write(html)
		last = i + 1
	}
	w.Write(b[last:])
}

// HTMLEscapeString returns the escaped HTML equivalent of the plain text data s.
func HTMLEscapeString(s string) string {
	// Avoid allocation if we can.
	if strings.IndexAny(s, `'"&<>`) < 0 {
		return s
	}
	var b bytes.Buffer
	HTMLEscape(&b, []byte(s))
	return b.String()
}

// JavaScript escaping.

var (
	jsLowUni = []byte(`\u00`)
	hex      = []byte("0123456789ABCDEF")

	jsBackslash = []byte(`\\`)
	jsApos      = []byte(`\'`)
	jsQuot      = []byte(`\"`)
	jsLt        = []byte(`\x3C`)
	jsGt        = []byte(`\x3E`)
)

// JSEscape writes to w the escaped JavaScript equivalent of the plain text data b.
func JSEscape(w io.Writer, b []byte) {
	last := 0
	for i := 0; i < len(b); i++ {
		c := b[i]

		if !jsIsSpecial(rune(c)) {
			// fast path: nothing to do
			continue
		}
		w.Write(b[last:i])

		if c < utf8.RuneSelf {
			// Quotes, slashes and angle brackets get quoted.
			// Control characters get written as \u00XX.
			switch c {
			case '\\':
				w.Write(jsBackslash)
			case '\'':
				w.Write(jsApos)
			case '"':
				w.Write(jsQuot)
			case '<':
				w.Write(jsLt)
			case '>':
				w.Write(jsGt)
			default:
				w.Write(jsLowUni)
				t, b := c>>4, c&0x0f
				w.Write(hex[t : t+1])
				w.Write(hex[b : b+1])
			}
		} else {
			// Unicode rune.
			r, size := utf8.DecodeRune(b[i:])
			if unicode.IsPrint(r) {
				w.Write(b[i : i+size])
			} else {
				fmt.Fprintf(w, "\\u%04X", r)
			}
			i += size - 1
		}
		last = i + 1
	}
	w.Write(b[last:])
}

// JSEscapeString returns the escaped JavaScript equivalent of the plain text data s.
func JSEscapeString(s string) string {
	// Avoid allocation if we can.
	if strings.IndexFunc(s, jsIsSpecial) < 0 {
		return s
	}
	var b bytes.Buffer
	JSEscape(&b, []byte(s))
	return b.String()
}

func jsIsSpecial(r rune) bool {
	switch r {
	case '\\', '\'', '"', '<', '>':
		return true
	}
	return r < ' ' || utf8.RuneSelf <= r
}

// HTMLEscaper returns the escaped HTML equivalent of the textual
// representation of its arguments.
func HTMLEscaper(args ...interface{}) string {
	ok := false
	var s string
	if len(args) == 1 {
		s, ok = args[0].(string)
	}
	if !ok {
		s = fmt.Sprint(args...)
	}
	return HTMLEscapeString(s)
}

// JSEscaper returns the escaped JavaScript equivalent of the textual
// representation of its arguments.
func JSEscaper(args ...interface{}) string {
	ok := false
	var s string
	if len(args) == 1 {
		s, ok = args[0].(string)
	}
	if !ok {
		s = fmt.Sprint(args...)
	}
	return JSEscapeString(s)
}

// URLQueryEscaper returns the escaped value of the textual representation of
// its arguments in a form suitable for embedding in a URL query.
func URLQueryEscaper(args ...interface{}) string {
	s, ok := "", false
	if len(args) == 1 {
		s, ok = args[0].(string)
	}
	if !ok {
		s = fmt.Sprint(args...)
	}
	return url.QueryEscape(s)
}
