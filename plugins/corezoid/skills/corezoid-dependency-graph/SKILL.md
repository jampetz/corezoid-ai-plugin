---
name: corezoid-dependency-graph
description: >
  Recursive, multi-level Corezoid dependency crawler and interactive HTML
  visualizer. Two modes: Single-Process (crawls one process's full outbound
  call tree via api_rpc / api_copy, across project/stage boundaries) and
  Stage-Wide (graphs an entire pulled stage/project in one pass from local
  files, no per-process pulls). Produces a zoomable/searchable vis-network
  graph with depth filtering, color-by-depth or color-by-alias, and two node
  size modes selectable in the viewer itself: by node count (internal
  complexity) or by reference count / fan-in (how many other processes call
  it, by numeric conv_id or @alias alike) to surface the biggest dependencies
  at a glance. Use as an optional deeper dive after /corezoid-review or
  /corezoid-project-review. Activate when the user says "dependency graph",
  "map all dependencies", "full dependency tree", "recursive dependency
  crawl", "interactive graph", "visualize dependencies", "show everything
  this process calls", "graph the whole call tree", "graph the whole
  project", "map this stage", "whole-project dependency graph", "biggest
  dependencies", or "most referenced processes".
---

# Corezoid Dependency Graph

You build a **recursive, multi-level** map of everything one or more processes call
(directly and transitively) and hand the user an interactive HTML viewer to explore it.

This skill is a standalone, optional deeper dive — it does not replace or restructure:
- `corezoid-review` Step 12, which builds a static **1-level** Mermaid diagram of direct
  dependencies only
- `corezoid-project-review` Step 2.1, which graphs dependencies **within one project's
  inventory** only

Reach for this skill when the user wants to see the *full* call tree across process and
project boundaries, explored interactively rather than read as a static diagram.

---

## Step 0: Choose Scope

Two mutually exclusive modes:

| The user gives you... | Mode |
|---|---|
| A specific process — file path, name, or ID | **Single-Process Mode** (Step 1 below) — crawl outward from that one root, across project/stage boundaries. |
| "the whole project", "this stage", "map the whole stage", or a bare `stage_id`/`folder_id` with no process named | **Stage-Wide Mode** (see below) — graph every process already pulled for that stage in one pass, no per-process pulls needed. Input is a **single parameter** — the `stage_id`/`folder_id` — the same shape as `pull-folder(folder_id=<id>)`. Default to `COREZOID_STAGE_ID` from `.env` if the user doesn't name a stage explicitly. |

If it's ambiguous which the user means, ask.

---

## Step 1: Identify the Process (Single-Process Mode — MANDATORY FIRST STEP)

Resolve the root process before doing anything else:

1. Check whether a process identifier — file path, name, or ID — was already given in
   this message or established earlier in the conversation (e.g. a `PROCESS_PATH` or
   `conv_id` from a prior `/corezoid-review` or `/corezoid-project-review` run in this
   session). If so, reuse it — don't re-ask.
2. Otherwise ask:

   > "Please specify the root process — a file path (e.g.
   > `1278273_Business.folder/2778176_payment.conv.json`), a process name, or a
   > process ID."

   Do **not** proceed until an identifier is provided.
3. If given a name or numeric ID (not a path), resolve it to a local file with `find`/`grep`
   (the project is assumed already pulled locally).

---

## Step 2: Crawl Configuration

No confirmation prompt is needed to start — same convention as `corezoid-project-review`
Phase 0 ("process all sizes automatically"). State the default cap up front:

> Crawling recursively, up to **150 processes** (root included). Say "raise the cap to
> N" if you need more.

150 is sized above the largest real crawl documented so far (138 processes / 290 edges /
5109 nodes for one production root) while still bounding worst-case blast radius (e.g. a
shared utility process reachable from most of a workspace).

---

## Step 3: Resolve the Alias Index (once per crawl)

Before crawling, resolve the stage's alias list **once** so every `@alias` conv_id
encountered during the crawl can be mapped to a numeric process ID without a repeat API
call per reference.

**Check for a local file first** — `pull-folder` writes a flat `_ALIASES_.json` at the
root of the pulled stage (sibling to the `*.stage.json` file), one entry per alias:

```json
{
  "obj_id": 62203,
  "obj_to_id": 1508234,
  "obj_to_title": "Send Escalation (Telegram)",
  "obj_to_type": "conv",
  "short_name": "tg-msg",
  "title": "tg-msg"
}
```

If `_ALIASES_.json` exists locally (the stage was already pulled this session or earlier),
read it directly — **no API call needed.** Only fall back to a live API call (same request
as `corezoid-alias-manager` "Workflow: List aliases") if no local `_ALIASES_.json` is found
(e.g. a single process was pulled with `pull-process` rather than a full `pull-folder`):

```
POST {COREZOID_API_URL}/api/2/json
Authorization: Simulator {ACCESS_TOKEN}
Content-Type: application/json

{
  "ops": [{
    "type": "list",
    "obj": "aliases",
    "sort": "date",
    "order": "desc",
    "id": "<WORKSPACE_ID>",
    "company_id": "<WORKSPACE_ID>",
    "project_id": <PROJECT_ID>,
    "stage_id": <STAGE_ID>
  }]
}
```

`WORKSPACE_ID` / `COREZOID_API_URL` / `ACCESS_TOKEN` come from `.env` /
`~/.corezoid/credentials`; `project_id` from the `*.stage.json` file's `parent_id` — see
`corezoid-alias-manager/SKILL.md` § "Resolving environment values" if any are missing.

Either way, build two in-memory maps for the whole crawl from the `short_name`/`obj_to_id`
pairs (local file) or `short_name`/`obj_to_id` fields (API response — same names):
- `alias_by_name`: `short_name` → `obj_to_id` (resolves `@alias` conv_ids to a numeric pid)
- `alias_by_pid`: `obj_to_id` → `short_name` (reverse index, for `hasAlias`/`label` per node)

If both the local file is missing and the API call fails (auth, network, missing
`project_id`), **do not abort the crawl** — proceed with both maps empty and flag in the
final report that alias resolution was unavailable, so every `@alias` reference surfaces
as unresolved instead of silently becoming a broken edge.

---

## Step 4: Recursive Crawl (breadth-first)

Crawl breadth-first, not depth-first: if the cap cuts the crawl short, a BFS crawl leaves
a wide, mostly-complete shallow picture instead of one fully-expanded deep branch with
its siblings missing entirely.

```
visited   = {}              # resolved numeric pid -> node record
edges     = set()           # (from_pid, to_pid) unique pairs
unresolved = []             # dynamic {{...}}, unresolved @alias, failed pulls
queue     = deque([root_pid])
queued    = {root_pid}
processed = 0
cap       = 150             # or user-raised value

while queue and processed < cap:
    pid = queue.popleft()
    if pid in visited: continue

    path = locate_or_pull(pid)          # see "Pulling a dependency" below
    if path is None:
        visited[pid] = stub_node(pid, reason="pull failed — access denied or not found")
        processed += 1
        continue

    nodes = read_conv_json(path)        # scheme['nodes'] — the on-disk .conv.json has no
                                         # ops[] wrapper; scheme is a dict directly, not a
                                         # list (verified against real pull-folder output)
    title = process_title(path)
    alias = alias_by_pid.get(pid)
    visited[pid] = {
        "id": pid,
        "label": alias if alias else title,
        "title": title,
        "hasAlias": bool(alias),
        "nodeCount": len(nodes),
    }
    processed += 1
    if processed % 10 == 0:
        report_progress(processed, cap, len(queue), title, pid)

    for lg in all_logics(nodes):
        if lg["type"] not in ("api_rpc", "api_copy"):
            continue
        kind, target = classify_conv_id(lg["conv_id"], alias_by_name)
        if kind in ("dynamic", "alias_unresolved"):
            unresolved.append({"from": pid, "raw": lg["conv_id"], "kind": kind})
            continue
        edges.add((pid, target))
        if target not in visited and target not in queued:
            queue.append(target)
            queued.add(target)

cap_hit = bool(queue)   # unvisited pids remained when the loop ended
```

`classify_conv_id(conv_id, alias_by_name)`:

| Raw `conv_id` | Classification | Result |
|---|---|---|
| Integer, or numeric-looking string (`"21123"`) | `numeric` | resolved pid directly |
| String starting with `@` | `alias` | look up in `alias_by_name`; not found → `alias_unresolved` |
| String containing `{{` | `dynamic` | cannot be statically resolved (per `docs/nodes/call-process-node.md`) |

**On-disk file shape** (verified against real `pull-folder`/`pull-process` output — this
differs from the raw API response shape some other docs assume): a pulled `.conv.json` has
no `ops[]` wrapper. `scheme` is a **dict** directly at the top level, holding `nodes` and
`web_settings` — read nodes as `raw['scheme']['nodes']`, not `ops[0]['scheme'][0]['scheme']['nodes']`.
`node['condition']['logics']` is where `api_rpc`/`api_copy` entries live — never
`node.get('logic', [])`.

**Scope note:** `conv[@alias].ref[...]` state-store reads are a *different* kind of
dependency (a data read, not a process call) and are **out of scope for graph edges** in
this skill — they are not crawled and do not appear as edges. `corezoid-review` Step 10
already tracks these separately.

**De-duplication guarantee:** `visited` is keyed by the **resolved numeric pid**, never
by the raw `conv_id` string — a process reached once via `@payment-checkout` and again
via its numeric id `21123` is pulled exactly once and appears as exactly one node. Edges
are stored as a de-duplicated set of `(from, to)` pairs — one edge per unique pair
regardless of how many call sites in the source process target it.

### Pulling a dependency

Before pulling, check whether a local file matching `**/<pid>_*.conv.json` already
exists — if so (same crawl session, e.g. after a cap-raise re-run) reuse it instead of
re-pulling. On a fresh or later invocation, pull fresh (the process may have changed).

Prefer the MCP server's CLI mode for the crawl loop, since it may run tens to 100+ times
and an agent-orchestrated tool call per dependency is comparatively slow:

```bash
sh "${CLAUDE_PLUGIN_ROOT}/mcp-server/run.sh" pull-process process_id=<pid>
```

Parse the returned `Process <pid> saved to <path>` message for the file location. Fall
back to the agent-orchestrated `pull-process(process_id=<pid>)` MCP tool call (same
return message format) if CLI mode is unavailable in the environment.

### Progress reporting

Log every 10 processes — same "batches of 10" convention as `corezoid-project-review`
Phase 0:

> Crawled 40/150 (queue: 12 remaining): "Send OTP" (conv_id 21940)

### Cap hit

Stop immediately when the cap is reached — do not drain the rest of the queue. For every
pid still queued or that failed to pull, emit a **stub node** (`nodeCount: 0`,
`hasAlias: false`, a descriptive `title`/`label` like `"(not crawled — cap reached)"` or
`"(pull failed — access denied)"`) so that edges referencing it are not silently dropped
by the viewer (`dependency_graph.html` drops any edge whose `from`/`to` has no matching
node). Report clearly:

> Stopped at 150/150 processes — 23 still queued. Say "raise the cap to 250" to continue
> (already-pulled processes are reused, not re-fetched).

---

## Step 5: Write the Output JSON

Write to the current working directory as `dependency_graph_<rootpid>.json`
(overwriting on re-run — no timestamp suffix needed, `<rootpid>` is already the natural
unique key per root). Exact schema expected by the bundled viewer:

```json
{
  "rootPid": 21939,
  "rootTitle": "Process Name",
  "nodes": [
    {"id": 21939, "label": "Process Name", "title": "Process Name", "hasAlias": true, "nodeCount": 42}
  ],
  "edges": [
    {"from": 21939, "to": 21940}
  ]
}
```

Field notes:
- `label` — alias `short_name` (no leading `@`, the viewer adds it) when `hasAlias` is
  true, else the process title.
- `title` — always the real process title (used in the hover tooltip and search).
- `nodeCount` — that process's **own** node count, not cumulative subtree size.
- `rootPid` must be a JSON number, matching `nodes[].id` type.

---

## Step 6: Report and Point to the Viewer

```markdown
## Dependency Graph

Generated: `dependency_graph_21939.json` (138 processes, 290 edges, 5109 total nodes)
Unresolved references: 3 dynamic `{{...}}` conv_ids, 1 unresolvable `@alias` (see below)

Open `${CLAUDE_PLUGIN_ROOT}/skills/corezoid-dependency-graph/assets/dependency_graph.html`
in a browser (needs one-time internet access to load the vis-network script) and load the
JSON via drag-and-drop or the "📂 Load JSON" file picker. Use the **Size** dropdown to
switch lenses:

- **By node count** (default) — each dot sized by that process's own internal complexity.
- **By references** — each dot sized by how many other processes call it (fan-in, counting
  numeric `conv_id` and `@alias` references the same way). Switch to this to spot the
  biggest dependencies at a glance — the most depended-upon processes render as the largest
  dots, since breaking one has the widest blast radius.
```

If `unresolved` is non-empty, add:

```markdown
### Unresolved References

| From (pid / title) | Raw `conv_id` | Reason |
|---|---|---|
| 21939 / "Payment Init" | `{{target_process}}` | dynamic — cannot be statically resolved |
| 21942 / "Refund Flow" | `@old-refund-service` | alias not found in project alias list |
```

If alias resolution failed entirely in Step 3, add a note that `hasAlias`/`label` reflect
numeric IDs only for this run.

---

## Stage-Wide Mode (whole project / whole stage)

Use this instead of Step 1 when Step 0 selected Stage-Wide Mode. The input is a single
`stage_id`/`folder_id` — mirroring `pull-folder(folder_id=<id>)`, which is exactly why this
mode doesn't need a BFS/pull loop: that tool already fetches every process in the stage in
one recursive call, so the graph can be built from local files in a single pass.

### Stage Step 1: Ensure the stage is pulled locally

If the stage isn't already pulled (no local `.conv.json` files under it), call
**`pull-folder(folder_id=<stage_id>)`** once. If it's already pulled (e.g. from an earlier
`corezoid-init` or `corezoid-project-review` run this session), reuse those local files —
don't re-pull.

### Stage Step 2: Build nodes directly from local files — no per-process pulls

The key difference from Single-Process Mode: since `pull-folder` already fetched every
process in scope, there is no queue and no repeated `pull-process` calls. One pass:

```
local_files = find <stage_dir> -name "*.conv.json"    # every process already in scope
local_pids  = {extract_pid(f): f for f in local_files}

nodes = {}
for pid, path in local_pids.items():
    scheme_nodes = read_conv_json(path)    # same indexing as Single-Process Mode
    alias = alias_by_pid.get(pid)          # same one-time alias resolution as Step 3
    nodes[pid] = {
        "id": pid, "label": alias or process_title(path), "title": process_title(path),
        "hasAlias": bool(alias), "nodeCount": len(scheme_nodes),
    }
```

Resolve the alias index the same way as Step 3 above (one API call, once, before the pass).

### Stage Step 3: Classify every api_rpc/api_copy reference

Same `classify_conv_id()` as Single-Process Mode, but the branch on the result differs:

- Target pid **is** in `local_pids` → normal in-stage edge, add `(from, to)`.
- Target pid is **not** in `local_pids` → an **external dependency** (points outside this
  stage/project). Do not recursively pull by default — Stage-Wide Mode is bounded to what's
  already local. Emit a stub node (`nodeCount: 0`, `hasAlias: false`, `title`/`label`:
  `"(external — outside this stage)"`) and still draw the edge, so cross-stage coupling is
  visible without expanding scope. If the user explicitly asks to also follow external
  dependencies, fall back to Single-Process Mode's recursive pull-and-cap logic (Step 4
  above) for just those external pids.
- `dynamic` / `alias_unresolved` → same `unresolved[]` list as Single-Process Mode.

### Stage Step 4: Output JSON

Same schema as Step 5 above, written as `dependency_graph_stage_<stage_id>.json`. Set
`rootPid`/`rootTitle` to the process with the highest out-degree (most outbound calls) in
the stage — a representative anchor for the viewer's BFS-depth computation, not a literal
"root": a whole stage has many independent entry points, not one.

**Known cosmetic limitation:** the viewer's "Color: By depth level" mode assumes
single-root connectivity — any process not reachable from the chosen anchor renders as if
at depth 0 (the "Root" color). For Stage-Wide graphs, tell the user to switch **Color** to
**"By alias / type"** instead, which colors by `hasAlias`/`nodeCount` and doesn't depend on
a single root.

### Stage Step 5: Report

```markdown
## Stage Dependency Graph — "<stage title>" (stage_id=570300)

Generated: `dependency_graph_stage_570300.json` (514 processes, N edges)
External dependencies (outside this stage): M references — shown as stub nodes
Unresolved references: X dynamic, Y unresolvable alias

Open `${CLAUDE_PLUGIN_ROOT}/skills/corezoid-dependency-graph/assets/dependency_graph.html`
and load the JSON. For Stage-Wide graphs, prefer switching **Size** to **"By references"**
— fan-in doesn't depend on a single root, so it isn't affected by the multi-entry-point
limitation below, and it surfaces the stage's true hub processes at a glance. Also switch
**Color** to **"By alias / type"** — depth coloring assumes a single root and this graph
has many independent entry points.
```

---

## Reference Documents

| Path | When to read |
|---|---|
| `${CLAUDE_PLUGIN_ROOT}/docs/nodes/call-process-node.md` | `api_rpc`/`api_copy` `conv_id` semantics (static numeric, alias, or dynamic) |
| `${CLAUDE_PLUGIN_ROOT}/skills/corezoid-alias-manager/SKILL.md` | Alias list API request/response shape, environment value resolution |
| `${CLAUDE_PLUGIN_ROOT}/skills/corezoid-review/SKILL.md` | Steps 10–11 — the 1-level dependency inventory this skill extends recursively |
| `${CLAUDE_PLUGIN_ROOT}/skills/corezoid-node-layout/SKILL.md` | Precedent for the MCP server's CLI mode (`mcp-server/run.sh`) used here for the pull loop |
