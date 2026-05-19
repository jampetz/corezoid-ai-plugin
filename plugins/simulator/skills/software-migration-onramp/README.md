# software-migration-onramp

Discovery facilitator skill for **Smart Company Onramp** — guides a new client
through a structured 60-90 minute migration onboarding interview and emits a
complete actor-graph JSON snapshot to disk.

This is the **first of 10 AI-agents** defined in the Master Spec v1.4 — it's
the entry point to the migration funnel. Subsequent agents (SystemProfilerAgent,
MappingAgent, DataMigrator, ProcessMigrator, ValidationAgent, CutoverAgent,
EvolutionAgent, ResumptionAgent) can be added as separate skills later.

---

## What it does

1. Detects user intent to start a new Discovery interview
2. Plays the role of **DiscoveryAgent** (consultative, not anketa)
3. Conducts a 5-phase dialog: Profile → Business Flow → Volumes → Specifics → Presentation
4. Identifies which Industry Pack fits the client's verticals (Furniture
   Retail, Services, Auto Parts, etc.)
5. Generates a personalized Migration Roadmap as an actor graph
6. **Writes results as local JSON files** under `./discovery-output/<client-slug>/`

**It does NOT push anything to Simulator/Corezoid.** The output JSON is
structured to be ready for `POST` via simulator MCP tools as a separate step.

---

## Files in this package

```
software-migration-onramp/
├── README.md                 ← this file
├── SKILL.md                  ← main skill manifest (read by Claude on activation)
├── references/               ← operational helpers, compact & ready-for-action
│   ├── agent-persona.md      ← DiscoveryAgent character, language, principles
│   ├── dialog-prompts.md     ← 5-phase dialog procedure with concrete prompts
│   ├── actor-schemas.md      ← JSON structure for each of 10 actor types
│   └── quality-gates.md      ← 8 Quality Gates G1-G8 with pass/fail logic
└── source-spec/              ← bundled original specs from leadership (traceability)
    ├── 00_README.md          ← package overview & terminology
    ├── 01_master_spec.md     ← Master Spec v1.4 — full architecture
    ├── 02_discovery_agent_spec.md   ← Discovery Agent Spec v1.3 — CANONICAL
    ├── 03_discovery_agent_kb_bundle.md  ← KB Bundle v1.1 — Gates & examples
    ├── 04_system_profiler_agent_spec.md ← downstream agent context
    ├── 05_golden_house_simulation.md    ← worked example end-to-end
    └── 06_signoff_chart.md   ← approval roadmap
```

**Rule of precedence:** `references/` is the **operational canon** that
Claude reads at runtime. `source-spec/` are the original leadership
documents from which references/ were synthesized — bundled for traceability
and audit. If a discrepancy is found, treat it as a bug in references/ (sync
references/ to match the source). Do not switch to source-spec mid-session.

---

## For maintainers (Claude reading this skill at activation — skip this section)

This installation guide is for the human maintainer moving the skill package
into the plugin project. Claude does NOT need to execute these steps during
a Discovery session.

### Installation into Simulator plugin

When you're ready to use this in your project (`corezoid-ai-plugin`):

1. Copy the entire `software-migration-onramp/` folder into
   `plugins/simulator/skills/`:

```bash
cp -r /Users/user/Documents/Middleware/Migration\ of\ any\ software\ to\ the\ simulator/software-migration-onramp \
      /Users/user/Documents/Middleware/AI\ SIMULATOR\ AGENT/corezoid-ai-plugin/plugins/simulator/skills/
```

2. Reload Claude Code (or quit + relaunch) so it picks up the new skill.

3. Test activation by typing one of the trigger phrases in a Claude Code
   session in any project directory:

```
> начни discovery с новым клиентом
> начать онбординг
> новый клиент, проведи discovery
> software migration onramp для клиента
```

Claude should activate the `software-migration-onramp` skill and begin the
pre-flight step (asking whether it's a real client or simulation, language).

---

## Activation triggers

The canonical list lives in the `description:` field of `SKILL.md` frontmatter.
It's intentionally narrow — only phrases that specifically signal **starting
a Discovery session** trigger the skill. Examples:

- "начать discovery", "проведи discovery", "сделай discovery"
- "запусти онбординг клиента", "discovery с клиентом"
- "discovery agent", "migration onramp discovery"
- "software migration onramp"
- "новый клиент — запускай" (the explicit "запускай / start / kick off"
  is what disambiguates from architecture discussion)

The skill does NOT activate for general discussion of the Onramp
architecture, Smart Company concept, or theoretical questions — those are
design conversations, not Discovery sessions. If you find yourself wanting
that, manually invoke the skill or rephrase your message to include one of
the trigger phrases above.

---

## Output

After running the skill:

```
./discovery-output/<client-slug>/
└── actor-graph.json    ← single combined JSON with all created actors
```

The `actor-graph.json` schema is documented in `references/actor-schemas.md`.
Top-level structure:

```json
{
  "meta": { /* session metadata, gates, track, pack match */ },
  "actors": {
    "lead": { ... },
    "case_profiles": [ ... ],
    "integration_mentions": [ ... ],
    "requirements": [ ... ],
    "prior_attempts": [ ... ],
    "prototype": { ... },
    "migration": { ... },
    "phases": [ ... ]
  },
  "events": [ /* chronological log */ ]
}
```

**Typical session results in ~15-35 actors** (1 Lead + 6-12 CaseProfile +
5-15 IntegrationMention + 3-10 Requirement + 0-3 PriorAttempt + 1 Prototype +
1 Migration + 9-15 Phase).

---

## Usage modes

### Real client mode

User is the operator. Claude waits for the real client to write first
(or operator provides client's input).

```
> Олексій, наш новый клиент Golden House хочет начать. Запускай discovery.
```

→ Claude starts dialog. Each client response → Claude updates actor-graph.json.

### Simulation mode

For demos / testing — Claude plays both interviewer and client using sensible
defaults.

```
> Прогони симуляцию discovery для типичного мебельного ритейлера 4 салона 28 сотрудников.
```

→ Claude generates the conversation symbolically and writes the actor graph.

---

## Next steps (after Discovery)

Once `actor-graph.json` exists, you can:

1. **Review with SA** — open the file, walk through actors with solution architect
2. **Generate Discovery Brief** (PDF/Markdown) — separate skill, not in this package
3. **Push to Simulator** — for each actor, POST via simulator MCP:
   - `post-actors-actor-formId` → create actor
   - `post-accounts-actorId` → create accounts
   - `post-transactions-accountId` → write initial values
   - `post-actors-link-accId` → connect Lead ↔ sub-actors
4. **Trigger downstream agents** — SystemProfilerAgent for actual source-system profiling, MappingAgent, etc.

---

## Versioning

- **v1.0.0** (this version) — initial release, all 5 phases, 10 actor types,
  8 Quality Gates, simulation + real-client modes. Output: local JSON only.

---

## Source documents (bundled in `source-spec/`)

This skill is the implementation of leadership-provided specifications:

- **`source-spec/01_master_spec.md`** — Master Spec v1.4
  Smart Company Onramp Universal Migration Project. Migration as Actor Graph
  with nested roadmap_graph of Phase actors, 10 AI-agents, 4 tracks
  (L1-L4) × 4 deployment modes, Onramp Process pack as meta-Industry-Pack.

- **`source-spec/02_discovery_agent_spec.md`** — Discovery Agent Spec v1.3
  Detailed DiscoveryAgent specification. §1 system prompt, §2 MCP tool
  signatures (8 tools incl. `generate_roadmap_graph`), §3 Lead state machine,
  §4 5-phase detailed dialog prompts, §5 Furniture Retail Industry Pack
  pattern by 8 Golden House cases, §6 Lead actor schema (~64 accounts),
  §7 DiscoveryAgent actor schema, §8 classify_track logic, §9 escalation
  triggers (8 types), §10 Discovery Brief template, §11 MVP roadmap.

- **`source-spec/03_discovery_agent_kb_bundle.md`** — Discovery Agent KB
  Bundle v1.1. Knowledge base for prompt engineering. §1 RAG content
  (Golden House full as reference + Industry Packs + UA Compliance +
  competitors + pricing), §1.2a Onramp Process pack signals, §2 few-shot
  Golden House example, §3 Completion Checklist (10 requirements
  coverage + 8 dashboards + MVP demo), **§4 Quality Gates G1-G8**
  (canonical), §5 quality metrics, §6 supervisor console.

- **`source-spec/04_system_profiler_agent_spec.md`** — SystemProfilerAgent
  Spec v1.1. Downstream agent. Useful for understanding what Discovery's
  output (System Profile per source system) needs to enable.

- **`source-spec/05_golden_house_simulation.md`** — Golden House Simulation
  v1.1. Full narrative path of one example client through the 5-phase
  Discovery + downstream agents. Worked example for «what good looks like».

- **`source-spec/06_signoff_chart.md`** — Sign-off Chart v1.1. Approval
  roadmap for the entire Onramp launch. Not directly used during Discovery.

These files are **bundled inside the skill package** so Claude has them in
context. They're the same files as in the parent
`Migration of any software to the simulator/` folder (master copies for
ongoing edits).

---

## License

Same license as the parent `corezoid-ai-plugin` repository.

---

## Future skills in this family

When the spec for downstream agents is complete, add:

```
plugins/simulator/skills/
├── software-migration-onramp/        (this — Phase 1: Discovery)
├── software-migration-profiler/      (Phase 2: SystemProfilerAgent, 5 modes)
├── software-migration-mapper/        (Phase 3: GraphSynthesizer + MappingAgent)
├── software-migration-data/          (Phase 4: DataMigrator)
├── software-migration-process/       (Phase 5: ProcessMigrator)
├── software-migration-validator/     (Phase 6: ValidationAgent)
├── software-migration-cutover/       (Phase 7: CutoverAgent)
├── software-migration-evolve/        (Phase 8: EvolutionAgent)
└── software-migration-resumption/    (Cross-phase: ResumptionAgent for pause/resume)
```

Each picks up where the previous left off, reading the actor-graph.json
written by the previous phase and emitting an updated version.
