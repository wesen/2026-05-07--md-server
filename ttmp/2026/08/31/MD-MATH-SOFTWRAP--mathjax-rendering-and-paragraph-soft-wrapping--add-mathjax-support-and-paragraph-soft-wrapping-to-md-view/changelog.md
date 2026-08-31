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

