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
    - Path: repo://frontend/dist/augment.js
      Note: initMathTypeset + startup.promise race handling (commit 9224848)
    - Path: repo://frontend/dist/index.html
      Note: mathjax script tag order (commit 9224848)
    - Path: repo://pkg/renderer/renderer.go
      Note: |-
        root cause WithHardWraps found here
        removed WithHardWraps (commit b842010)
        Render() mathjax script block (commit 9438be0)
    - Path: repo://pkg/renderer/static/mathjax-config.js
      Note: MathJax config + MDSMathTypeset (commit edbf71f)
    - Path: repo://pkg/renderer/static/mathjax.min.js
      Note: vendored MathJax v3 tex-svg (commit edbf71f)
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

## Step 3: Phase 2 — vendor and embed MathJax

This step vendored the MathJax v3 `tex-svg` component into the repo and wired
it into Go's embed system, mirroring the existing `mermaid.min.js` pattern.
No page wiring yet (that is Phase 3).

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Commit (code):** edbf71f — "feat(renderer): vendor and embed MathJax v3 (tex-svg)"

### What I did
- `curl -sL -o pkg/renderer/static/mathjax.min.js
  https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-svg.js` (2.1 MB).
- Created `pkg/renderer/static/mathjax-config.js` (window.MathJax config +
  `MDSMathTypeset()` typeset function).
- Added `//go:embed` declarations and `MathjaxJS()` / `MathjaxConfigJS()`
  getters in `pkg/renderer/renderer.go`.
- Copied both files into `frontend/dist/` (same convention as mermaid.min.js,
  which is also a plain copy — `make frontend-css` only regenerates chroma.css
  and ui.css).
- `go build ./...` and `go test ./pkg/renderer/` pass.

### Why
Offline-capable single-binary viewer (N1): MathJax must be embedded, not
loaded from a CDN. tex-svg avoids font-file loading races (DR-3).

### What worked
- Exact reuse of the mermaid embed pattern; zero surprises.

### What didn't work
- N/A.

### What I learned
- `cmd/gen-chroma-css` only generates chroma.css/ui.css; the other dist
  assets (base.css, dark.css, mermaid.min.js, ...) are plain copies, so
  syncing MathJax to frontend/dist by cp matches the existing convention.

### What was tricky to build
- Ordering constraint: `mathjax-config.js` must load BEFORE the MathJax
  library because v3 freezes `window.MathJax` at load time. Encoded in both
  the file comment and the getter docstring.

### What warrants a second pair of eyes
- Bundle size: mathjax.min.js is 2.1 MB unminified-checked? (tex-svg.js as
  served is ~2.1 MB raw on disk). Confirm acceptable binary growth after
  `make build`.

### What should be done in the future
- Consider a stripped MathJax component build if binary size becomes a
  complaint.

### Code review instructions
- Start at the embed block in `pkg/renderer/renderer.go` (search `mathjax`).
- Validate: `go test ./pkg/renderer/ -count=1`.

### Technical details
- CDN source: https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-svg.js
- MD5 in repo; treat as vendored third-party code (do not hand-edit).

## Step 4: Phase 3 — desktop frontend wiring

This step wired MathJax into the desktop page chrome: script tags in
`index.html`, an `initMathTypeset()` hook in `augment.js` called from
`MDSAugmentPage()`, and math CSS for display-equation overflow. The wiring
was verified end-to-end in a real browser (Python http.server over
frontend/dist + Playwright), not just by inspection.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Commit (code):** 9224848 — "feat(frontend): wire MathJax into the desktop app"

### What I did
- `frontend/dist/index.html`: added `<script src="mathjax-config.js">` BEFORE
  `<script src="mathjax.min.js" async>` (config-before-library ordering).
- `frontend/dist/augment.js`: added `initMathTypeset()` with the
  `MathJax.startup.promise` fallback for the async-load race; called it from
  `MDSAugmentPage()`.
- Appended MathJax CSS to `pkg/renderer/static/base.css` (display equations
  scroll horizontally) and `dark.css` (currentColor inheritance), copied
  both to `frontend/dist/`.
- `make build` succeeds (binary builds with embedded assets).
- Browser verification: `python3 -m http.server 8901` in frontend/dist +
  Playwright: injected content with inline math, display math, a `<pre><code>`
  block containing `$NOTMATH`, and inline code `$y=1$`. Result:
  `mjxCount: 2`, `codeUntouched: true`, `inlineCodeUntouched: true`,
  `mathJaxLoaded: true`.

### Why
The Wails frontend swaps `#content.innerHTML` per file/reload; MathJax must
re-typeset after every swap, scoped to `#content`, idempotently.

### What worked
- Exact copy of the augment.js contract; browser test proved skipHtmlTags
  protects code blocks without any Go-side work.

### What didn't work
- Tried to trigger the real async race (swap before MathJax ready) by
  reloading and immediately swapping; MathJax loads too fast from localhost
  (`wasReadyAtSwap: true`), so the `startup.promise` branch was not exercised
  end-to-end — only by code review.

### What I learned
- MathJax v3 resolves `window.MathJax` config at library load, so the config
  script tag order is a hard constraint, not a convention.
- `pkill -f "http.server 8901"` killed the agent's own bash shell (the
  command string matched the pattern) — the Phase-3 commit initially never
  ran; had to re-run it after noticing `git log` was unchanged.

### What was tricky to build
- The startup race: `mathjax.min.js` is `async`, and `OnDomReady` opens the
  first file potentially before MathJax finishes loading. Solved by chaining
  the typeset through `MathJax.startup.promise` when `typesetPromise` is not
  yet present.

### What warrants a second pair of eyes
- The un-exercised `startup.promise` branch (see "What didn't work"). A
  slow-disk or throttled-CPU first launch is the scenario it guards.

### What should be done in the future
- Optionally add a Playwright e2e test with script-load throttling to cover
  the race branch.

### Code review instructions
- `frontend/dist/index.html` script order; `frontend/dist/augment.js`
  `initMathTypeset`.
- Validate: serve frontend/dist over http and check `mjx-container` appears
  after `MDSAugmentPage()`.

### Technical details
- Verified with Playwright evaluate: see "What I did" for the exact results.

## Step 5: Phase 4 — legacy full-page renderer

This step added MathJax to `renderer.Render()`, the full standalone HTML
assembler: the config script inline, a `/static/mathjax.min.js` library tag
(mirroring mermaid), and a `MathJax.startup.promise`-chained typeset. A test
pins the config-before-library ordering and verbatim math passthrough.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Commit (code):** 9438be0 — "feat(renderer): wire MathJax into the legacy full-page Render()"

### What I did
- Added `mathjaxScript` block in `Render()` after `mermaidScript`, and the
corresponding `%s` in the page template + arg list.
- Added `TestRenderIncludesMathJaxScripts` covering library reference, config
  presence, typeset call, ordering, and `$x^2$` passthrough.
- Full `go test ./...` green.

### Why
`Render()` is still compiled and is the assembler for any full-page consumer;
leaving it math-less would silently regress that path.

### What worked
- Reusing the exact mermaid script-block pattern.

### What didn't work
- First test run failed: `--- FAIL: TestRenderIncludesMathJaxScripts ...
  config@24149, lib@23701`. Root cause: `strings.Index(html,
  "mathjax.min.js")` matched the filename mentioned inside the config
  script's own header comment, not the `<script src>` tag. Fixed by matching
  the full `src="http://localhost:8080/static/mathjax.min.js"` tag.
- While fixing the template arg count (`fmt.Sprintf call needs 11 args but
  has 12`), I first added one `%s` too many (7 instead of 6 in the body
  block); caught immediately by the compiler and corrected.

### What I learned
- Test heuristics that search for a bare filename are fragile when the
  embedded assets' comments mention the same filename.

### What was tricky to build
- fmt.Sprintf positional alignment: the page template is a long list of
  `%s` placeholders; inserting an arg requires touching both the template
  and the arg list, and the compiler error only reports the count.

### What warrants a second pair of eyes
- The `/static/mathjax.min.js` URL is only servable if an HTTP daemon
  exists; pkg/server was previously deleted, so `Render()` consumers need
  their own /static route (same pre-existing caveat as mermaid).

### What should be done in the future
- If a server is reintroduced, register mathjax.min.js on /static like
  mermaid.min.js.

### Code review instructions
- `pkg/renderer/renderer.go`: search `mathjaxScript`.
- Validate: `go test ./pkg/renderer/ -run TestRenderIncludesMathJaxScripts -v`.
