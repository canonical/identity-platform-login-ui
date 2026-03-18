# Skill: compare-design

Use this skill to compare a running page in the app against its Figma design,
identify visual discrepancies, and point to the specific code to fix.

Invoke with:
```
Use the compare-design skill.
Figma frame: <paste Figma frame URL>
Page: <URL of the running page, e.g. http://localhost/ui/verification>
```

---

## When to Use This Skill

- Before opening a PR for any frontend change that has a Figma design reference
- During code review when a reviewer suspects a design mismatch
- When a PR comment says "this doesn't match the design"

## Prerequisites

- The `figma` MCP server must be configured with a valid Figma API key (see `.vscode/mcp.json`)
- The full dev stack must be running (see `setup-dev-env` prompt)
- You need a Figma frame URL pointing to the specific screen being implemented

---

## Workflow

### Step 1 — Fetch the Figma design data

Use the `figma` MCP server to retrieve the frame. Provide the Figma URL exactly as
copied from the browser address bar or via "Copy link to selection" in Figma.

The server returns simplified layout data: component names, dimensions, spacing,
colours (as hex), font sizes, font weights, and text content.

Focus on:
- **Colour values** — background, border, text, icon colours
- **Spacing** — padding, margins, gap between elements
- **Typography** — font size, weight, line height
- **Component state** — what an input looks like when empty, focused, errored

### Step 2 — Inspect the running page

Use the `playwright` MCP server to open the target page URL and:

1. Take a screenshot of the page at the relevant state
2. Inspect the DOM for the elements that correspond to Figma components
3. For each element, capture the computed CSS: `color`, `background-color`,
   `border-color`, `padding`, `margin`, `font-size`, `font-weight`

To inspect computed styles via Playwright:
```
Navigate to <URL>, then evaluate:
  window.getComputedStyle(document.querySelector('<selector>'))
```

### Step 3 — Compare and list discrepancies

Compare the Figma data from Step 1 against the computed CSS from Step 2.

For each discrepancy, record:

| Property    | Figma value | Actual value | Element / selector |
|-------------|-------------|--------------|-------------------|
| color       | `#000000`   | `#21ba45`    | `.p-form-validation__message` |
| padding     | `16px`      | `8px`        | `.verification-code-input` |

### Step 4 — Map to source files

For each discrepancy, identify where to fix it:

- **Vanilla Framework class incorrect** → wrong class applied in the `NodeInput*.tsx`
  component JSX (e.g. using `is-success` instead of leaving it unstyled)
- **Custom CSS override** → look in `ui/static/css/` or component-level `.scss` files
- **Component prop wrong** → `@canonical/react-components` component passed wrong `status`
  or `type` prop in the `NodeInput*.tsx`
- **Logic error** → validation state being applied before user interaction, check the
  component's state initialisation

### Step 5 — Report

Produce a concise report:
1. Screenshot showing the visual difference
2. Table of discrepancies (from Step 3)
3. File + line for each fix, with the exact change needed
4. Confirm: are there any states from the Figma design (hover, error, disabled, empty)
   that are not reachable in the current implementation?
