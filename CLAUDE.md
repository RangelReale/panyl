# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Panyl ("Parse ANY Log") is a Go library (module `github.com/RangelReale/panyl/v2`, Go 1.23) that parses log streams containing mixed formats (e.g. multiple services interleaved in one file) through a chain of plugins. This repo holds only the core engine plus a few generic plugins; format-specific parsers (Go/Ruby/Mongo/NGINX logs, docker-compose metadata, etc.) live in the separate `panyl-plugins` repo, and the CLI lives in `panyl-cli`.

## Commands

- Build: `go build ./...`
- Test all: `go test ./...` (also `task test` via Taskfile)
- Single test: `go test -run TestProcessor_CreatePlugin .`
- CI (`.github/workflows`) runs `go build -v ./...` and `go test -v ./...` on Go 1.23. Releases are made by pushing a git tag (`task release-version VERSION=vX.Y.Z`), which triggers goreleaser.

## Architecture

- `Processor` (`processor.go`) holds registered plugins, bucketed by type. `RegisterPlugin` type-asserts a single `Plugin` against every plugin interface, so one struct can implement several stages (e.g. `plugins/metadata.ForceApplication` is both `PluginMetadata` and `PluginSequence`). Every plugin must implement the marker method `IsPanylPlugin()`.
- `Processor.Process`/`ProcessProvider` creates a `Job` per run and feeds it lines from a `LineProvider` (`line.go`/`line_impl.go`). A line can be a `string` or a pre-built `*Item` (for structured sources).
- `Job` (`job.go`) is where the pipeline lives. Key mechanics that span the plugin interfaces:
  - `*Item` lines are copied on input (the caller's instance is never modified). Unmatched lines accumulate in a backlog (`Job.lines`). `PluginStructure` and then `PluginParse` are tried against progressively larger *suffixes* of the backlog (`lines[curline:]`, from the newest line backwards), which is how multi-line structures (e.g. pretty-printed JSON) are detected. The first match wins. After each failed attempt the item is restored from a snapshot, so a plugin's changes only stick when it returns true.
  - On a match, all backlog lines *before* the match are flushed via `processResultLines`, which offers them to `PluginConsolidate` (which must consume from the top and returns `topLines`), falling back to emitting each line as its own item.
  - With no match, `PluginSequence.BlockSequence` on the last two lines can force a flush of the older lines. The backlog is also force-flushed past `MaxBacklogLines` (default 50) and at `Finish`.
  - `outputItem` runs `PluginParseFormat` (only if `MetadataFormat` isn't set), then `PluginPostProcess` (sorted by `PostProcessOrder()`, 0–10, default 5), fills a missing `MetadataTimestamp` from the last seen timestamp (marking `MetadataTimestampCalculated`), drops items with `MetadataSkip`, and wraps `Output.OnItem` with `PluginCreate.CreateBefore/CreateAfter` (created items get `MetadataCreated` and post-processing, but do not recursively trigger Create). If `OnItem` returns false, the job stops (`ErrFinished`) and the backlog is not flushed.
  - The full stage order is documented in README.md under "Plugin execution order"; keep it in sync when changing `job.go`.
- `Item` (`item.go`) is the unit passed through the pipeline: `Metadata` and `Data` are `MapValue` maps that are always non-nil; `Line` is the unparsed remainder (plugins clear it when they consume the whole line); `Source`/`RawSource` are only populated with `WithIncludeSource(true)`. Structure/parse plugins should call `item.MergeLinesData(lines)` to carry over metadata from the consumed backlog lines.
- Well-known metadata keys are constants in `metadata.go` (`MetadataTimestamp` must hold a `time.Time`).
- `Output` (`output.go`) needs `OnItem`, `OnFlush`, `OnClose`; `output_impl.go` provides `OutputFunc`, `OutputArray` (handy in tests), and `OutputNull`.
- Options: `Option` configures the `Processor` (`WithPlugins`, `WithDebugLog`, `WithOnJobFinished`); `JobOption` configures a single run (`WithLineLimit`, `WithMaxBacklogLines`, `WithIncludeSource`). Returning `ErrFinished` from a job stops processing cleanly.
- `plugins/` contains the in-repo generic plugins (`clean.AnsiEscape`, `structure.JSON`, `consolidate.JoinAllLines`, `metadata.ForceApplication`); follow their shape (value receiver, `var _ panyl.PluginX = T{}` assertion, `IsPanylPlugin()`) when adding new ones.
