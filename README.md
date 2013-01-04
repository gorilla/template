gorilla/template
================

*Warning:* This is a work in progress, and the v0 API is subject of
changes.

Go templates are a great foundation but suffer from some design
flaws, and this is a pity. They could be really awesome.
gorilla/template is yet another attempt to fix them.

Goals
-----
- Better composition options.
- Improved HTML support.
- Client-side support.
- Auto-documentation.
- Simpler spec and API.

These goals will be achieved through several phases.

### Phase 1 - Dec.2012
The initial work makes templates self-contained, removing root
templates. This greatly simplifies internals, spec and API without
losing any functionality. Highlights:

- Removed New("name") API: templates are self-contained and must
  be named using {{define}}.
- Templates are always executed passing the template name, so there's
  no need for two Execute methods.
- The Set type replaces the Template type: a Set is a group
  of parsed templates.
- Parse tree: added DefineNode and Tree types.
- Node.String() methods result in templates that reproduce the
  original template exactly.

### Phase 2 - Dec.2012
In this step the contextual escaping mechanism from html/template
becomes part of the gorilla/template package, effectively making it
a combination of text/template and html/template. Highlights:

- Same functionality of text/template and html/template but:
  - 1119 less lines of code.
  - 33 less types, functions and methods.
- HTML contextual escaping is set explicitly calling Escape().
- Types to encapsulate safe strings are placed in the template/escape
  package: CSS, JS, JSStr, HTML, HTMLAttr.

### Phase 3 - Jan.2013
We finally add template inheritance and introduce two new actions:
{{block}} and {{fill}}. Highlights:

- A {{block}} defines placeholders in a base template, and a {{fill}}
  fills a placeholder in a parent template.
- An inherited template is defined passing the parent template name
  in the {{define}} action, as in {{define "child" "parent"}}.
- New templates can't be added to a Set after execution: the set
  is "compiled" and locked, as in html/template.
- Inheritance and the new actions are inlined, so contextual escaping
  and execution didn't require any changes.
