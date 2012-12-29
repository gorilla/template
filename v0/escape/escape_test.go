// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package escape

import (
	"testing"

	"github.com/gorilla/template/v0/parse"
)

func TestEscapeText(t *testing.T) {
	tests := []struct {
		input  string
		output context
	}{
		{
			``,
			context{},
		},
		{
			`Hello, World!`,
			context{},
		},
		{
			// An orphaned "<" is OK.
			`I <3 Ponies!`,
			context{},
		},
		{
			`<a`,
			context{state: stateTag},
		},
		{
			`<a `,
			context{state: stateTag},
		},
		{
			`<a>`,
			context{state: stateText},
		},
		{
			`<a href`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a on`,
			context{state: stateAttrName, attr: attrScript},
		},
		{
			`<a href `,
			context{state: stateAfterName, attr: attrURL},
		},
		{
			`<a style  =  `,
			context{state: stateBeforeValue, attr: attrStyle},
		},
		{
			`<a href=`,
			context{state: stateBeforeValue, attr: attrURL},
		},
		{
			`<a href=x`,
			context{state: stateURL, delim: delimSpaceOrTagEnd, urlPart: urlPartPreQuery},
		},
		{
			`<a href=x `,
			context{state: stateTag},
		},
		{
			`<a href=>`,
			context{state: stateText},
		},
		{
			`<a href=x>`,
			context{state: stateText},
		},
		{
			`<a href ='`,
			context{state: stateURL, delim: delimSingleQuote},
		},
		{
			`<a href=''`,
			context{state: stateTag},
		},
		{
			`<a href= "`,
			context{state: stateURL, delim: delimDoubleQuote},
		},
		{
			`<a href=""`,
			context{state: stateTag},
		},
		{
			`<a title="`,
			context{state: stateAttr, delim: delimDoubleQuote},
		},
		{
			`<a HREF='http:`,
			context{state: stateURL, delim: delimSingleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a Href='/`,
			context{state: stateURL, delim: delimSingleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href='"`,
			context{state: stateURL, delim: delimSingleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href="'`,
			context{state: stateURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href='&apos;`,
			context{state: stateURL, delim: delimSingleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href="&quot;`,
			context{state: stateURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href="&#34;`,
			context{state: stateURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a href=&quot;`,
			context{state: stateURL, delim: delimSpaceOrTagEnd, urlPart: urlPartPreQuery},
		},
		{
			`<img alt="1">`,
			context{state: stateText},
		},
		{
			`<img alt="1>"`,
			context{state: stateTag},
		},
		{
			`<img alt="1>">`,
			context{state: stateText},
		},
		{
			`<input checked type="checkbox"`,
			context{state: stateTag},
		},
		{
			`<a onclick="`,
			context{state: stateJS, delim: delimDoubleQuote},
		},
		{
			`<a onclick="//foo`,
			context{state: stateJSLineCmt, delim: delimDoubleQuote},
		},
		{
			"<a onclick='//\n",
			context{state: stateJS, delim: delimSingleQuote},
		},
		{
			"<a onclick='//\r\n",
			context{state: stateJS, delim: delimSingleQuote},
		},
		{
			"<a onclick='//\u2028",
			context{state: stateJS, delim: delimSingleQuote},
		},
		{
			`<a onclick="/*`,
			context{state: stateJSBlockCmt, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/*/`,
			context{state: stateJSBlockCmt, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/**/`,
			context{state: stateJS, delim: delimDoubleQuote},
		},
		{
			`<a onkeypress="&quot;`,
			context{state: stateJSDqStr, delim: delimDoubleQuote},
		},
		{
			`<a onclick='&quot;foo&quot;`,
			context{state: stateJS, delim: delimSingleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<a onclick=&#39;foo&#39;`,
			context{state: stateJS, delim: delimSpaceOrTagEnd, jsCtx: jsCtxDivOp},
		},
		{
			`<a onclick=&#39;foo`,
			context{state: stateJSSqStr, delim: delimSpaceOrTagEnd},
		},
		{
			`<a onclick="&quot;foo'`,
			context{state: stateJSDqStr, delim: delimDoubleQuote},
		},
		{
			`<a onclick="'foo&quot;`,
			context{state: stateJSSqStr, delim: delimDoubleQuote},
		},
		{
			`<A ONCLICK="'`,
			context{state: stateJSSqStr, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/`,
			context{state: stateJSRegexp, delim: delimDoubleQuote},
		},
		{
			`<a onclick="'foo'`,
			context{state: stateJS, delim: delimDoubleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<a onclick="'foo\'`,
			context{state: stateJSSqStr, delim: delimDoubleQuote},
		},
		{
			`<a onclick="'foo\'`,
			context{state: stateJSSqStr, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/foo/`,
			context{state: stateJS, delim: delimDoubleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<script>/foo/ /=`,
			context{state: stateJS, element: elementScript},
		},
		{
			`<a onclick="1 /foo`,
			context{state: stateJS, delim: delimDoubleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<a onclick="1 /*c*/ /foo`,
			context{state: stateJS, delim: delimDoubleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<a onclick="/foo[/]`,
			context{state: stateJSRegexp, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/foo\/`,
			context{state: stateJSRegexp, delim: delimDoubleQuote},
		},
		{
			`<a onclick="/foo/`,
			context{state: stateJS, delim: delimDoubleQuote, jsCtx: jsCtxDivOp},
		},
		{
			`<input checked style="`,
			context{state: stateCSS, delim: delimDoubleQuote},
		},
		{
			`<a style="//`,
			context{state: stateCSSLineCmt, delim: delimDoubleQuote},
		},
		{
			`<a style="//</script>`,
			context{state: stateCSSLineCmt, delim: delimDoubleQuote},
		},
		{
			"<a style='//\n",
			context{state: stateCSS, delim: delimSingleQuote},
		},
		{
			"<a style='//\r",
			context{state: stateCSS, delim: delimSingleQuote},
		},
		{
			`<a style="/*`,
			context{state: stateCSSBlockCmt, delim: delimDoubleQuote},
		},
		{
			`<a style="/*/`,
			context{state: stateCSSBlockCmt, delim: delimDoubleQuote},
		},
		{
			`<a style="/**/`,
			context{state: stateCSS, delim: delimDoubleQuote},
		},
		{
			`<a style="background: '`,
			context{state: stateCSSSqStr, delim: delimDoubleQuote},
		},
		{
			`<a style="background: &quot;`,
			context{state: stateCSSDqStr, delim: delimDoubleQuote},
		},
		{
			`<a style="background: '/foo?img=`,
			context{state: stateCSSSqStr, delim: delimDoubleQuote, urlPart: urlPartQueryOrFrag},
		},
		{
			`<a style="background: '/`,
			context{state: stateCSSSqStr, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url(&#x22;/`,
			context{state: stateCSSDqURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url('/`,
			context{state: stateCSSSqURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url('/)`,
			context{state: stateCSSSqURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url('/ `,
			context{state: stateCSSSqURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url(/`,
			context{state: stateCSSURL, delim: delimDoubleQuote, urlPart: urlPartPreQuery},
		},
		{
			`<a style="background: url( `,
			context{state: stateCSSURL, delim: delimDoubleQuote},
		},
		{
			`<a style="background: url( /image?name=`,
			context{state: stateCSSURL, delim: delimDoubleQuote, urlPart: urlPartQueryOrFrag},
		},
		{
			`<a style="background: url(x)`,
			context{state: stateCSS, delim: delimDoubleQuote},
		},
		{
			`<a style="background: url('x'`,
			context{state: stateCSS, delim: delimDoubleQuote},
		},
		{
			`<a style="background: url( x `,
			context{state: stateCSS, delim: delimDoubleQuote},
		},
		{
			`<!-- foo`,
			context{state: stateHTMLCmt},
		},
		{
			`<!-->`,
			context{state: stateHTMLCmt},
		},
		{
			`<!--->`,
			context{state: stateHTMLCmt},
		},
		{
			`<!-- foo -->`,
			context{state: stateText},
		},
		{
			`<script`,
			context{state: stateTag, element: elementScript},
		},
		{
			`<script `,
			context{state: stateTag, element: elementScript},
		},
		{
			`<script src="foo.js" `,
			context{state: stateTag, element: elementScript},
		},
		{
			`<script src='foo.js' `,
			context{state: stateTag, element: elementScript},
		},
		{
			`<script type=text/javascript `,
			context{state: stateTag, element: elementScript},
		},
		{
			`<script>foo`,
			context{state: stateJS, jsCtx: jsCtxDivOp, element: elementScript},
		},
		{
			`<script>foo</script>`,
			context{state: stateText},
		},
		{
			`<script>foo</script><!--`,
			context{state: stateHTMLCmt},
		},
		{
			`<script>document.write("<p>foo</p>");`,
			context{state: stateJS, element: elementScript},
		},
		{
			`<script>document.write("<p>foo<\/script>");`,
			context{state: stateJS, element: elementScript},
		},
		{
			`<script>document.write("<script>alert(1)</script>");`,
			context{state: stateText},
		},
		{
			`<Script>`,
			context{state: stateJS, element: elementScript},
		},
		{
			`<SCRIPT>foo`,
			context{state: stateJS, jsCtx: jsCtxDivOp, element: elementScript},
		},
		{
			`<textarea>value`,
			context{state: stateRCDATA, element: elementTextarea},
		},
		{
			`<textarea>value</TEXTAREA>`,
			context{state: stateText},
		},
		{
			`<textarea name=html><b`,
			context{state: stateRCDATA, element: elementTextarea},
		},
		{
			`<title>value`,
			context{state: stateRCDATA, element: elementTitle},
		},
		{
			`<style>value`,
			context{state: stateCSS, element: elementStyle},
		},
		{
			`<a xlink:href`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a xmlns`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a xmlns:foo`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a xmlnsxyz`,
			context{state: stateAttrName},
		},
		{
			`<a data-url`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a data-iconUri`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a data-urlItem`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a g:`,
			context{state: stateAttrName},
		},
		{
			`<a g:url`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a g:iconUri`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a g:urlItem`,
			context{state: stateAttrName, attr: attrURL},
		},
		{
			`<a g:value`,
			context{state: stateAttrName},
		},
		{
			`<a svg:style='`,
			context{state: stateCSS, delim: delimSingleQuote},
		},
		{
			`<svg:font-face`,
			context{state: stateTag},
		},
		{
			`<svg:a svg:onclick="`,
			context{state: stateJS, delim: delimDoubleQuote},
		},
	}

	for _, test := range tests {
		b, e := []byte(test.input), newEscaper(nil)
		c := e.escapeText(context{}, &parse.TextNode{NodeType: parse.NodeText, Text: b})
		if !test.output.eq(c) {
			t.Errorf("input %q: want context\n\t%v\ngot\n\t%v", test.input, test.output, c)
			continue
		}
		if test.input != string(b) {
			t.Errorf("input %q: text node was modified: want %q got %q", test.input, test.input, b)
			continue
		}
	}
}
