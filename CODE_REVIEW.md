# Code Review — panyl core

Scope: the whole repository at `7447f10` (core engine in the package root, `plugins/`, `util/`).
`go build ./...`, `go vet ./...` and `go test ./...` all pass. Findings marked **(verified)** were reproduced
with throwaway tests. Those tests were not committed.

Severity: 🔴 bug with user-visible impact · 🟠 robustness / API issue · 🟡 minor / cleanup.

---

## 🔴 Bugs

> **Status:** items 1 to 9 are fixed, with regression tests in `job_test.go`, `value_test.go` and
> `plugins/structure/json_test.go`.

### 1. Structured `*Item` lines with only `Data` are silently dropped (verified)
`job.go:93-98`: after the Clean plugins run, every item is dropped if `strings.TrimSpace(process.Line)` is empty.
But `LineProvider.Line()` says of `*Item` lines: *"usually only Item.Data should be filled"*
(`line.go:16`). An item such as `InitItem(WithInitCustom(func(i *Item){ i.Data["a"] = 1 }))` never
reaches the output: there is no error and no output item.

**Fix:** for `*Item` input, skip only when `Line` is empty **and** `Data` and `Metadata` are empty. You could also skip the
empty-line check completely for `*Item`.

### 2. A consolidate plugin returning `topLines <= 0` hangs forever (verified)
`job.go:271`: the check covers only the upper bound. If `Consolidate` returns `(true, 0, nil)`, `startLine += 0`
and the `for startLine < len(lines)` loop never ends. A negative value moves `startLine` backwards.

**Fix:** reject `topLines < 1` with an error, just as too-large values are rejected. Also fix the typo
"requestd" and use a lowercase error string (`job.go:272`).

### 3. `WithLineLimit` outputs one line too many (verified)
`job.go:54-60`: with `WithLineLimit(2, 2)` over lines `1..6`, the output is `[2 3 4]`. The condition
`p.lineno > p.StartLine+p.LineAmount` should be `>=`. You could also count lines emitted instead.
The README example `WithLineLimit(0, 100)` therefore processes 101 lines.

### 4. Errors from `ProcessLine` skip `Finish`: the backlog is lost and the output is never flushed or closed (verified)
`processor.go:76-88`: if a plugin returns an error (or `scanner.Err()` is non-nil), `ProcessProvider` returns
early. `job.Finish` is never called, so:
- lines already in the backlog are never emitted;
- `Output.OnFlush` and `Output.OnClose` are never called, so network and file outputs leak.

In the verified case, 2 buffered lines were lost and `OnFlush` was never called. Decide on the intended behavior. At a minimum, call
`OnClose` in a `defer`, and consider flushing the backlog before you return the error.

### 5. Unchecked `time.Time` type assertions panic on bad plugin data (verified)
`job.go:166` and `job.go:339` use `pts.(time.Time)` without a check. A plugin that stores the timestamp as a
string (a common mistake when you copy it from JSON) causes the panic `interface {} is string, not time.Time`, which
takes down the whole process. Use the comma-ok form and either ignore the value or return a descriptive error.

### 6. Created items with nil `Metadata` panic (verified)
`job.go:361`: `item.Metadata[MetadataCreated] = true` panics with *"assignment to entry in nil map"* if a
`PluginCreate` returns `&Item{...}` instead of using `InitItem`. Call `p.ensureItem(item)` (or the equivalent)
before you write to the map.

### 7. `MapValue.MapValue()` never matches a nested `MapValue` (verified)
`value.go:119-128`: the type switch has only `case map[string]any`. A value whose dynamic type is the named type
`MapValue` does not match that case, so `MapValue{"x": MapValue{...}}.MapValue("x")` returns `nil`. Add
`case MapValue:`.

### 8. `ListValueAdd`, `ListValue` and `ListValueContains` disagree on single-string values (verified)
- `ListValue` treats a plain `string` as a one-element list, but `ListValueContains` does not:
  `MapValue{"l":"a"}.ListValueContains("l","a") == false`.
- `ListValueAdd` removes duplicates for `[]string` but not for `string`: adding `"a"` to `"a"` gives `["a","a"]`.
- None of them handle `[]any`, which is what `encoding/json` produces. This affects `MetadataExtraCategories`
  and similar keys when they come from JSON.

### 9. `structure.JSON` accepts trailing `}` or `]` garbage (verified)
`plugins/structure/json.go:25`: `jdec.More()` returns `false` when the next token is `}` or `]`, so
`{"a":1} }` or `{"a":1}]` count as a full match and the extra characters are silently lost. Instead, check that a
second `Decode` returns `io.EOF`. You could also check that `InputOffset()` plus the trailing whitespace covers the whole input.

---

## 🟠 Robustness / API issues

> **Status:** items 10 to 19 are fixed, with regression tests in `robustness_test.go` and
> `plugins/structure/json_test.go`. For item 15, `UseNumber` is an opt-in field on `structure.JSON`, so existing
> users keep `float64` numbers.

10. **`Output.OnItem`'s `cont bool` result is ignored** (`job.go:383`). The interface suggests that an output
    can stop processing, but the return value is discarded. Either honor it (map `false` to `ErrFinished`) or
    remove it from the interface.

11. **An unknown line type causes a nil-pointer panic** (`job.go:66-83`). If `line` is neither a `string` nor an `*Item`
    (for example `nil`, which `StaticLineProvider.Line()` returns on error), `process` is nil and the Clean loop
    dereferences it. Add a `default:` case that returns an error. `StaticLineProvider` also sets `err`
    only inside `Line()`, after `Scan()` has already returned `true`, so the error is never seen in time.

12. **The context is never checked.** `ReaderLineProvider.Scan` ignores `ctx`, and `ProcessProvider` never checks
    `ctx.Done()`. Cancelling a long `Process` over a pipe or file has no effect. Check `ctx.Err()` once per
    loop iteration.

13. **`onJobFinished` errors are discarded** (`processor.go:91`). The callback signature returns `error`, but the
    result is ignored with `_ =`. Propagate it, or change the signature to return nothing.

14. **A failed match leaves partial mutations on the item.** The same `process` item (which is also
    `p.lines[len-1]`) is passed to every Structure and Parse attempt for every backlog suffix. If a plugin changes
    `item` and then returns `false`, the change leaks into later attempts and into the backlog. Document
    this rule clearly ("only mutate on success"), or pass a scratch clone and commit it only on success.

15. **Merge precedence in `structure.JSON`.** `MergeLinesData` runs first, and then `mergo.Map(&item.Data, jdata)`
    runs **without** `mergo.WithOverride`. When keys collide, data from the backlog lines wins over the freshly
    parsed JSON. That behavior is probably unintended. JSON numbers also decode as `float64`, which loses precision for large
    IDs and nanosecond timestamps. Consider `UseNumber()`.

16. **`lastTime` updates are inconsistent** (`job.go:171`). In the match branch, the timestamp returned by
    `processResultLines` for the flushed lines is discarded (`_, err =`). If the matched item has no
    timestamp, it inherits a stale `lastTime` instead of the timestamp of the lines just before it. Also, the
    `create` path passes `lastTime` rather than `retTime` (`job.go:362`), so `CreateAfter` items get the
    *previous* item's timestamp instead of the current one.

17. **`Item.Clone` / `CloneData` are shallow.** `mergo.Map` copies nested maps and slices by reference. A later
    `ListValueAdd` (which uses `append`) on the clone can modify the original's backing array. Either document
    this or deep-copy the values.

18. **Locking is inconsistent.** `ProcessLine` takes `p.m`, but `Finish` does not. Calling `Finish` twice also
    re-emits the backlog, because `p.lines` is never cleared.

19. **Caller items are mutated.** For `*Item` input, `process.LineNo` is overwritten and `RawSource` is cleared
    in place (`job.go:72`, `job.go:254-256`). That can surprise callers who reuse items.

---

## 🟡 Minor / cleanup

- `job.go:363-367`: the nested `if err != nil { if err != nil { ... } }` check is duplicated.
- `job.go:319`: the doc comment on `internalOutputItem` says `outputItem`.
- `processor.go:65`: the doc comment on `Process` says "Item reads lines from an [io.Reander]", which has two typos.
- `plugin.go:27`: the comment has the typo "Metdatada".
- `debug_log.go:9`: the comment has the typo "receiced".
- `slog.go:11`: the comment names `LoggerCtxKey`, but the constant is `slogLoggerCtxKey`.
- `debug_log_impl.go`: the `rawLine` argument of `LogSourceLine` is never used. `IncludeSource` has no constructor option.
- `value.go`: `FloatValue` has a redundant `float64(vv)` for `float64`. `IntValue` silently overflows on
  `uint64`. `BoolValue` handles `int` but not the other integer kinds.
- `getSortedPluginPostProcess` could be replaced by a copy plus `sort.SliceStable`. Orders outside `0..10` are not
  clamped or rejected, although the docs call these values limits.
- `util.AnsiEscapeString`: the regexp works on runes, so a raw `0x9B` byte (C1 CSI in 8-bit encodings) is not matched.
  Returning `(false, "")` instead of the unchanged string is an unusual contract.
- Error strings are sometimes capitalized (`"Error merging structs: %v"`) and use `%v` instead of `%w`
  (`item.go:72,74`, `json.go:39`).
- Dependencies: `github.com/imdario/mergo v0.3.12` has been renamed to `dario.cat/mergo`, and `testify v1.7.1` is old. Consider updating both.
- README still uses the old names `PostProcessOrder_Default`, `PostProcessOrder_First` and `PostProcessOrder_Last` (around line 203).
  The code uses `PostProcessOrderDefault` and the related constants.
- `interface{}` and `any` are mixed across files. Pick one.

---

## ⚡ Performance

For every unmatched line, the job tries every Structure and Parse plugin against every backlog suffix, up to
`MaxBacklogLines` = 50. Each attempt rebuilds the joined string (`lines.Line()`) and, for `structure.JSON`, runs
a full JSON decode. Non-JSON-heavy logs therefore pay about 50 string joins and 50 failed decodes per line.
Cheap improvements:
- In `structure.JSON`, reject quickly unless the first line starts with `{` and the last line ends with `}`.
- Cache the joined suffixes for each `ProcessLine` call, or build them incrementally from the tail.

---

## 🧪 Test coverage

Tests exist only for plugin registration, Create ordering, PostProcess ordering and the line providers. There are no
tests for:
- multi-line structure detection and backlog flushing (the core feature);
- `PluginSequence`, `PluginConsolidate`, `MaxBacklogLines`, `WithLineLimit` and `WithIncludeSource`;
- timestamp propagation and `MetadataSkip`;
- any of the in-repo plugins (`plugins/*` and `util` have no test files);
- `MapValue` helpers.

Most of the bugs above would have been caught by small table tests in these areas. Start with regression tests
for items 1 to 9.
