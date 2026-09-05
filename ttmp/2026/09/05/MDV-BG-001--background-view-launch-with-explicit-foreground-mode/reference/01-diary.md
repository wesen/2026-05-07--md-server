---
Title: Diary
Ticket: MDV-BG-001
Status: active
Topics:
    - cli
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://README.md
      Note: Updated lifecycle overview
    - Path: repo://docs/getting-started.md
      Note: New default and diagnostics
    - Path: repo://docs/user-guide.md
      Note: Flag logs and migration
    - Path: repo://internal/launch/detach_nonunix_test.go
      Note: Native-test limitation
    - Path: repo://internal/launch/detach_other.go
      Note: Unsupported platform error
    - Path: repo://internal/launch/detach_unix.go
      Note: Unix session policy
    - Path: repo://internal/launch/detach_unix_test.go
      Note: Session verification
    - Path: repo://internal/launch/detach_windows.go
      Note: Windows process flags
    - Path: repo://internal/launch/launch.go
      Note: Detached launch implementation
    - Path: repo://internal/launch/launch_test.go
      Note: Real child process and failures
    - Path: repo://main_test.go
      Note: Command dispatch and validation
ExternalSources: []
Summary: Chronological background-view implementation and delivery evidence.
LastUpdated: 2026-09-05T17:35:00-04:00
WhatFor: Review and resume work with explicit rationale and validation evidence.
WhenToUse: When reviewing the launch lifecycle change.
---


# Diary

## Goal

Record investigation, design, implementation, testing, printed phase slips, and reMarkable delivery for background-by-default explicit view commands.

## Step 1: Isolate the task and design the launch boundary

Created `/home/manuel/code/wesen/go-go-golems/md-view-background` on branch `task/background-view`, based on clean current HEAD `d11c193`. The original checkout is untouched. Interpreted “new docmgr app” as a new docmgr ticket for this application change, rather than a second unrelated application.

Mapped Cobra dispatch, Wails readiness and single-instance forwarding, path normalization, rendering, and watcher ownership. The design adds a small OS launcher around the unchanged desktop lifecycle and deliberately uses the literal requested `--foregruond` flag. The detailed intern guide includes diagrams, API references, decisions, pseudocode, file anchors, risks, and phased test instructions.

### Prompt Context

**User prompt (verbatim):**

> Create a new docmgr app to let `md-view view ...` run in the background, and only with --foregruond flag stay in foreground. Create a new task worktree and work in that, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
>
> Then work on it, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done.
>
> [REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory.

**Assistant interpretation:** Create an isolated ticket/worktree, publish an intern-ready design, then implement background default with explicit foreground mode, phased printing, commits, and a detailed diary.

**Inferred user intent:** Regain the terminal after opening Markdown while keeping implementation understandable and progress available on paper and reMarkable.

### What I did

- Ran `git worktree add ../md-view-background -b task/background-view`.
- Created ticket MDV-BG-001 and design/diary documents with docmgr.
- Read `AGENT.md`, `main.go`, `cli.go`, `cli_test.go`, lifecycle portions of `app.go`, Makefile, README, and documentation references.
- Ran `go test -tags webkit2_41 ./...`: all packages passed before changes.
- Printed overall plan and P1 START using `work_slip.py`; both responses reported `printed: true` (2026-09-05T21:29:21Z and 21:29:23Z).
- Wrote `design-doc/01-background-launch-intern-guide.md` before implementation.

### Why

The code already has correct Wails ownership and file forwarding. Re-executing the same binary preserves these boundaries; a goroutine would not detach the desktop from its parent process.

### What worked

Worktree creation, baseline tests, docmgr initialization, and both actual thermal prints succeeded. DISPLAY is available as `:0`, and wails/remarquee/tmux are installed.

### What didn't work

No failures in the investigation commands. Native Windows/macOS runtime validation is unavailable on this Linux host and will not be claimed from cross-compilation.

### What I learned

Wails forwards os.Args and macOS can supply the executable directory as cwd. Resolve the file before re-exec and retain the existing absolute-argument handling. Bare launch supports double-click/development and should remain direct.

### What was tricky to build

The main design issue is distinguishing successful process creation from a ready window. A sleep would not prove readiness, and adding an IPC handshake would enlarge the scope. Chose an explicit exec-only acknowledgement with private log output and foreground diagnostics.

### What warrants a second pair of eyes

Review the literal flag spelling and explicit-view-only scope; review process session/stream detachment and single-instance forwarding invariants before implementation.

### What should be done in the future

Implement P2; run native production verification in P3; retain native cross-platform verification as an explicit limitation if unavailable.

### Code review instructions

Start with the design's current-state map and decision records. Compare the baseline `main.go:32-148`, `cli.go:29-80`, and `app.go:53-153`. Re-run `go test -tags webkit2_41 ./...`.

### Technical details

- Worktree: `/home/manuel/code/wesen/go-go-golems/md-view-background`.
- Branch: `task/background-view`.
- Phases: P1 investigate/publish; P2 implement; P3 validate/deliver.
- Primary guide: `design-doc/01-background-launch-intern-guide.md`.


## Step 2: Implement detached launch without touching the desktop lifecycle

Implemented a small Wails-independent launcher package and a testable Cobra command factory. Default explicit view now re-executes the binary with `--foregruond`, absolute file arguments, and optional dark mode. The child inherits cwd/environment but gets null stdin, private log-backed output, and platform-specific detachment attributes.

The desktop callbacks and renderer remain unchanged. User docs now distinguish the temporary launcher from the long-lived desktop and explain asynchronous errors, log retention, intentional flag spelling, and unchanged bare launch. Focused and full package tests passed on the first implementation run.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the published process-ownership design and verify it independently of a native WebView.

**Inferred user intent:** Make the normal command return immediately without losing file/theme behavior or the ability to debug in foreground.

**Commit (code):** 2db896d23a764aa284b4ef1d7d75ebc9db629588 — "feat: detach view launches unless --foregruond is set"

### What I did

- Added `internal/launch/launch.go`, OS-specific detachment files, and portable/helper-process tests.
- Extracted `newRootCommand` and added dispatch/error/argument-validation tests in `main_test.go`.
- Added forwarded-child argument regression coverage in `cli_test.go`.
- Updated README, getting-started, and user-guide command/lifecycle explanations.
- Ran `gofmt -w main.go main_test.go cli_test.go internal/launch/*.go`, `go test -tags webkit2_41 ./...`, and `git diff --check`: passed.
- Committed implementation separately from ticket bookkeeping.
- P1 publication succeeded after dry-run: `OK: uploaded MDV-BG-001 Background Launch Guide.pdf -> /ai/2026/09/05/MDV-BG-001`.
- P1 DONE and P2 START slips reported `printed: true` at 21:35:11Z and 21:35:14Z. P1 design commit is `33c35b3`.

### Why

`exec.Command` avoids shell interpretation and preserves one deployable binary. The foreground flag both serves the user and prevents child recursion. Private regular log files prevent shell pipes from staying open and retain asynchronous startup stderr.

### What worked

The real helper process test confirmed Start returns before the child exits, args/cwd/environment survive, stdin reaches EOF, stdout/stderr reach a 0600 log, and on Unix the child SID equals its PID. A release-file gate shuts down the helper; a 15-second deadline limits leaks if the parent is interrupted. Invalid executable startup removes unused logs. Command tests confirm invalid flags/extra files do not launch anything.

### What didn't work

No failing implementation tests or commands in this step. Windows/macOS runtime behavior remains untested; their compilation and Linux native GUI smoke are scheduled for P3.

### What I learned

The launch package can be tested without any GTK or Wails dependencies. `Process.Release` is the appropriate ownership handoff for a short-lived CLI, while `Wait` would restore the original blocking behavior. A process PID is not a window identity when Wails hands off to an existing instance.

### What was tricky to build

Null standard streams and session detachment solve different problems: Setsid removes the controlling terminal, but inherited stdout would still keep a caller's pipe open. Set both output descriptors to a regular log and stdin to nil. Windows requires closing a failed-start log before removing it. Canonical arguments place an absolute file after `--`, avoiding both shell injection and flag-looking file names while retaining lenient Wails forwarding.

### What warrants a second pair of eyes

Review platform attributes, cleanup after Start failure, and the distinction between exec acknowledgement and readiness. The launcher does not wait/reap indefinitely because the CLI exits immediately; do not reuse it as a long-running supervisor API without revisiting ownership.

### What should be done in the future

Run race/vet/full production build and native smoke; cross-compile the isolated package; publish final evidence and diary. Native Windows/macOS runtime checks remain follow-up work if no host is available.

### Code review instructions

Start with `main.go:newRootCommand`, then `internal/launch/launch.go:Start`, `childArgs`, and `start`. Inspect each `detach_*.go` policy. Run `go test -tags webkit2_41 ./...` and `go test ./internal/launch -count=10` to exercise process timing repeatedly.

### Technical details

Child vector: `view --foregruond [--dark] -- /absolute/file`. Logs: `os.UserCacheDir()/md-view/launch-*.log`. Unix: Setsid. Windows: DETACHED_PROCESS plus CREATE_NEW_PROCESS_GROUP. Other platforms return an actionable foreground-mode error.
