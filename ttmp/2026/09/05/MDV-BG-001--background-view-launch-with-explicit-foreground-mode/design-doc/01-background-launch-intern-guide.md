---
Title: Background launch intern guide
Ticket: MDV-BG-001
Status: active
Topics:
    - cli
    - wails
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://AGENT.md
      Note: Native build and tmux requirements
    - Path: repo://Makefile
      Note: Required Wails build and tests
    - Path: repo://app.go
      Note: Readiness and single-instance handoff
    - Path: repo://cli.go
      Note: Absolute paths and forwarded arguments
    - Path: repo://cli_test.go
      Note: Existing parser regression coverage
    - Path: repo://main.go
      Note: Cobra dispatch and Wails lifecycle
ExternalSources: []
Summary: Intern-oriented design and implementation guide for detached desktop launch.
LastUpdated: 2026-09-05T17:35:00-04:00
WhatFor: Understand and implement background-by-default view invocations.
WhenToUse: Before changing CLI process ownership or troubleshooting detached launch.
---



# Background launch intern guide

## 1. Executive summary

`md-view` is a native Markdown desktop application, not a web server. Today its Cobra `view` command directly enters the Wails event loop and does not return until the window closes. This ticket changes **process ownership**, not rendering: `md-view view file.md` will start a detached copy of the same executable and immediately return to the shell. `md-view view --foregruond file.md` will instead run Wails in the invoking process.

The unusual spelling `--foregruond` is intentional: it is the exact requested public flag. There is no additional spelling alias. Bare `md-view` remains a direct desktop launch for double-click and Wails development workflows; only the explicit `view` command changes. The background copy internally uses the same foreground flag to avoid recursively spawning copies.

A successful background command means the operating system accepted the child process, **not** that a window finished opening or the file rendered. Failures after exec are asynchronous and go to a private per-launch log. This distinction is fundamental to the design and must remain visible in help and documentation.

## 2. Problem, scope, and vocabulary

A shell normally waits for a command it starts. Wails owns a native GUI event loop and blocks the goroutine calling `wails.Run` until application shutdown. Combining those behaviors ties the terminal to the window even though the user merely wanted to open a document.

A **launcher** is the short-lived CLI process. A **desktop child** is a second execution of the same binary that owns the WebView. **Detachment** means it no longer shares the launcher's terminal session and standard streams; merely putting `wails.Run` in a goroutine would not accomplish this. A **single-instance handoff** is Wails forwarding the child's arguments to an already-running window and then exiting the redundant child.

Included:

- Background-by-default explicit `view`, including empty-window and dark-theme requests.
- Explicit foreground mode, useful for diagnostics and supervised execution.
- Correct argument and working-directory handling, platform detachment, logs, and launch errors.
- Existing single-instance behavior, tests, user docs, and production-build smoke validation.

Excluded:

- A daemon, PID registry, new IPC protocol, server, stop/status command, or process supervisor.
- Changing how Markdown, assets, recent files, live reload, or window deduplication work.
- A readiness protocol that waits for DOM rendering before returning.
- Fixing existing Linux D-Bus single-instance limitations.

## 3. Read the existing system from the outside inward

### 3.1 Entry point and argument parsing

At the pre-change baseline `d11c193`, `main.go:32-85` constructs Cobra root and `view` commands. `view` validates at most one positional file, binds `--dark`, and calls `runDesktop`. Root launch also calls `runDesktop`, without the view-specific theme flag. `main.go:90-148` resolves the file, initializes the application, and passes callbacks, embedded assets, menus, and the single-instance lock to Wails.

There are two parsers for different purposes. Cobra validates fresh user commands and rejects unknown flags. `cli.go:29` contains `ParseViewArgs`, a deliberately lenient parser for Wails-forwarded arguments. Do not replace Cobra with that parser: it skips unknown flags but does not understand their values. `cli.go:63`, `absolutizeFileArg`, resolves relative paths and rewrites `os.Args` because Wails forwards that vector verbatim. On macOS, the forwarded working directory can be the executable directory, so absolute arguments are necessary even if the first window could resolve a relative path.

### 3.2 Desktop lifecycle and state

`App` in `app.go:23-37` owns window-facing state: context, current file, theme, recent files, watcher, and image allow-list. `NewApp` initializes its maps and default theme. `Startup` (`app.go:53`) saves Wails' context, loads recents, subscribes to file drops, and starts fsnotify. Saving the runtime context is necessary because later Wails calls need it.

`runDesktop` sets `PendingOpen` and `PendingDark` before the DOM exists. `OnDomReady` (`app.go:82`) consumes those values, opens and renders the file, and emits `file-opened`. Trying to render into the frontend during CLI parsing would bypass this readiness boundary. `Shutdown` (`app.go:71`) persists recent files and closes the watcher; the desktop child must retain normal ownership of these callbacks.

`OnSecondInstanceLaunch` (`app.go:115`) parses forwarded args, resolves relative paths against the supplied directory when necessary, applies the theme, opens the requested file, and shows the existing window. Background launch must enter this same Wails machinery, rather than inventing a parallel socket protocol.

### 3.3 Rendering, assets, and live reload

`pkg/renderer/renderer.go:630`, `RenderBody`, produces the HTML fragment for a document. The renderer handles Markdown, highlighting, frontmatter, and relative image rewriting. The embedded frontend (`main.go:18`, `frontend/dist/app.js:69`) receives `file-opened` and swaps the content. `assets.go` restricts referenced-file reads to allowed directories.

`events.go:19`, `watchFile`, turns watcher notifications into `file-changed` events. `frontend/dist/app.js:102` responds by requesting a fresh render. `pkg/watcher/watcher.go:24,38,55,83` implements creation, registration, event processing, and shutdown. These components remain inside the desktop process. None needs to know whether a terminal launcher started that process.

```text
BEFORE
shell --waits--> Cobra view --> runDesktop --> Wails event loop
                                                |
                                                +--> App --> renderer
                                                +--> WebView <--> App events
                                                +--> watcher

AFTER: default view
shell --> launcher --> exec same binary, view --foregruond
             |                         |
             +--> print PID/log        +--> Wails / existing-instance handoff
             +--> return               +--> normal desktop lifecycle

AFTER: explicit foreground
shell --waits--> Cobra view --foregruond --> runDesktop --> Wails
```

### 3.4 Build boundaries

Read `AGENT.md` and `Makefile:24-54` before running anything. Production binaries must be built with `make build`, which generates CSS and invokes `wails build -tags webkit2_41 -s`. A plain `go build` binary can compile but refuses to launch because it lacks Wails production tags. Linux tests use `go test -tags webkit2_41 ./...` and require GTK/WebKit development libraries. The baseline command passed before implementation.

## 4. Gap analysis and invariants

The existing code has no separate launch decision, child process configuration, asynchronous diagnostic location, or tests for parent/child lifecycle. It already has the required desktop lifecycle and handoff parser, so the change should surround those APIs rather than rewrite them.

Preserve these invariants:

1. User arguments are validated before any process is spawned.
2. One default invocation spawns at most one desktop child.
3. Child arguments encode the parsed file/theme, not arbitrary raw parent arguments.
4. A file is absolute before crossing the process boundary.
5. The child retains the caller's working directory and environment, including DISPLAY and D-Bus variables.
6. No shell interprets file names, spaces, semicolons, or dollar signs.
7. Background child stdin is null; stdout/stderr are not terminal or pipe handles inherited from the invoking shell.
8. Foreground mode directly returns desktop errors; background mode only reports synchronous launch errors.
9. Bare launch and Wails callbacks keep their previous lifecycle.

## 5. Proposed implementation and API contracts

### 5.1 Testable Cobra construction

Extract `newRootCommand(desktop, background)` from `main`. Each parameter is a `func(string, bool) error`. Define flag variables inside the factory so repeated command construction in tests does not leak flag state. Root uses `desktop(file, false)`; `view` chooses between the injected functions after Cobra validates arguments. `main` supplies `runDesktop` and a wrapper around the launcher package.

This is a small dependency-injection seam, not a general command framework. It lets tests assert dispatch, files, themes, help, error propagation, invalid flags, and multiple positional arguments without opening windows.

```go
func newRootCommand(
    desktop func(string, bool) error,
    background func(string, bool) error,
) *cobra.Command
```

### 5.2 Isolated process-launch package

Add `internal/launch` with no Wails dependencies. Its exported surface is intentionally narrow:

```go
type Result struct {
    PID int
    LogPath string
}
func Start(file string, dark bool) (Result, error)
```

`Start` finds the current executable via `os.Executable`, resolves the optional file via `filepath.Abs`, chooses `os.UserCacheDir()/md-view`, creates that directory with mode 0700, and uses `os.CreateTemp` to create a mode-0600 `launch-*.log`. It constructs canonical arguments `view --foregruond [--dark] -- ABSOLUTE_FILE`. The separator protects flag-looking file names; absolute paths also make Wails' lenient handoff parser safe. An omitted file needs no separator or positional argument.

Use `exec.Command`, not `exec.CommandContext`: cancellation of the launcher must not kill an independently owned desktop. Leave `Env` and `Dir` unset to inherit them. A nil stdin makes Go connect it to the null device. Set stdout and stderr to the same regular log file; close the parent's descriptor after Start. Call `Process.Release` rather than `Wait`, because waiting defeats the requirement. Preserve the log on successful start; remove an unused log if process creation fails.

The CLI prints the child PID and log path. That PID may be short-lived if Wails immediately hands off to another process; it is diagnostic output, not a stable window identity. Logs may contain private file paths. Per-launch files avoid interleaved startup messages, but accumulate until users remove them; log rotation is outside this change.

### 5.3 Platform boundary

Add build-tagged process configuration files in the launch package:

- Linux and macOS: `syscall.SysProcAttr{Setsid: true}` creates a new session with no controlling terminal. Null stdin and log-backed stdout/stderr ensure terminal closure and shell pipelines do not retain the child's streams.
- Windows: `CreationFlags` combines `DETACHED_PROCESS` and `CREATE_NEW_PROCESS_GROUP`. Do not use a new console or depend on a shell's `start` built-in.
- Other targets: return an explicit unsupported-platform error rather than silently pretending to detach.

Session detachment does not protect against a login manager killing every process in a logout session, Windows job-object policies, or a machine shutting down. The claim is terminal-independent launch, not a service guarantee.

### 5.4 Control flow pseudocode

```text
execute view:
    Cobra validates flags and zero-or-one file
    if foregruond:
        return runDesktop(file, dark)
    result = launch.Start(file, dark)
    if error:
        return error
    print "started PID; log PATH"
    return success

launch.Start:
    executable = current executable
    file = absolute(file) if supplied
    logfile = private unique cache file
    child = exec.Command(executable, canonical foreground args)
    configure platform detachment
    child.stdin = null
    child.stdout = child.stderr = logfile
    start child, close parent log handle
    on start failure: remove unused log, return wrapped error
    save PID, release process handle
    return PID and log path

child:
    parses --foregruond
    enters runDesktop without launching again
    Wails either opens its own window or forwards to existing instance
```

## 6. Decision records

### Decision: re-exec the desktop instead of daemonizing in place

- **Context:** Wails owns a blocking native GUI loop and native threads.
- **Options considered:** goroutine, shell background syntax, Unix fork without exec, same-binary re-exec, separate daemon.
- **Decision:** same-binary `os/exec` launch.
- **Rationale:** avoids unsafe post-fork Go runtime use and platform-specific shell quoting; retains one deployable binary and existing Wails integration.
- **Consequences:** startup pays one extra process execution; successful parent exit cannot report later GUI failure.
- **Status:** accepted.

### Decision: literal requested foreground flag, explicit view only

- **Context:** the request names `md-view view` and spells the opt-in `--foregruond`.
- **Options considered:** silently correct the spelling, expose two aliases, use the requested spelling only.
- **Decision:** expose only `--foregruond`; leave bare launch direct.
- **Rationale:** fulfills the literal request without unrequested compatibility aliases or changing double-click/development behavior.
- **Consequences:** help and examples must make the spelling obvious; a future rename is an intentional public interface change.
- **Status:** accepted.

### Decision: exec acknowledgement, not GUI readiness

- **Context:** readiness can mean process created, Wails initialized, DOM ready, file rendered, or existing-instance handoff acknowledged.
- **Options considered:** pipe handshake, polling PID/window, wait briefly for exit, return after Start.
- **Decision:** return after successful Start and handle release; retain per-launch logs.
- **Rationale:** a readiness protocol adds cross-process synchronization and timeout policy disproportionate to shell detachment. A short sleep is not a reliable readiness test.
- **Consequences:** a missing display or bad file can fail after a successful command. Foreground is the synchronous diagnostic mode; logs cover child stderr, while UI file errors remain UI errors.
- **Status:** accepted.

## 7. Intern implementation phases

### P1: investigation and published guide

Create a task worktree from the clean current checkout, inventory entry points and lifecycle, run the baseline tests, write this guide and the chronological diary, relate source files, validate the ticket, and upload the guide before changing implementation. Print the overall plan and phase-start slip before investigation; print phase completion after publication. Commit the design separately so later readers can distinguish intended behavior from implementation outcomes.

### P2: implementation and focused verification

Before editing, print the phase-start slip. Extract the Cobra factory in `main.go`, add the launch package and platform files, and add focused tests. Do not modify rendering or callbacks unless a demonstrated invariant requires it. Update README and both user docs, including the old statements that there is no second process or background process. Keep the architecture picture explicitly scoped to the desktop process.

Test command dispatch with injected callbacks. Test argument generation for spaces, shell metacharacters, empty files, dark mode, relative paths, and files named `view` or beginning with a dash. Test real OS launch using a helper test process that writes its observed arguments/environment/session/stdin status, remains alive after the launch function returns, and exits when the test releases it. Test invalid executable errors and unused-log cleanup. Commit code and then diary/bookkeeping; print completion only after focused tests pass.

### P3: production validation and final delivery

Print the start slip, then run the full test suite, race tests, vet, and production Wails build. Cross-compile the isolated launcher tests for Windows and macOS; this validates APIs/build tags, not native runtime behavior. Launch the production binary in tmux: default command must return while a child lives, explicit foreground must keep the command waiting, help and malformed invocations must not spawn, and a relative file plus dark mode must survive re-exec. Inspect the native window if desktop tooling permits. Stop only test-owned processes and record any environment limitations honestly.

Finish the diary, relate all material code/docs to focused documents, run docmgr doctor, commit final changes, and upload a separate final bundle containing guide and diary (do not overwrite annotated initial delivery). Print the completion slip with the final commit reference.

## 8. Test strategy and acceptance matrix

- **Dispatch:** default view invokes background once; `--foregruond` invokes desktop once; `--foregruond=false` still backgrounds; root remains direct; dark flag survives; invalid input launches neither callback.
- **Arguments:** normalized absolute file follows `--`; caller spelling and arbitrary shell content stay a single argument; child always receives foreground true; empty-file launch is valid.
- **Lifecycle:** real helper PID exists when Start returns, is a new Unix session leader, sees EOF on stdin, writes to the private log, inherits cwd/environment, and terminates through test cleanup.
- **Failure:** executable/cache/log errors propagate, no spurious success output, no unused log after failed exec. Errors after exec are intentionally outside parent acknowledgement.
- **Regression:** existing `cli_test.go` and `openpath_test.go` continue passing; Wails-forwarded foreground flag is ignored by the lenient parser without becoming a file.
- **Production:** build with Wails, test parent return and foreground blocking, observe actual rendered file/theme if possible, document single-instance outcome separately from detachment.

Commands:

```bash
make test
go test -race -tags webkit2_41 ./...
go vet -tags webkit2_41 ./...
make build
GOOS=windows CGO_ENABLED=0 go test -c ./internal/launch -o /tmp/mdv-launch-windows.test.exe
GOOS=darwin CGO_ENABLED=0 go test -c ./internal/launch -o /tmp/mdv-launch-darwin.test
build/bin/md-view view --help
build/bin/md-view view --foregruond README.md
```

## 9. Operational guidance, risks, and migration

Scripts that previously relied on the command waiting must add `--foregruond`. `view` now returns launch diagnostics instead of carrying the desktop's eventual exit code. Terminal closure should not kill the detached desktop, but desktop logout may. Closing the window remains the normal way to stop the app; no daemon management commands are introduced.

If no window appears, read the printed log and retry with `--foregruond`. A successful launcher cannot promise that Wails found a display. File-read failures may be delivered to the frontend rather than stderr; do not advertise logs as a complete audit trail. Logs are private when created, but the cache directory may already exist with user-modified permissions. Do not truncate shared logs or change permissions on unrelated directories.

Potential review points are child stream ownership, session flags on each OS, accidental recursion, path normalization before forwarding, and confusion between spawned PID and existing-window PID. Native macOS/Windows checks remain necessary even if cross-compilation succeeds. Existing Linux single-instance behavior is explicitly best-effort and must not be conflated with a detachment regression.

## 10. API and file reference map

Repository references above use baseline `d11c193` line numbers; after edits, navigate by the named symbols.

- `main.go`: Cobra command construction, `runDesktop`, `singleInstanceID`, embedded assets and Wails options.
- `cli.go`: `ParseViewArgs`, `absolutizeFileArg`; `cli_test.go`: argument and forwarding regressions.
- `app.go`: `App`, `Startup`, `OnDomReady`, `OnSecondInstanceLaunch`, `Shutdown`, `openPath`.
- `events.go`, `pkg/watcher/watcher.go`: file-watch lifecycle and frontend notifications.
- `pkg/renderer/renderer.go`: `RenderBody`; `frontend/dist/app.js`: frontend event consumers.
- `assets.go`, `recent.go`: local-file allow-list and persisted recent files, unchanged by launch design.
- `Makefile`, `AGENT.md`: authoritative native build/test requirements.
- Planned `internal/launch/launch.go`: Start and canonical child args; platform files: session policy; tests: portable launch verification.
- Go API reference: https://pkg.go.dev/os/exec#Cmd.Start and https://pkg.go.dev/os#Process.Release — creation versus waiting/releasing.
- Go filesystem/process APIs: https://pkg.go.dev/os#Executable, https://pkg.go.dev/os#CreateTemp, https://pkg.go.dev/os#UserCacheDir.
- Platform process attributes: https://pkg.go.dev/syscall#SysProcAttr (fields differ by target OS).
- Cobra command contract: https://pkg.go.dev/github.com/spf13/cobra#Command.
- Wails options contract: https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options#SingleInstanceLock.


## 11. Implementation outcome and verified evidence

The implementation follows the proposed architecture: `main.go:newRootCommand` dispatches injected direct/background functions; `internal/launch/launch.go` implements `Start`, `childArgs`, and `start`; build-tagged detachment files select platform policy. No application callbacks or frontend rendering files changed. The production Wails build promoted existing `github.com/pkg/errors` and `golang.org/x/sys` dependencies from indirect to direct in `go.mod`; no versions changed.

Implementation commit is `2db896d`; reproducible native validation and dependency metadata are in `1721ce4`. The script `scripts/01-native-smoke.py` uses uniquely named tmux sessions, a temporary cache/config, and a document with a recognizable frontmatter title. It only closes its own test windows/processes. Run it from the repository root after `make build`. Its recorded output is `scripts/02-native-smoke-results.json`.

Verified on Linux:

- Full uncached tests, race tests, vet, ten repeated launcher-package test runs, and Wails production build passed.
- Windows and macOS launcher test binaries cross-compiled successfully with CGO disabled.
- Background production launcher returned zero while child PID 3353305 remained alive; SID was also 3353305.
- `/proc/PID/cmdline` showed `view --foregruond --dark -- /tmp/.../relative file.md` as distinct args.
- Child descriptors were `/dev/null` for stdin and the private log for stdout/stderr.
- The native window title was `md-view: MDV-BG-001 native smoke`, demonstrating that the absolute file reached rendering/frontmatter title handling.
- The desktop survived closing the launcher tmux session and then exited when its own window closed.
- Foreground mode did not return while its window was open and returned zero after the window closed.
- Help, extra positional args, and unknown flags created no new launch logs; invalid invocations returned nonzero.
- No task-owned application processes or tmux sessions remained after validation.

The smoke observes dark mode in child args, not rendered theme pixels. It does not establish Windows/macOS native session behavior or repeated-instance deduplication. Those remain explicit validation limitations, not claims inferred from Linux. Native startup emitted the nonfatal WebKit diagnostic `Overriding existing handler for signal 10. Set JSC_SIGNAL_FOR_GC if you want WebKit to use a different signal`. Wails binding generation also warned about unresolved standard-library model types (`url.Userinfo`, `big.Int`, `time.Time`, `x509.OID`), but the production build completed successfully. Neither warning prevented native file loading or normal shutdown.
