---
Title: Background view launch with explicit foreground mode
Ticket: MDV-BG-001
Status: complete
Topics:
    - cli
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Background-by-default view implemented and Linux native lifecycle verified.
LastUpdated: 2026-09-05T17:29:44.087935264-04:00
WhatFor: ""
WhenToUse: ""
---

# Background view launch with explicit foreground mode

## Overview

Explicit `md-view view` now launches a terminal-independent copy of the same Wails binary and returns a child PID plus private log path. `--foregruond` keeps it attached; bare `md-view` remains a direct desktop launch. No daemon or new IPC protocol is introduced.

- Worktree: `/home/manuel/code/wesen/go-go-golems/md-view-background`
- Branch: `task/background-view`
- Base: `d11c193`; design: `33c35b3`; implementation: `2db896d`; native validation: `1721ce4`.

Tests, race, vet, production build, repeated helper launch, and Linux native window/terminal smoke pass. Windows/macOS launcher cross-compilation passes; native runtime checks on those systems remain unverified. GUI theme pixels and repeated-instance deduplication were not separately validated; existing Wails behavior is unchanged.

## Key Links

- [Intern analysis, design, and implementation guide](design-doc/01-background-launch-intern-guide.md)
- [Detailed chronological diary](reference/01-diary.md)
- [Reproducible native smoke](scripts/01-native-smoke.py)
- [Native evidence JSON](scripts/02-native-smoke-results.json)
- reMarkable: `/ai/2026/09/05/MDV-BG-001` (initial guide and final guide/diary bundle).
- Source relations live on the focused guide and diary.

## Status

Current status: **complete**. Initial guide and final guide/diary bundle uploaded successfully after dry-runs. All seven requested slips printed: overall plan, and start/done for each of three phases. Doctor passed. Changes committed in the task worktree; not pushed or merged.

## Topics

- cli

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and intern implementation guide
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
