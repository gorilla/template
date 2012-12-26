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