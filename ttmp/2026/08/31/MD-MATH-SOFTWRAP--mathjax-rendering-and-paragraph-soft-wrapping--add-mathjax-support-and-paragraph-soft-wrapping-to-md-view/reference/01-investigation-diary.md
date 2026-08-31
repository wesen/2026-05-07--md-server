---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/2026-08-31--bayesian-marketing/artifacts/report-day1.md
      Note: example math corpus motivating the ticket
    - Path: repo://pkg/renderer/renderer.go
      Note: |-
        root cause WithHardWraps found here
        removed WithHardWraps (commit b842010)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Investigate and design MathJax math rendering plus paragraph soft-wrapping
for md-view, producing an intern-ready design/implementation guide.

## Step 1: Evidence gathering and design authoring

The repository was explored to map the two render paths (Wails desktop via
`App.openPath()` → `renderer.RenderBody()` → `innerHTML` swap + `MDSAugmentPage()`,
and the legacy full-page `renderer.Render()`), the embedded-static-asset
pattern used for mermaid, and the exact location of the hard-wrap behavior.
The motivating math corpus (report-day1.md) was inspected to confirm
`$...$`/`$$...$$` delimiters and statistics-flavored LaTeX commands. The
design doc was then written with phased implementation, decision records,
tests, and risk analysis.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add mathjax support to md-view. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

One file for example is /home/manuel/code/wesen/2026-08-31--bayesian-marketing/artifacts/report-day1.md

The other is that we should still treat newline separate lines in a paragraph as a paragraph and not do jagged edges. This doesn't look great on mobile."

**Assistant interpretation:** Create a docmgr ticket and a comprehensive,
intern-oriented design/implementation guide for two md-view changes:
MathJax math rendering (example: the bayesian-marketing Day 1 report) and
soft-wrapping paragraph newlines instead of hard breaks. Then upload to
reMarkable.

**Inferred user intent:** md-view should render LaTeX math in statistics
reports correctly and reflow paragraphs nicely on narrow/mobile windows.

### What I did
- Mapped render pipeline: `app.go` (`openPath`), `pkg/renderer/renderer.go`
  (`RenderBody`, `Render`, embeds), `frontend/dist/{index.html,app.js,augment.js}`.
- Confirmed no existing math support (no MathJax/KaTeX in Go or dist JS).
- Located root cause of jagged paragraphs: `html.WithHardWraps()` in
  `RenderBody()` goldmark options.
- Inspected example corpus for delimiter style (`$...$`, `$$...$$`,
  `\operatorname`, display derivations).
- Created ticket `MD-MATH-SOFTWRAP--mathjax-rendering-and-paragraph-soft-wrapping`,
  wrote design doc and this diary.

### Why
User wants intern-ready documentation before implementation and delivery to
reMarkable for reading.

### What worked
- The mermaid integration is an exact template for MathJax (embed + index.html
  script + augment.js re-runnable hook), which made the design concrete.
- The MathJax `skipHtmlTags` config removes the need for any Go-side math
  parsing (code blocks are automatically excluded).

### What didn't work
- N/A (no failures; grep for "mathjax" initially matched KaTeX code bundled
  inside mermaid.min.js — noted and ignored).

### What I learned
- md-view has two HTML paths (Wails fragment swap vs legacy full-page Render);
  both need MathJax wiring.
- `html.WithHardWraps()` is GitHub-comment-style, not GitHub-render-style;
  CommonMark soft break is the correct default for a document viewer.

### What was tricky to build
- N/A (documentation step). The trickiest design point identified for the
  implementer: the race between the async MathJax script load and the first
  `MDSAugmentPage()` call on startup — solved via `MathJax.startup.promise`.

### What warrants a second pair of eyes
- DR-3 (tex-svg vs tex-chtml) and DR-4 (no hard-wrap escape-hatch flag) are
  judgment calls.
- The claim that goldmark still emits `<br>` for two-trailing-space breaks
  without `WithHardWraps` should be verified by the Phase 1 test (Test B).

### What should be done in the future
- Implement Phases 1–5 of the design doc.

### Code review instructions
- Read the design doc in `design-doc/01-...md`, sections 3 and 5 first.
- Validate evidence: `grep -n "WithHardWraps" pkg/renderer/renderer.go`,
  `grep -n "MDSAugmentPage" frontend/dist/augment.js`.

### Technical details
- Ticket path: `ttmp/2026/08/31/MD-MATH-SOFTWRAP--mathjax-rendering-and-paragraph-soft-wrapping--add-mathjax-support-and-paragraph-soft-wrapping-to-md-view`

## Step 2: Phase 1 — paragraph soft-wrapping

This step implemented the Phase 1 of the design doc: goldmark's
`html.WithHardWraps()` was removed from `RenderBody()` so that a single
newline inside a paragraph renders as a space (soft wrap), and three tests
were added to pin the behavior (soft newline, two-space hard break, blank
line splitting). All tests pass.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the five phases of the design doc,
committing at each phase boundary and printing thermal work slips before and
after each phase.

**Inferred user intent:** The design should actually be implemented, with a
traceable commit history and printed artifacts on the thermal printer.

**Commit (code):** b842010 — "feat(renderer): soft-wrap paragraphs, drop WithHardWraps"

### What I did
- Removed `html.WithHardWraps()` from `goldmark.WithRendererOptions` in
  `pkg/renderer/renderer.go` `RenderBody()`, with an explanatory comment.
- Added `TestParagraphSoftWrap`, `TestHardBreakStillWorks`,
  `TestBlankLineSplitsParagraphs` to `pkg/renderer/renderer_test.go`.
- Ran `gofmt -l pkg/renderer/` (clean) and `go test ./... -count=1` (all ok).

### Why
With hard wraps, 80-column-authored paragraphs render as staircase lines on
narrow/mobile viewports. CommonMark soft break is the correct default for a
viewer (DR-4 in the design doc).

### What worked
- Single-line removal; goldmark's parser already handles two-space hard
  breaks independently of the renderer option, so the escape hatch kept
  working with zero extra code.

### What didn't work
- N/A (tests passed first try).

### What I learned
- goldmark's `WithHardWraps` is a *renderer* option; hard-break detection via
  trailing spaces is a *parser* concern, so removing the renderer option does
  not affect explicit breaks.

### What was tricky to build
- Nothing in this phase. The only design risk (test B assumption from the
  design doc) was confirmed empirically by `TestHardBreakStillWorks`.

### What warrants a second pair of eyes
- Any markdown corpus that relied on implicit hard breaks for layout (e.g.
  ASCII poetry) will visually change. Documented as a behavior change.

### What should be done in the future
- Mention the behavior change in README/docs (Phase 5).

### Code review instructions
- Start at `pkg/renderer/renderer.go`, `RenderBody()` goldmark options.
- Validate: `go test ./pkg/renderer/ -run 'TestParagraph|TestHardBreak|TestBlankLine' -count=1 -v`.

### Technical details
- Behavior change: `"a\nb"` now renders `<p>a b</p>` (previously `<p>a<br />\nb</p>`).
