# Skill: write-pr-description

Use this skill to generate a PR description draft by reading your staged changes,
identifying affected pages, and capturing screenshots of their current state.

Invoke with: `Use the write-pr-description skill`

---

## Honest Limitations (read before using)

- **"After" state only.** This skill shows how things look *now*, not before your
  changes — you can't get a before/after comparison without checking out the base
  branch and running the app twice. If you need a true before/after, take a manual
  screenshot before you start coding.
- **Screenshots are local files.** GitHub PR descriptions require images to be attached
  or hosted. After this skill runs, you will need to drag the screenshot files into
  the GitHub PR description editor yourself.
- **Videos go stale.** If you record a video and then address review comments, the
  video in the description is wrong. Default to screenshots unless the change is
  inherently temporal (e.g. a countdown timer, an animation).

---

## When This Is Most Useful

- PR touches `ui/pages/*.tsx` or `ui/components/*.tsx` (visible UI changes)
- The linked issue includes a Figma design or a described user flow
- The change involves a new user interaction (form submission, redirect, error state)
- The PR is from a contributor who is not the regular reviewer — more context needed

---

## Workflow

### Step 1 — Read the diff

```bash
git diff main...HEAD --stat
git diff main...HEAD -- 'ui/**' 'pkg/**'
```

From the diff, identify:
- Which `ui/pages/*.tsx` files changed → these are the pages to screenshot
- Which `ui/components/*.tsx` files changed → note which pages use them
  (search `ui/pages/` for imports of the changed component)
- Which Go handler files changed (`pkg/*/handlers.go`) → note the API endpoint
  and whether the response shape changed

### Step 2 — Screenshot affected pages

For each affected page, use the `playwright` MCP to:
1. Navigate to the page (base URL: `http://localhost`)
2. Screenshot the **initial state** (page as it loads)
3. Screenshot any **interactive states** relevant to the change:
   - Empty form with a validation error visible
   - Success state after submit
   - The specific UI element that changed

Save screenshots with descriptive names:
```
verification-initial-state.png
verification-code-input-error-state.png
verification-resend-cooldown.png
```

If the change involves timing or animation (e.g. a countdown), record a short clip
using your OS screen recorder and attach it to the PR description manually.

> **Reminder**: screenshot files are saved locally. You must attach them to the
> GitHub PR description manually by dragging them into the text area.

### Step 3 — Generate the PR description

Using the diff (Step 1) and screenshots (Step 2), produce a PR description in this
structure:

```markdown
## What

<One paragraph: what user-visible change does this PR make? What was broken or
missing before, and what does it do now?>

## Why

<One paragraph: link to the issue or Jira ticket. Why is this change needed?>

## Changes

<Bullet list of the meaningful changes — not a file list, but what each change
*does*. Example: "Added 60-second cooldown on the resend code button to prevent
abuse", not "Modified NodeInputSubmit.tsx">

## Screenshots

<Embed screenshots here — drag files from Step 2 into this section>

| State | Screenshot |
|-------|-----------|
| ... | ... |

## Testing

<Steps for the reviewer to manually verify the change. Be specific: include the
exact URL, what to type, what to click, and what the expected result is.>

## Notes for reviewers

<Anything the reviewer should pay attention to: known limitations, deliberate
trade-offs, follow-up issues that are out of scope for this PR.>
```

### Step 4 — Self-check before posting

Before posting the PR, verify:
- [ ] Screenshots match what's described in the "Changes" section
- [ ] Testing steps are reproducible (someone else can follow them cold)
- [ ] If there's a Figma design: run the `compare-design` skill and include any
  discrepancies as known issues or confirm they are intentional
- [ ] No `console.log` left in the diff (`git diff main...HEAD | grep console.log`)
- [ ] Prettier / ESLint pass: `cd ui && npm run build`
