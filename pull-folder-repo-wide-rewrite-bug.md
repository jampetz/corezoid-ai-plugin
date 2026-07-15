# `pull-folder` rewrites files repo-wide, outside the pulled folder

## Summary

Calling the `pull-folder` MCP tool for one Corezoid stage (`folder_id=570300`, workspace
"Middleware") from a working directory that is the plugin repo root did not just export
that stage's ~466 processes — it also rewrote **68 pre-existing, git-tracked files** that
have nothing to do with that stage: plugin manifests, JSON schemas, `.github` config, and
several plugin sample/test files. 56 of those got pure cosmetic reformatting; 9 got a
`uuid` field silently stripped in addition. No process node data was lost (see Impact).

Discovered 2026-07-14 while verifying the new `corezoid-dependency-graph` skill
(Stage-Wide Mode) against a real pulled project. Reproduced once; not yet fixed or
reported upstream.

## Root cause

**Entry point** — `handlePullFolder` (`plugins/corezoid/mcp-server/mcp_handlers_process.go:98`):

```go
if err := downloadStageRecursively(v, folderID, "."); err != nil {   // line 110
```

The literal `"."` — i.e. `COREZOID_WORK_DIR`, the whole working directory — is passed as
the pull target, not an isolated per-pull scratch directory.

**The over-broad walk** — `downloadStageRecursively` (`plugins/corezoid/mcp-server/pull-project.go:153`)
unzips the pulled stage and merges its contents into that same `filePath="."` via
`moveContents` (line 228), then runs two post-processing passes rooted at that same `"."`:

```go
// pull-project.go:242, 246
err = renameFiles2Folders(filePath)       // walks "." recursively
...
err = formatJSONWithFallback(e, filePath) // walks "." recursively again
```

Both recurse via plain `os.ReadDir` with **no exclusion of dot-directories** — unlike an
earlier helper in the same file (`walkDepth`, used only to *locate* the stage dir) which
explicitly skips `strings.HasPrefix(e.Name(), ".")`. Because the pulled stage's own tree
is merged directly into `filePath` itself (nothing is left in an isolated subtree after
`moveContents`), the post-processing step has no way to tell "files I just wrote" apart
from "everything else already in the repo." It just formats every `.json` file under `.`.

**Unconditional read-modify-write** — `formatJSONWithFallback` (`pull-project.go:302-390`)
is generic, keyed only on file extension:

```go
if filepath.Ext(f.Name()) != ".json" { continue }
...
var dataRsp any
json.Unmarshal(dataJson, &dataRsp)
...
dataRspBin, _ := json.MarshalIndent(dataRsp, "", "  ")
os.WriteFile(filePath1, dataRspBin, 0644)
```

It always re-marshals and rewrites, even when nothing changed. Go's `encoding/json`
serializes `map[string]interface{}` keys alphabetically, and `MarshalIndent` never emits a
trailing newline — exactly the "keys reordered + trailing newline stripped" pattern seen
in all 56 unrelated files.

**Why `uuid` disappears** — explicit, unconditional deletes gated only on file *shape*,
not on "is this the file I just pulled":

```go
// pull-project.go:363-369 — inside scheme.nodes handling
for _, node := range nodes1 {
    if nodeMap, ok := node.(map[string]interface{}); ok {
        delete(nodeMap, "uuid")
    }
}
...
// pull-project.go:376-378
if d, ok := dataRsp.(map[string]interface{}); ok {
    delete(d, "uuid")
}
```

Any pre-existing `.json` file with a root-level `uuid` key, or a `scheme.nodes` array
whose node objects carry `uuid`, gets it stripped — regardless of whether that file was
part of this pull.

**Scope of the bug — exclusive to `pull-folder`.** `handlePullProcess`
(`mcp_handlers_process.go:36-94`, the single-file `pull-process` tool) resolves exactly
one target directory via `resolveFolderPathFromAPI` and writes exactly one file; it never
calls `renameFiles2Folders` or `formatJSONWithFallback`. Single-process pulls are unaffected.

**Unrelated to `_ALIASES_.json`** — no Go code generates or references that filename
(`grep -rn "_ALIASES_" --include="*.go"` finds nothing); it comes directly from the
server's ZIP payload and isn't part of this bug.

## Reproduction

1. Open a working directory that contains other `.json` files unrelated to Corezoid
   (e.g. this plugin's own repo root).
2. Run `pull-folder(folder_id=<any stage>)`.
3. `git status` / `git diff` afterward — every `.json` file anywhere under the working
   directory tree (including dot-directories) will have been touched, not just the files
   belonging to the pulled stage.

## Impact assessment

- **56 files, cosmetic only** — key reorder (alphabetical) + stripped trailing newline.
  No data change. Examples: `context7.json`, `.github/mlc_config.json`,
  `.claude-plugin/marketplace.json`, `.agents/plugins/marketplace.json`,
  `plugins/corezoid/.claude-plugin/plugin.json`, `plugins/corezoid/.codex-plugin/plugin.json`,
  `plugins/corezoid/.kiro-plugin/plugin.json`, `plugins/corezoid/.mcp.json`,
  `plugins/corezoid/.mcp.kiro.json`, `docs/corezoid-swagger.json`,
  `public/.well-known/skills/index.json`, all of `plugins/corezoid/mcp-server/json-schema/**/*.json`,
  several `plugins/corezoid/mcp-server/samples/*.json`, and the
  `plugins/corezoid/mcp-server/testdata/golden/layout_*.json` regression-test fixtures.
- **9 files, reshaped + `uuid` stripped** — node IDs, node counts, `obj_id`, `parent_id`,
  and `title` all confirmed byte-identical before/after (no content/data loss), but the
  top-level `uuid` field is gone and the file was re-serialized into the current
  canonical pull-output shape. Files: `plugins/corezoid/samples/create-user.json`,
  `plugins/corezoid/samples/database-call.json`, `plugins/corezoid/samples/delay-retry-pattern.json`,
  `plugins/corezoid/samples/error-handling.json`, `plugins/corezoid/samples/git-call.json`,
  `plugins/corezoid/samples/stripe-checkout.json`,
  `plugins/corezoid/samples/state-diagrams/user-status-driver-process.conv.json`,
  `plugins/corezoid/samples/state-diagrams/user-status-state-diagram.conv.json`,
  `plugins/corezoid/mcp-server/samples/reply_literal_values.json`.
- Most notable risk: the rewritten `mcp-server/testdata/golden/*.json` files are the
  layout engine's regression-test baselines (`plugins/corezoid/skills/corezoid-node-layout/SKILL.md`
  describes them as "golden coordinate files that freeze every fixture against unintended
  churn") — silently touching these could mask a real regression or cause spurious test
  diffs on next `go test`.
- All 65 non-intentional changes share the identical filesystem mtime (`2026-07-14
  18:46:54/55`), matching the exact moment the `pull-folder` call completed — confirming
  a single, atomic side effect of that one call, not separate unrelated edits.

## Suggested fix directions (for whoever picks this up)

- Extract the pulled stage's ZIP into an isolated temp directory and only move the
  resulting new files into the working tree — never merge directly into `"."`.
- Scope `renameFiles2Folders` / `formatJSONWithFallback` recursion to just the
  newly-extracted subtree (or track the exact list of files written by this pull and only
  post-process those).
- Make `formatJSONWithFallback` a no-op when the marshaled output is unchanged (aside from
  formatting) rather than always writing — or better, drop the blanket walk over
  pre-existing files entirely.
- If stripping `uuid` is intentional for freshly-pulled process nodes, scope that mutation
  to files this call actually wrote, not any file on disk that happens to match the shape.

## Status

- Working tree currently still has all 68 changes present — left as-is per a 2026-07-14
  decision, not yet reverted.
- Revert command, when ready (excludes the 3 `SKILL.md` files intentionally edited while
  building the `corezoid-dependency-graph` skill):
  ```bash
  git checkout -- \
    .agents/plugins/marketplace.json \
    .claude-plugin/marketplace.json \
    .github/mlc_config.json \
    context7.json \
    docs/corezoid-swagger.json \
    plugins/corezoid/.claude-plugin/plugin.json \
    plugins/corezoid/.codex-plugin/plugin.json \
    plugins/corezoid/.kiro-plugin/plugin.json \
    plugins/corezoid/.mcp.json \
    plugins/corezoid/.mcp.kiro.json \
    plugins/corezoid/mcp-server/json-schema/logics.json \
    "plugins/corezoid/mcp-server/json-schema/logics/*.json" \
    plugins/corezoid/mcp-server/json-schema/node.json \
    plugins/corezoid/mcp-server/json-schema/process.json \
    "plugins/corezoid/mcp-server/samples/*.json" \
    "plugins/corezoid/mcp-server/testdata/golden/*.json" \
    plugins/corezoid/mcp-server/testdata/layered_rail_pile.conv.json \
    plugins/corezoid/samples/api-post.json \
    plugins/corezoid/samples/corezoid-api-node-list.conv.json \
    plugins/corezoid/samples/create-actors.json \
    plugins/corezoid/samples/create-user.json \
    plugins/corezoid/samples/database-call.json \
    plugins/corezoid/samples/delay-retry-pattern.json \
    plugins/corezoid/samples/error-handling.json \
    plugins/corezoid/samples/git-call.json \
    plugins/corezoid/samples/stripe-checkout.json \
    "plugins/corezoid/samples/state-diagrams/*.conv.json" \
    public/.well-known/skills/index.json
  ```
- Not yet reported to the Corezoid team via `send-feedback` — pending.

## References

| Location | Role |
|---|---|
| `plugins/corezoid/mcp-server/mcp_handlers_process.go:98` | `handlePullFolder` — call site passing `"."` as target |
| `plugins/corezoid/mcp-server/mcp_handlers_process.go:36-94` | `handlePullProcess` — single-file pull, unaffected (for comparison) |
| `plugins/corezoid/mcp-server/pull-project.go:153` | `downloadStageRecursively` |
| `plugins/corezoid/mcp-server/pull-project.go:228` | `moveContents` — merges pulled stage into `filePath` itself |
| `plugins/corezoid/mcp-server/pull-project.go:242` | call to `renameFiles2Folders(filePath)` |
| `plugins/corezoid/mcp-server/pull-project.go:246` | call to `formatJSONWithFallback(e, filePath)` |
| `plugins/corezoid/mcp-server/pull-project.go:302-390` | `formatJSONWithFallback` body — generic unconditional rewrite + `uuid` strip |
