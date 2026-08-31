---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://app.go
      Note: App.openPath render entry and Wails event wiring
    - Path: repo://frontend/dist/augment.js
      Note: MDSAugmentPage re-runnable augmentation contract to extend with MathJax typesetting
    - Path: repo://frontend/dist/index.html
      Note: page chrome where MathJax script tags go
    - Path: repo://pkg/renderer/renderer.go
      Note: goldmark setup with WithHardWraps, embed pattern, RenderBody/Render to modify
    - Path: repo://pkg/renderer/static/mermaid.min.js
      Note: existing vendored-library pattern MathJax must mirror
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# MathJax Support and Paragraph Soft-Wrapping in md-view

## Executive Summary

md-view is a desktop Markdown viewer (Wails v2 Go app) that renders Markdown
files into a WebView. Today it handles GitHub-Flavored Markdown, code
highlighting (Chroma), Mermaid diagrams, and frontmatter — but **not math**.
Two changes are requested:

1. **MathJax support.** Reports such as
   `/home/manuel/code/wesen/2026-08-31--bayesian-marketing/artifacts/report-day1.md`
   use inline LaTeX math (`$\nu_i \sim N(-0.15, 1.0)$`) and display math
   (`$$...$$`). These currently render as raw `$...$` text. We will integrate
   MathJax v3 so both inline and block math render correctly in the desktop
   app and (where applicable) the legacy full-page renderer.

2. **Paragraph soft-wrapping.** The renderer currently enables goldmark's
   `html.WithHardWraps()`, which turns every newline inside a paragraph into a
   `<br>`. Markdown files written at ~80 columns therefore show jagged,
   staircase-shaped paragraphs on mobile and narrow windows. We want a single
   newline inside a paragraph to be treated as a **space** (standard
   CommonMark behavior), so paragraphs reflow to the viewport width.

This document explains, for an intern with no prior context: what md-view is,
how a Markdown file travels from disk to pixels, every component involved,
why each design decision is made, and a step-by-step implementation plan with
pseudocode, diagrams, API references, and file references.

---

## 1. Problem Statement and Scope

### 1.1 Problem A: No math rendering

The example report (Bayesian marketing Day 1 chapter) is written in
GitHub-style Markdown with inline math:

```markdown
The generator draws each customer's latent log order rate
$\nu_i \sim N(-0.15, 1.0)$, converts it to a mean order count
$\mu_i = e^{\nu_i}$ ...
```

GitHub renders `$...$` inline math and `$$...$$` display math. md-view passes
the text through goldmark's GFM extension set, which does **not** include a
math extension, so the dollar-delimited LaTeX appears verbatim in the WebView:

```
$\nu_i \sim N(-0.15, 1.0)$
```

### 1.2 Problem B: Hard line breaks produce jagged paragraphs

In `pkg/renderer/renderer.go`, `RenderBody()` configures goldmark with:

```go
goldmark.WithRendererOptions(
    html.WithHardWraps(),   // <-- every "\n" inside a paragraph becomes <br>
    html.WithXHTML(),
)
```

A paragraph authored as:

```markdown
The generator draws each customer's latent log order rate
$\nu_i \sim N(-0.15, 1.0)$, converts it to a mean order count
$\mu_i = e^{\nu_i}$, then draws a per-customer Gamma rate
```

is rendered as three visual lines broken exactly at the author's 80-column
line ends, regardless of the viewport. On a wide window you get long lines
with odd mid-line breaks; on mobile you get short staircase lines ("jagged
edges"). Standard CommonMark behavior (and GitHub, and most books) is that a
single newline is a **soft wrap** — it becomes a space and the paragraph
flows to the container width. Authors who want a hard break write two spaces
before the newline, or a blank line for a new paragraph.

### 1.3 In scope

- MathJax v3 integration for `$...$` (inline) and `$$...$$` (display) math in
  the Wails desktop frontend, and in the legacy full-page `Render()` output.
- Removing hard wraps (making single newlines soft), with a fallback escape
  hatch for users who depended on the old behavior.
- Theme-aware math rendering (light/dark).
- Live-reload compatibility (math must re-typeset after `#content` swaps).
- Tests for both behaviors.

### 1.4 Out of scope

- MathJax v2, server-side pre-rendering (e.g. mathjax-node / temml), and
  KaTeX as the primary engine (see Decision Record DR-2 for why MathJax v3
  over KaTeX).
- Editing support (md-view is a viewer).
- `\(...\)` / `\[...\]` LaTeX-style delimiters are supported by configuration
  but are not required by the example corpus.

---

## 2. What md-view Is — System Overview for a New Engineer

md-view lives in this repository (`github.com/go-go-golems/md-view`). It is a
single-binary desktop application built with **Wails v2**: a Go backend plus
a plain-JavaScript frontend rendered in the OS WebView (WKWebView on macOS,
WebView2 on Windows, WebKitGTK on Linux).

### 2.1 The two rendering paths

There are historically **two** HTML-producing paths. Understanding both is
essential, because the MathJax work touches both:

1. **Desktop (Wails) path — the primary product.**
   `app.go` → `App.openPath()` → `renderer.RenderBody()` → the HTML *fragment*
   is sent to the frontend via a Wails event (`file-opened`), and
   `frontend/dist/app.js` swaps it into `#content.innerHTML`, then calls
   `window.MDSAugmentPage()` (`frontend/dist/augment.js`) to re-inject copy
   buttons and render Mermaid blocks.

2. **Legacy full-page path.**
   `renderer.Render()` wraps `RenderBody()` output with a complete HTML
   document (CSS, scripts, Mermaid, reload) for the old HTTP-server model.
   It is still compiled in and is where all page chrome assembly lives.

### 2.2 End-to-end data flow (current)

```text
┌────────────┐   read file      ┌───────────────────────────────┐
│  .md file  │ ───────────────► │ app.go: App.openPath()        │
└────────────┘                  │  - filepath.Abs               │
                                │  - renderer.RenderBody(abs,   │
                                │      renderer.Options{})      │
                                └───────────────┬───────────────┘
                                        ┌───────▼───────────────┐
                                        │ pkg/renderer:          │
                                        │  extractFrontmatter()  │
                                        │  goldmark (GFM +       │
                                        │   chroma highlighting, │
                                        │   WithHardWraps)       │
                                        │  rewriteImagePaths()   │
                                        └───────┬───────────────┘
                                                │ BodyHTML{Frontmatter,
                                                │           Body, Title}
                          ┌─────────────────────▼──────────────────────┐
                          │ Wails event "file-opened" {html,path,title} │
                          └─────────────────────┬──────────────────────┘
                                                │
                  frontend/dist/app.js: #content.innerHTML = html
                                                │
                          window.MDSAugmentPage()  (augment.js)
                           ├── initCopyButtons()
                           └── initMermaid()      (mermaid.min.js)
                                                │
                                                ▼
                                     WebView paints pixels
```

Live reload: `pkg/watcher` (fsnotify) emits a `file-changed` Wails event on
save; the frontend calls `ReopenCurrent()`, which re-runs `openPath()` and
the whole pipeline, including the `innerHTML` swap and `MDSAugmentPage()`.

### 2.3 The key files you will touch

| File | Role |
|---|---|
| `pkg/renderer/renderer.go` | Markdown → HTML conversion. goldmark setup, frontmatter, image path rewriting, full-page `Render()`. Contains `html.WithHardWraps()` (to remove) and is where math pre-processing/asset plumbing goes. |
| `pkg/renderer/static/` | Embedded static assets (`base.css`, `dark.css`, `mermaid.min.js`, `copy-button.js`, `mermaid-init.js`, `toolbar-buttons.js`, `remarkable-button.js`, `reload.js`). MathJax assets and an `mathjax-init.js` go here. |
| `app.go` | Wails app backend. `openPath()` is the render entry point. |
| `frontend/dist/index.html` | The stable page chrome of the desktop app. Loads `mermaid.min.js`, `augment.js`, `app.js` as static `<script>` tags. A MathJax `<script>` tag is added here. |
| `frontend/dist/app.js` | Frontend controller: handles `file-opened`, swaps `#content.innerHTML`, calls `MDSAugmentPage()`. |
| `frontend/dist/augment.js` | Re-runnable content augmentation (copy buttons, Mermaid). MathJax typesetting is added here. |
| `frontend/dist/base.css`, `dark.css`, `style.css` | Content styling, light and dark themes. Math overflow/theme CSS goes here. |
| `pkg/renderer/renderer_test.go` | Renderer tests. New tests for math and soft-wrapping go here. |
| `Makefile` | `frontend-css` target regenerates `frontend/dist/{base.css,dark.css,ui.css,chroma.css,...}` from `pkg/renderer/static` and generated chroma CSS. Run it after touching embedded static assets. |

Note: `frontend/dist/*` is **generated/synced** from `pkg/renderer/static`
plus a build step. Check `Makefile` target `frontend-css` and the Wails asset
configuration (`assets.go`, `wails.json`) before hand-editing dist files;
edit the source-of-truth and re-run `make frontend-css`.

### 2.4 Where the old daemon/HTTP model still shows

`renderer.Render()` builds a standalone page with `<script
src="http://localhost:%d/static/mermaid.min.js">`. The Wails frontend instead
loads the same libraries as local static assets via `index.html`. When adding
MathJax you must mirror the **mermaid pattern** exactly: embed in
`pkg/renderer/static/`, expose via Go embed, wire into both `Render()` and
`frontend/dist/index.html`.

---

## 3. Current-State Architecture, With Evidence

### 3.1 goldmark pipeline (`pkg/renderer/renderer.go`, `RenderBody()`)

```go
md := goldmark.New(
    goldmark.WithExtensions(
        extension.GFM,
        highlighting.NewHighlighting(
            highlighting.WithStyle(chromaStyle),
            highlighting.WithFormatOptions(chroma_html.WithClasses(true)),
        ),
    ),
    goldmark.WithRendererOptions(
        html.WithHardWraps(),
        html.WithXHTML(),
    ),
)
var buf bytes.Buffer
md.Convert(body, &buf)
renderedHTML := rewriteImagePaths(buf.String(), filePath, opts.Port)
```

- `extension.GFM` gives tables, task lists, strikethrough, autolinks.
- No math extension is registered. goldmark has **no built-in math**; there
  is an upstream `goldmark-mathjax` extension (see DR-1) that adds
  `$...$`/`$$...$$` parsing, but the simplest correct approach for a viewer
  is client-side MathJax scanning (see §5).
- `html.WithHardWraps()` maps `\n` inside paragraphs to `<br />` — the root
  cause of Problem B.

### 3.2 Frontend swap loop (`frontend/dist/app.js`)

On `file-opened`:

```js
contentDiv.innerHTML = html;                    // full fragment swap
if (window.MDSAugmentPage) window.MDSAugmentPage();
```

And `augment.js` exposes `window.MDSAugmentPage()` (idempotent, re-runnable
after every swap) plus `window.MDSMermaidRerender(theme)` for theme switches.
**Any MathJax integration must follow this same idempotent, re-runnable
contract**, because MathJax's default `typesetPromise()` on a whole document
does not "just work" when you replace `innerHTML` afterwards.

### 3.3 Theme switching

Themes are driven by a `data-theme` attribute on `<html>`/`<body>` and CSS
`[data-theme="dark"]` overrides (`dark.css`, chroma dual-theme CSS via
`ChromaCSSBoth()`). Mermaid re-renders on theme change via
`MDSMermaidRerender`. MathJax output color is inherited (it uses `currentColor`),
so mostly it needs CSS, not re-typesetting — but see §6.3 for the dark-theme
`mjx-container` contrast caveat.

### 3.4 Static asset embedding pattern

`pkg/renderer/renderer.go` uses `//go:embed static/<file>` and exposes getter
functions (`MermaidJS()`, `CopyButtonJS()`, ...). `assets.go` registers the
Wails Assets handler that serves `frontend/dist` to the WebView.

---

## 4. Requirements

**Functional**

- R1: Inline math `$...$` renders typeset math in the desktop app.
- R2: Display math `$$...$$` (paragraph-level, one per block) renders as a
  centered display equation.
- R3: Math renders in both light and dark themes with readable contrast.
- R4: Math re-typesets after live-reload and file-open swaps.
- R5: Code blocks containing `$` (e.g. shell variables, currency) are **not**
  typeset. Inline code `` `...$...` `` likewise.
- R6: A single newline within a paragraph renders as a space (soft wrap).
  Paragraphs reflow to the container width on any window size, including
  narrow/mobile.
- R7: An explicit hard break (two trailing spaces + newline, or `\` + newline
  with GFM) still produces a `<br>`.

**Non-functional**

- N1: No network access at runtime — MathJax must be bundled/embedded (the
  app is a local file viewer; offline must work).
- N2: Rendering must stay fast; MathJax v3 is async and lazy, but we must
  typeset only on demand (after swaps), not on a timer.
- N3: Binary size growth from embedding MathJax (~1 MB gzipped for the
  tex-chromium component bundle) is acceptable; keep only the components
  needed (see §5.2).
- N4: Existing tests keep passing; new tests cover math passthrough and soft
  wrap behavior.

---

## 5. Proposed Design: MathJax Integration

### 5.1 Approach overview

MathJax v3 works as a client-side typesetter: you include `mathjax/tex-mml-chtml.js`
(or a custom component), configure delimiters, and call
`MathJax.typesetPromise([element])` on the region containing new content.
Crucially, MathJax can **scan the raw text nodes of the DOM**, finding
`$...$` and `$$...$$` itself. That means we do **not** need a goldmark
extension to produce `<math>` or custom spans; we keep goldmark untouched for
math and let MathJax do delimiter scanning after each content swap.

Pipeline addition (bold = new):

```text
RenderBody() ──► innerHTML swap ──► MDSAugmentPage()
                                     ├── initCopyButtons()
                                     ├── initMermaid()
                                     └── initMath()            (NEW)
                                          └─ MathJax.typesetPromise(#content)
```

### 5.2 Which MathJax artifact to embed

MathJax v3 ships prebuilt *components*. Recommended: **`tex-chtml.js`**
(TeX input → CommonHTML output). It is a single ~1 MB file (minified, ~300 KB
gzipped) that includes the TeX input processor and the CHTML output — no
fonts to fetch beyond the embedded `output/chtml/fonts/woff-v2` (CHTML uses
system/webfont math glyphs; the tex-chromium component embeds the fonts).
Alternatives:

- `tex-svg.js` — SVG output; slightly larger output DOM but no font files and
  crisper scaling. Both are viable; **choose `tex-svg.js`** to avoid font
  loading entirely (see DR-3).
- `tex-mml-chtml.js` — also accepts MathML; unnecessary here.

Vendor it as `pkg/renderer/static/mathjax.min.js` (single-file component,
matching how `mermaid.min.js` is vendored) plus `frontend/dist/mathjax.min.js`
via `make frontend-css`/asset sync.

### 5.3 Configuration and init script

Create `pkg/renderer/static/mathjax-init.js`:

```js
// mathjax-init.js — MathJax v3 configuration and idempotent typesetting.
//
// MathJax itself is loaded as a plain <script> (mathjax.min.js) BEFORE this
// file's window.MDSMathInit() is first called. Because MathJax v3 reads its
// global MathJax config at load time, we set window.MathJax BEFORE the
// library script tag in index.html (see §5.4).
window.MathJax = {
    tex: {
        inlineMath: [['$', '$'], ['\\(', '\\)']],
        displayMath: [['$$', '$$'], ['\\[', '\\]']],
        processEscapes: true,       // allow \$ to escape a dollar
        processEnvironments: false  // we do not support \begin{...} env blocks
    },
    options: {
        skipHtmlTags: ['script', 'noscript', 'style', 'textarea', 'pre', 'code'],
        ignoreHtmlClass: 'md-view-nomath',
        renderActions: {
            addMenu: [],            // disable context menu for cleaner UX
            checkLoading: []
        }
    },
    startup: {
        typeset: false              // we typeset manually after swaps
    }
};

// Idempotent, re-runnable typeset of the content area.
// Follows the same contract as window.MDSAugmentPage().
window.MDSMathTypeset = function () {
    var content = document.getElementById('content');
    if (!content || typeof MathJax === 'undefined' ||
        !MathJax.typesetPromise) return Promise.resolve();
    return MathJax.typesetPromise([content]).catch(function (err) {
        console.error('MathJax typeset error:', err);
    });
};
```

Key points an intern must understand:

- **`skipHtmlTags` includes `pre` and `code`** — this satisfies R5 without any
  goldmark work. MathJax will not scan code blocks or inline code.
- **`startup.typeset: false`** — the page chrome is stable in the Wails model;
  we typeset only `#content` after each swap.
- **`processEscapes: true`** — `\$` writes a literal dollar, needed for
  finance-flavored documents.

### 5.4 Wiring into the desktop frontend

`frontend/dist/index.html` — add before `augment.js` (order matters: the
config object must exist before the library script loads):

```html
<script src="mathjax-config.js"></script>   <!-- sets window.MathJax -->
<script src="mathjax.min.js" async></script> <!-- vendored tex-svg.js -->
<script src="augment.js"></script>
```

`frontend/dist/augment.js` — extend `MDSAugmentPage()`:

```js
window.MDSAugmentPage = function () {
    initCopyButtons();
    initMermaid();
    initMathTypeset();          // NEW
};

function initMathTypeset() {
    // MathJax loads async; if it isn't ready yet, queue for startup.promise.
    if (window.MathJax && window.MathJax.typesetPromise) {
        window.MDSMathTypeset();
    } else if (window.MathJax && window.MathJax.startup) {
        window.MathJax.startup.promise.then(window.MDSMathTypeset);
    }
    // If MathJax is entirely absent, silently skip (viewer still works).
}
```

Note the subtlety: when `mathjax.min.js` has `async`, it may not be loaded at
the first `MDSAugmentPage()` call. `MathJax.startup.promise` resolves when the
library is fully initialized, so chaining through it covers the race. This is
the trickiest timing detail — test the very first file opened at startup
(`OnDomReady` → `file-opened` → swap happens before MathJax finishes loading)
and a file opened afterwards.

### 5.5 Wiring into the legacy full-page renderer

In `renderer.Render()` (mirroring the mermaid pattern):

```go
//go:embed static/mathjax.min.js
var mathjaxJS []byte

//go:embed static/mathjax-config.js
var mathjaxConfigJS []byte

func MathjaxJS() []byte     { return mathjaxJS }
func MathjaxConfigJS() []byte { return mathjaxConfigJS }
```

And in the page assembly (pseudocode of the added block):

```
mathScript := fmt.Sprintf(`<script>
%s
</script>
<script src="http://localhost:%d/static/mathjax.min.js"></script>
<script>
MathJax.startup.promise.then(function(){ return MathJax.typesetPromise([document.body]); });
</script>`, string(mathjaxConfigJS), opts.Port)
```

The legacy path also serves `/static/*` from the daemon routes (where
`mermaid.min.js` is already served); add `mathjax.min.js` to the same handler.

### 5.6 CSS

Add to `pkg/renderer/static/base.css` (and dark overrides in `dark.css`):

```css
/* MathJax containers: allow long display equations to scroll horizontally
   instead of overflowing the page on narrow/mobile widths. */
mjx-container[jax="CHTML"][display="true"],
mjx-container[jax="SVG"] {
    overflow-x: auto;
    overflow-y: hidden;
    padding: 2px 0;
}
/* Display equations centered like GitHub. */
mjx-container[display="true"] {
    margin: 1em 0;
    text-align: center;
}
/* Dark theme: MathJax inherits currentColor, so math text follows the body
   color; only the menu/tooltip chrome needs adjusting if enabled. */
[data-theme="dark"] mjx-container {
    color: inherit;
}
```

---

## 6. Proposed Design: Paragraph Soft-Wrapping

### 6.1 Root cause and the one-line fix

goldmark's `html.WithHardWraps()` converts every newline in paragraph text to
`<br />`. Standard CommonMark (and GitHub) treats a single newline as a
space. The fix in `RenderBody()`:

```go
goldmark.WithRendererOptions(
    // REMOVED: html.WithHardWraps(),
    html.WithXHTML(),
),
```

That single deletion implements R6. goldmark's *default* paragraph renderer
emits soft-wrapped text (newlines are normalized to spaces), while:

- a blank line still starts a new `<p>` (unaffected),
- `"  \n"` (two trailing spaces) still emits `<br />` (goldmark handles this
  in the parser regardless of `WithHardWraps`),
- list items, headings, tables, code blocks are block-level and unaffected.

### 6.2 Behavior change and escape hatch

This is a deliberate behavior change (documented, no back-compat shim per
repo convention — update tests/docs instead). If a user genuinely wants the
old GitHub-comment-style hard wraps, they can:

1. Write two trailing spaces or a trailing backslash where a break is
   intended (CommonMark/GFM), or
2. (Optional, stretch goal) add a `--hard-wraps` CLI flag / frontmatter
   `hardWraps: true` that sets the option again. **Recommendation: skip the
   flag in v1** — soft wrap is the correct default and the flag is unused
   complexity. Record as a future option only.

### 6.3 Interaction between the two changes

Display math `$$...$$` blocks in the example corpus are authored as a
`$$`-delimited block with newlines inside. This is fine for MathJax scanning
(a paragraph's text nodes still contain the source), but note two
interactions:

- With hard wraps removed, a multi-line inline-math source
  (`$\mu_i =\n e^{\nu_i}$` split across lines) previously contained `<br>`
  inside the `$...$` span. MathJax's TeX parser treats a `<br>`-free newline
  as a space in math mode — which is the correct LaTeX semantics. Removing
  hard wraps therefore *also improves* math robustness. Do not add
  `processHtml` tricks; plain soft-wrapped text is what TeX expects.
- After MathJax typesets, it **replaces** the `$...$` text with
  `mjx-container` markup. A later re-render of the same content must start
  from the freshly swapped innerHTML (it does — each `file-opened`/reload
  swap replaces the whole fragment), so no "double typeset" issue arises.

---

## 7. Decision Records

### DR-1: Client-side MathJax scanning vs. goldmark math extension

- **Context:** We need `$...$`/`$$...$$` to render. Options: (a) add the
  `goldmark-mathjax` goldmark extension so the parser emits `<span
  class="math">`/`<div class="math">` and then typeset with MathJax; (b) let
  MathJax scan raw text.
- **Options considered:** goldmark-mathjax extension (github.com/litao91/goldmark-mathjax);
  MathJax native scanning; KaTeX auto-render.
- **Decision:** (b) MathJax native scanning.
- **Rationale:** Zero Go-side parsing risk (delimiters inside code are
  excluded by `skipHtmlTags`), one less dependency, and MathJax's scanner is
  battle-tested. The goldmark extension buys structure we don't need in a
  viewer.
- **Consequences:** Math is not visible in the HTML fragment until JS runs
  (fine for a WebView app); plain-`Render()` consumers without MathJax see
  raw `$...$` (acceptable, same as GitHub's raw markdown).
- **Status:** accepted.

### DR-2: MathJax v3 vs. KaTeX

- **Context:** Both are client-side TeX renderers. KaTeX is faster and
  smaller; MathJax has broader LaTeX coverage (e.g. `\operatorname`,
  aligned environments, macros used in statistics reports).
- **Decision:** MathJax v3.
- **Rationale:** The motivating corpus is Bayesian statistics reports using
  constructs like `\operatorname{Var}(Y_i \mid \mu_i)` and display
  derivations; MathJax's coverage avoids "unsupported command" surprises.
  MathJax v3 performance is acceptable for a single-document viewer.
- **Consequences:** Larger bundle (~1 MB embedded vs ~300 KB KaTeX+fonts).
- **Status:** accepted. Revisit if startup latency complaints arise.

### DR-3: CHTML vs SVG MathJax output

- **Context:** tex-chtml needs webfont files; tex-svg embeds glyphs as SVG.
- **Decision:** `tex-svg.js`.
- **Rationale:** Single self-contained file, no font-loading races in the
  Wails asset server, matches the "one vendored .min.js" pattern of
  mermaid.min.js.
- **Consequences:** Slightly heavier DOM for very math-dense pages; visually
  equivalent.
- **Status:** accepted.

### DR-4: Remove `WithHardWraps` unconditionally vs. make it configurable

- **Context:** see §6.2.
- **Decision:** Remove unconditionally; no flag in v1.
- **Rationale:** Soft wrap is CommonMark/GitHub default and the user-reported
  desired behavior; a flag is speculative complexity.
- **Consequences:** Behavior change is documented in README/docs and tests.
- **Status:** accepted.

---

## 8. Phased Implementation Plan

Work top-down; each phase is independently committable and testable.

### Phase 1 — Soft-wrapping (small, do first)

1. In `pkg/renderer/renderer.go` `RenderBody()`: delete `html.WithHardWraps()`
   from `goldmark.WithRendererOptions(...)`.
2. Update/extend `pkg/renderer/renderer_test.go`:
   - Test A: `"line one\nline two\nline three"` in a paragraph renders as a
     single `<p>` with **no** `<br />`.
   - Test B: `"line one  \nline two"` (two trailing spaces) still renders
     `<br />`.
   - Test C: blank line still splits paragraphs.
3. Run `gofmt`, `go test ./... -count=1`, `make build`.
4. Manually verify in the app with
   `/home/manuel/code/wesen/2026-08-31--bayesian-marketing/artifacts/report-day1.md`
   — resize the window narrow/wide; paragraphs must reflow.

### Phase 2 — Vendor and embed MathJax

1. Download the MathJax v3 component:
   `curl -L -o pkg/renderer/static/mathjax.min.js \
      https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-svg.js`
2. Create `pkg/renderer/static/mathjax-config.js` (contents in §5.3; the
   `window.MathJax = {...}` part plus `window.MDSMathTypeset`).
3. Add `//go:embed` declarations and getter functions in
   `pkg/renderer/renderer.go` (`MathjaxJS()`, `MathjaxConfigJS()`), mirroring
   the mermaid block.
4. Sync into `frontend/dist/` (run `make frontend-css` / copy per the
   Makefile's dist-sync rules) so the Wails asset server serves it.

### Phase 3 — Desktop frontend wiring

1. `frontend/dist/index.html`: add the two `<script>` tags (config first,
   then `mathjax.min.js` with `async`) before `augment.js`.
2. `frontend/dist/augment.js`: add `initMathTypeset()` and call it from
   `MDSAugmentPage()` (§5.4).
3. Add CSS from §5.6 to `base.css`/`dark.css` sources.
4. Manual test matrix:
   - First file opened at startup (MathJax still loading → promise chain).
   - File opened later (MathJax ready).
   - Live-reload on save while a math-heavy file is open.
   - Theme toggle while math is displayed.
   - A file with `$` in code blocks and inline code (must not typeset).
   - Narrow window: display equations scroll, not overflow.

### Phase 4 — Legacy full-page renderer wiring

1. In `renderer.Render()` add the math script block (§5.5).
2. Ensure the daemon-style `/static/` route serves `mathjax.min.js` (follow
   the mermaid registration).
3. Add a renderer test that the full page contains the MathJax script and
   config when rendered.

### Phase 5 — Tests, docs, polish

1. Go tests: soft-wrap behavior (Phase 1), asset getters non-empty, full-page
   render includes MathJax tags.
2. Update `README.md`, `docs/user-guide.md`, `docs/getting-started.md` with:
   math support, delimiters, `\$` escaping, and the paragraph-reflow behavior
   change.
3. Run `make lint` and full test suite; commit per phase.

---

## 9. Testing and Validation Strategy

**Unit tests (Go, `pkg/renderer/renderer_test.go`):**

```go
func TestParagraphSoftWrap(t *testing.T) {
    // "a\nb" in a paragraph → <p>a b</p> with no <br />
}

func TestHardBreakStillWorks(t *testing.T) {
    // "a  \nb" (two trailing spaces) → <br />
}

func TestMathPassesThroughUntouched(t *testing.T) {
    // "$x^2$" appears verbatim in Body — MathJax handles it client-side
}

func TestRenderIncludesMathJaxScripts(t *testing.T) {
    // full Render() output contains "mathjax.min.js" and "MDSMathTypeset"
}
```

**Manual acceptance (desktop app):**

| Scenario | Expected |
|---|---|
| Open report-day1.md | All `$...$` inline math typeset; `$$...$$` centered |
| Narrow window (mobile width) | Paragraphs reflow full-width, no jagged edges; display equations scroll horizontally |
| Theme toggle | Math stays readable in dark mode |
| Save file with watcher active | Math re-typesets after reload |
| File with shell code `$VAR` | Code blocks show literal `$VAR`, no typesetting |
| `\$\$100` in prose | Literal "$100" |

**Regression:** `go test ./... -count=1`, `make build`, `make lint`.

---

## 10. Risks, Alternatives, Open Questions

**Risks**

- *Bundle size:* tex-svg.js is ~1 MB embedded. Acceptable for a desktop app;
  noted in DR-3.
- *First-paint flash of raw `$...$`:* Math typesets asynchronously; there is
  a brief moment of raw LaTeX before typeset. Mitigate visually only if it
  bothers users (e.g. CSS fade-in); not blocking.
- *MathJax scan of non-content chrome:* we scope typesetting to `#content`,
  so toolbar/sidebar/frontmatter are never scanned (frontmatter is inside
  `#content` in the desktop fragment — if it contains `$`, it would be
  scanned; acceptable, or add `md-view-nomath` class to the frontmatter
  `<details>` via `formatFrontmatterHTML` — cheap, do it).
- *Behavior change* from hard wraps may surprise users who authored
  intentionally short lines (e.g. ASCII "poetry"). Documented; GFM escape
  hatches exist.

**Alternatives**

- KaTeX + auto-render extension (rejected, DR-2).
- goldmark math extension producing semantic HTML (rejected, DR-1).
- Server-side pre-rendering to static SVG at render time in Go (rejected:
  requires a JS runtime dependency in the Go binary; big complexity).

**Open questions**

- Should `\begin{align}...\end{array}` LaTeX environments inside `$$...$$`
  be enabled (`processEnvironments`)? Default off; revisit on demand.
- Should the reMarkable upload flow (which renders the HTML server-side)
  eventually get math? Out of scope; the PDF pipeline is separate.

---

## 11. References

Key files (all paths relative to repo root `/home/manuel/code/wesen/2026-05-07--md-server`):

- `pkg/renderer/renderer.go` — goldmark setup (`RenderBody`, line ~634 for
  `WithHardWraps`), full-page `Render()`, embed pattern.
- `pkg/renderer/static/` — embedded assets (mermaid pattern to copy).
- `pkg/renderer/renderer_test.go` — existing test conventions.
- `app.go` — `App.openPath()` (render entry), Wails event wiring.
- `frontend/dist/index.html`, `app.js`, `augment.js` — frontend swap loop and
  augmentation contract (`MDSAugmentPage`, `MDSMermaidRerender`).
- `frontend/dist/base.css`, `dark.css` — theme CSS.
- `Makefile` — `frontend-css`, `build` targets.
- Example corpus:
  `/home/manuel/code/wesen/2026-08-31--bayesian-marketing/artifacts/report-day1.md`.

External APIs:

- MathJax v3 docs: configuration (`MathJax = {tex: {inlineMath, displayMath,
  processEscapes}, options: {skipHtmlTags}, startup: {typeset}}`),
  `MathJax.typesetPromise([elements])`, `MathJax.startup.promise` —
  https://docs.mathjax.org/en/latest/
- goldmark renderer options: `html.WithHardWraps`,
  `html.WithXHTML` — https://pkg.go.dev/github.com/yuin/goldmark/renderer/html
- CommonMark spec on soft line breaks:
  https://spec.commonmark.org/0.31.2/#soft-line-breaks
