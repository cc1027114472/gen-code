# Source Of Truth

Use this reference to decide what the prototype should mean, what it should include, and how to validate it.

## Core project files

Primary source directory:

- `F:\codex\artifacts\kb-audit`

Important documents inside that directory:

- the business-flow summary markdown file
- the site menu handbook markdown file
- the product module understanding markdown file
- the approval center design markdown file
- the role and permission boundary markdown file

Use the business-flow summary doc as the main source for node order.

## Planning and prototype-support docs

Also use these project docs when translating analysis into prototype structure:

- the Axure page-list markdown file
- the Axure 18-page wireframe markdown file
- the Axure build-order and master-components markdown file

## Existing prototype files

Current staged prototype directory:

- `F:\codex\artifacts\kb-audit\html-demo-prototype`

Primary files there:

- `index.html`
- `styles.css`
- `app.js`
- the usage-notes markdown file

Update this directory in place unless the user explicitly asks for a second variant.

## Chain-expansion order

Recommended order:

1. home and navigation
2. procurement and approval
3. labor
4. revenue-side
5. machinery

## Validation checklist

After each meaningful update:

1. JS passes syntax validation.
2. Menu entries, node labels, and actual pages are aligned.
3. The demoed chain text still matches the documented business chain.
4. Approval strong-binding rules remain visible where applicable.
5. The response clearly distinguishes:
   - fully interactive parts
   - context or support pages
   - future or lighter-depth nodes
