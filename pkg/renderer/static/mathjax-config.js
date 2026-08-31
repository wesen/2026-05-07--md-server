// mathjax-config.js — MathJax v3 configuration and idempotent typesetting
// for md-view.
//
// This file MUST be loaded BEFORE mathjax.min.js: MathJax v3 reads its
// global configuration from window.MathJax at library load time.
//
// Typesetting itself is re-runnable and scoped to #content, following the
// same contract as window.MDSAugmentPage() in augment.js: the Wails frontend
// swaps #content.innerHTML on every file open / live reload, so we typeset
// the content region after each swap (augment.js calls MDSMathTypeset()).
window.MathJax = {
    tex: {
        inlineMath: [['$', '$'], ['\\(', '\\)']],
        displayMath: [['$$', '$$'], ['\\[', '\\]']],
        processEscapes: true,        // \$ writes a literal dollar
        processEnvironments: false   // no \begin{...} environment blocks
    },
    options: {
        // Never scan code blocks or inline code: dollar signs in shell
        // snippets or currency stay literal.
        skipHtmlTags: ['script', 'noscript', 'style', 'textarea', 'pre', 'code'],
        ignoreHtmlClass: 'md-view-nomath',
        renderActions: {
            addMenu: [],       // disable the MathJax context menu
            checkLoading: []
        }
    },
    startup: {
        typeset: false  // we typeset manually after content swaps
    }
};

// Idempotent, re-runnable typeset of the content area.
// Safe to call repeatedly; silently no-ops if MathJax is not loaded.
window.MDSMathTypeset = function () {
    var content = document.getElementById('content') || document.body;
    if (typeof MathJax === 'undefined' || !MathJax.typesetPromise) {
        return Promise.resolve();
    }
    return MathJax.typesetPromise([content]).catch(function (err) {
        console.error('MathJax typeset error:', err);
    });
};
