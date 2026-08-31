# Changelog

## 2026-08-31

- Initial workspace created


## 2026-08-31

Created ticket: intern-ready design doc for MathJax ($...$/1010835...1010835) support and paragraph soft-wrapping (remove WithHardWraps), with phased implementation plan, decision records, and test strategy

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/ttmp/2026/08/31/MD-MATH-SOFTWRAP--mathjax-rendering-and-paragraph-soft-wrapping--add-mathjax-support-and-paragraph-soft-wrapping-to-md-view/design-doc/01-mathjax-support-and-paragraph-soft-wrapping-analysis-design-and-implementation-guide.md — primary deliverable


## 2026-08-31

Phase 1: removed html.WithHardWraps, paragraphs now soft-wrap; added 3 regression tests (commit b842010)

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/pkg/renderer/renderer.go — soft wrap change


## 2026-08-31

Phase 2: vendored MathJax v3 tex-svg + config, go:embed + getters, synced to frontend/dist (commit edbf71f)

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/pkg/renderer/renderer.go — mathjax embeds


## 2026-08-31

Phase 3: MathJax wired into desktop frontend (index.html, augment.js initMathTypeset, CSS); browser-verified (commit 9224848)

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/frontend/dist/augment.js — typeset after content swaps


## 2026-08-31

Phase 4: MathJax in legacy full-page Render() + ordering test (commit 9438be0)

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/pkg/renderer/renderer_test.go — TestRenderIncludesMathJaxScripts


## 2026-08-31

Phase 5: docs for math + soft wrap in README/user-guide; make test green, make lint 0 issues (commit 3a52b16). All phases complete.

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/README.md — feature bullets


## 2026-08-31

Follow-ups: sidebar toggle off-by-default + clip-article button (58568c7); fixed overlapping toolbar icons via scoped row layout (8570621); replaced stale legacy daemon binary

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/frontend/dist/app.js — sidebar toggle persistence


## 2026-08-31

Ticket closed

