# Actor Schemas — JSON structure for every actor type

The Discovery skill writes a single combined `actor-graph.json` file. **This
file is the canonical specification of the output format.** Every Discovery
session MUST produce JSON matching the schemas below.

> **Provenance:**
> - Lead actor base schema — `../source-spec/02_discovery_agent_spec.md` §6
> - Lead state machine — `../source-spec/02_discovery_agent_spec.md` §3
> - Migration container — `../source-spec/01_master_spec.md` Part 1.2
> - Phase actor — `../source-spec/01_master_spec.md` Part 1.2.1
>
> **Architectural extension over source spec:** in the original Discovery
> Agent Spec v1.3 §6, fields like `case_1_flow..case_8_flow`, `must_keep[]`,
> `owner_reports[]`, `banks[]`, `previous_attempts[]` live as plain accounts
> on the Lead actor. **This skill canonically decomposes those into
> first-class sub-actors** (CaseProfile / IntegrationMention / Requirement /
> PriorAttempt). The decomposition is deterministic and required — every
> Discovery session produces the same actor types in the same structure.
>
> **Why the extension:** (a) per-case audit trail with own confidence_pct
> and state; (b) cross-Lead pattern detection via catalog actors
> (Integration / PackPattern); (c) cleaner separation of «what client said»
> from «what we extracted»; (d) matches the v1.4 movement toward
> first-class Phase actors in Migration. The decomposition is discussed in
> `../discovery_form_tree.html` artifact (form-topology diagram).
>
> **What this means at runtime:** when Discovery completes, downstream
> agents (SystemProfilerAgent, MappingAgent, etc.) will find a
> `case_profiles[]` array — NOT `case_1_flow / case_2_flow / ...` fields
> on Lead. Do not mix the two representations.

All actors share a common envelope:

```json
{
  "id": "<ActorType>.<client-slug>[.<discriminator>]",
  "type": "<ActorType>",
  "ref": "<kebab-case-unique-ref>",
  "title": "<human-readable title>",
  "form_ref": "<form name in Simulator workspace>",
  "parent_<X>_id": "<reference to parent actor>",
  "state": "<lifecycle state>",
  "created_at": "<ISO 8601>",
  "updated_at": "<ISO 8601>",
  "accounts": { /* key-value: identity, state, accumulator */ },
  "events": [ /* list of events emitted on this actor */ ]
}
```

Identity / state / accumulator are flat key-value entries in `accounts`. For
fields requiring richer info (Dr/Cr semantics, formula, currency) use the
object form: `{ "value": ..., "type": "counter|asset|state|...", "income_type": "debit|credit" }`.

Below — each actor type with required and optional fields.

---

## 1. Lead

**Form ref:** `Lead`
**Cardinality:** exactly 1 per Discovery session
**Created at:** start of session
**Updated:** continuously, persisted after each phase

```json
{
  "id": "Lead.<client-slug>",
  "type": "Lead",
  "ref": "lead-<client-slug>",
  "title": "Lead · <CompanyName>",
  "form_ref": "Lead",
  "state": "INITIAL | GREETING | PROFILE | BUSINESS_FLOW | VOLUMES_INFRA | SPECIFICS | PRESENTATION | BRIEF_GENERATED | AWAITING_HUMAN_HANDOFF | BRIEF_SIGNED | ESCALATED | DROPPED",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "language": "uk | ru | en",
    "source": "web | telegram | referral | partner | operator_session",
    "discovery_started_at": "...",

    "company_name": "...",
    "contact_name": "...",
    "phone": "...",
    "email": "...",

    "industry": "...",
    "outlets_count": 0,
    "employees_count": 0,
    "company_age_years": 0,
    "assortment_type": "ready | custom | mixed",
    "current_system": "...",
    "current_system_version": "...",
    "current_system_pain": "...",
    "migration_trigger": "...",
    "target_deadline": "...",

    "flow_exceptions": [],
    "industry_pack_match_pct": 0,
    "matched_industry_pack": "...",
    "_note_business_flow": "individual case descriptions live in case_profiles[] sub-actors — see CaseProfile schema",

    "orders_per_month": "...",
    "contractors_count": 0,
    "vendors_count": 0,
    "sku_count": 0,
    "concurrent_users": 0,
    "data_dirtiness_flag": "...",
    "db_export_available": false,
    "db_export_format": "mysql | mssql | csv | excel | xml_1c | none",
    "it_ownership": "internal | outsourced | none",
    "_note_integrations": "ПРРО / banks / carriers / accounting integrations live in integration_mentions[] sub-actors — see IntegrationMention schema",

    "top_pain": "...",
    "internal_owner": "...",
    "decision_maker": "...",
    "budget_range": "...",
    "success_criteria": "...",
    "escalation_signals": [],
    "_note_requirements": "must_keep / owner_reports / success_criteria detail live in requirements[] sub-actors — see Requirement schema",
    "_note_prior_attempts": "failed-migration history lives in prior_attempts[] sub-actors — see PriorAttempt schema",

    "deployment_mode": "cloud | hybrid | on_prem | air_gapped",
    "target_cutover_date": "...",
    "predicted_track": "L1 | L2 | L3 | L4",
    "confidence_score": 0.0,

    "prototype_actor_id": "...",
    "migration_actor_id": "...",
    "brief_url": "...",
    "tom_blueprint_url": "...",
    "commercial_offer_url": "...",

    "time_spent_minutes": 0,
    "messages_exchanged": 0
  },
  "events": [
    {"ts": "...", "type": "Lead.Created"},
    {"ts": "...", "type": "Phase.Profile.Completed"},
    {"ts": "...", "type": "Phase.Flow.Completed"},
    {"ts": "...", "type": "Phase.Volumes.Completed"},
    {"ts": "...", "type": "Phase.Specifics.Completed"},
    {"ts": "...", "type": "Phase.Presentation.Completed"},
    {"ts": "...", "type": "EscalationSignal.Triggered", "signal": "...", "urgency": "immediate|next_bd"},
    {"ts": "...", "type": "Prototype.Created"},
    {"ts": "...", "type": "Brief.Generated"},
    {"ts": "...", "type": "Roadmap.Approved"},
    {"ts": "...", "type": "Brief.Signed"}
  ]
}
```

**Validation actions (concrete — do these when condition fires):**
- `outlets_count >= 0` (required). If `outlets_count > 5` → append `"network-logic-needed"` to `lead.accounts.delta_required[]` and ask follow-up about cross-outlet inventory/pricing logic in Phase 3.
- `employees_count >= 1` (required). If `> 100` → set `lead.accounts.predicted_track = "L3"` (override classify_track default) AND append `"large-org"` to `lead.accounts.escalation_signals[]` (urgency: next_business_day). If `> 200` → also emit `EscalationSignal.Triggered` event with urgency `immediate` and stop the dialog per quality-gates.md G7 procedure.
- `target_deadline` must be a date in the future. If less than 60 days from today AND `predicted_track != "L1"` → emit `EscalationSignal.Triggered` with urgency `immediate`, trigger `"impossible-deadline"`.
- `language` must be in `{uk, ru, en}`. If anything else detected (e.g. "pl") → record as `language: "ru"` (default) and add `"non-standard-language"` to `lead.accounts.escalation_signals[]` (next_business_day) — SA can decide whether to engage with the client in a different language.
- `industry` should be normalized to a known catalog entry where possible (`furniture_retail`, `services_design`, `auto_parts`, `wholesale_fmcg`, `horeca`, `light_manufacturing`, `apparel`, `pharmacy`, `construction_materials`, `gov`, `banking`, `healthcare`). If novel → record raw text and proceed; this lowers `industry_pack_match_pct` automatically.

---

## 2. CaseProfile

**Form ref:** `CaseProfile`
**Cardinality:** ~6-12 per Lead (depending on industry — services have 4-6, retail 6-10, multi-channel 8-12)
**Created at:** during Phase 2, as each case is discussed
**Linked to:** Lead (parent)

```json
{
  "id": "CaseProfile.<client-slug>.<case-slug>",
  "type": "CaseProfile",
  "ref": "case-<case-slug>-<client-slug>",
  "title": "Case <N>: <case_name>",
  "form_ref": "CaseProfile",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "pending | covered | n_a | needs_clarification",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "case_number": 1,
    "case_name": "Розничная продажа в салоне",
    "flow_text": "клиент в салоне → менеджер оформляет Order → ...",
    "identified_state_machine": "NEW → PREPAID → ... → CLOSED",
    "confidence_pct": 85,
    "delta_required": ["multi-vendor SubOrder", "InstallationTask"],
    "matched_pack_patterns": []
  },
  "events": [
    {"ts": "...", "type": "CaseProfile.Created"},
    {"ts": "...", "type": "CaseProfile.Submitted"},
    {"ts": "...", "type": "CaseProfile.NeedsClarification"}
  ]
}
```

**Validation actions:**
- `case_number` must be unique per Lead. Re-numbering happens automatically when cases are reordered.
- If `state == "covered"` AND (`flow_text` empty OR `confidence_pct < 50`) → set `state = "needs_clarification"` and emit `CaseProfile.NeedsClarification` event; DiscoveryAgent must revisit this case before exiting Phase 2.
- If client explicitly says "у нас такого нет" / "не работаем с этим" → set `state = "n_a"` with non-empty `flow_text = "n/a — client confirmed"`. This is acceptable for Quality Gate G2 (counts as covered).

---

## 3. IntegrationMention

**Form ref:** `IntegrationMention`
**Cardinality:** ~5-15 per Lead (1 ПРРО, 1-3 banks, 1-2 carriers, optional accounting)
**Created at:** during Phase 3, as each provider is mentioned
**Linked to:** Lead (parent) + (later) catalog Integration actor

```json
{
  "id": "IntegrationMention.<client-slug>.<provider-slug>",
  "type": "IntegrationMention",
  "ref": "integration-<provider-slug>-<client-slug>",
  "title": "<provider_name> · <provider_type>",
  "form_ref": "IntegrationMention",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "mentioned | verified | unsupported | needs_clarification",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "provider_name": "Cash Online",
    "provider_type": "prro | bank | carrier | accounting | messaging | other",
    "integration_pattern": "api | file | manual | webhook",
    "criticality": "critical | important | nice",
    "frequency": "realtime | batch | on_demand",
    "catalog_match_id": null
  },
  "events": [
    {"ts": "...", "type": "IntegrationMention.Identified"},
    {"ts": "...", "type": "IntegrationMention.Verified"}
  ]
}
```

**Validation actions:**
- Normalize `provider_name` using known vocabulary: "Cash Online", "Checkbox",
  "Вчасно" for ПРРО; "Privat24", "Monobank", "УкрСиб", "Райффайзен" for banks;
  "Нова Пошта", "Укрпошта" for carriers. Record normalized value; keep raw
  client phrase in `accounts.client_phrase_raw` if it differed.
- `provider_type` is required. If you cannot determine type from context → ask the client.
- If `criticality == "critical"` AND provider does NOT match a known
  Integration in the current Industry Pack vocabulary → append
  `"unknown-critical-integration:<provider_name>"` to `lead.accounts.delta_required[]`
  AND `"critical-integration-needs-sa-review"` to `lead.accounts.escalation_signals[]`
  (next_business_day urgency).

---

## 4. Requirement

**Form ref:** `Requirement`
**Cardinality:** ~3-10 per Lead
**Created at:** during Phase 4 from `must_keep[]`, `owner_reports[]`,
`success_criteria` keywords
**Linked to:** Lead (parent) + (later) catalog PackPattern actor

```json
{
  "id": "Requirement.<client-slug>.<req-slug>",
  "type": "Requirement",
  "ref": "req-<req-slug>-<client-slug>",
  "title": "<requirement_text> (<source>)",
  "form_ref": "Requirement",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "novel | matches_existing | contradicts_pack",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "requirement_text": "фильтрация заказов по поставщикам",
    "source": "must_keep | owner_report | success_criteria",
    "criticality": "must | should | nice",
    "similarity_score": 0.0,
    "matched_pattern_id": null
  },
  "events": []
}
```

**Validation actions:**
- `requirement_text` must be non-empty. Use the client's words, not a paraphrase.
- If `state == "contradicts_pack"` → append `"requirement-contradicts-pack:<req-slug>"`
  to `lead.accounts.escalation_signals[]` (next_business_day urgency); SA must
  decide whether to accept the contradiction (rare) or push back to the client
  (more common).

---

## 5. PriorAttempt

**Form ref:** `PriorAttempt`
**Cardinality:** 0-3 per Lead (optional; only if client mentions failed
migrations)
**Created at:** during Phase 4
**Linked to:** Lead (parent)

```json
{
  "id": "PriorAttempt.<client-slug>.<sys-slug>",
  "type": "PriorAttempt",
  "ref": "prior-<sys-slug>-<client-slug>",
  "title": "Прежняя попытка: <system_name>",
  "form_ref": "PriorAttempt",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "failed | pivoted | paused | succeeded_partially",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "system_name": "BAS Бухгалтерія",
    "attempted_at": "2025-Q1",
    "reason_failed": "слишком похоже на 1С, менеджеры не приняли",
    "lessons_learned": "нужна смена UX paradigm, не просто перенос",
    "escalation_weight": 7
  },
  "events": []
}
```

**Validation actions:**
- `escalation_weight` is in range `[1, 10]`. Score based on: how recent (<2y =
  +3), how relevant (same system family as current = +3), how the failure
  ended (cancelled mid-flight = +2 vs completed-but-rolled-back = +1).
- If `escalation_weight >= 7` AND `state == "failed"` → append
  `"previous-failed-migration"` to `lead.accounts.escalation_signals[]`
  (next_business_day urgency). The client has a higher-than-average risk
  profile; SA should review before commitment.

---

## 6. Prototype

**Form ref:** `Prototype`
**Cardinality:** exactly 1 per Lead (created in Phase 5)
**Created at:** Phase 5 Step 5
**Linked to:** Lead (parent), Industry Pack (template)
**Lifecycle:** ephemeral — TTL 14 days, then auto-cleanup

```json
{
  "id": "Prototype.<client-slug>",
  "type": "Prototype",
  "ref": "prototype-<client-slug>",
  "title": "Demo стенд · <CompanyName>",
  "form_ref": "Prototype",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "active | expired | promoted_to_production",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "industry_pack_actor_id": "<matched_pack_name>",
    "stand_url": "demo.simulator.company/p/<slug>-<random>",
    "expires_at": "<created_at + 14 days>",
    "sample_data_hint": "5 sample customers, 3 sample vendors, 15 SKU, 4 showrooms",
    "sample_customers": [],
    "sample_vendors": [],
    "sample_products": [],
    "showrooms": []
  },
  "events": [
    {"ts": "...", "type": "Prototype.Created"}
  ]
}
```

**Note:** for L1 Greenfield clients, Prototype state can transition to
`promoted_to_production` instead of expiring — record this as a note in
`accounts.notes` if applicable.

---

## 7. Migration (container)

**Form ref:** `Migration`
**Cardinality:** exactly 1 per Lead (created in Phase 5)
**Created at:** Phase 5 Step 6
**Linked to:** Lead (parent) + Phase actors (children, by `parent_migration_id`)

```json
{
  "id": "Migration.<client-slug>",
  "type": "Migration",
  "ref": "migration-<client-slug>",
  "title": "Migration · <CompanyName>",
  "form_ref": "Migration",
  "parent_lead_id": "Lead.<client-slug>",
  "state": "INITIATED | ROADMAP_GENERATED | ACTIVE | PAUSED | CUTOVER_READY | CUTOVER_EXECUTED | OPERATING | EVOLVING | ROLLBACK",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "client_name": "...",
    "track": "L1 | L2 | L3 | L4",
    "deployment_mode": "cloud | hybrid | on_prem | air_gapped",
    "start_date": "...",
    "target_cutover_date": "...",
    "industry": "...",
    "assigned_sa_id": null,
    "roadmap_graph_id": "RoadmapGraph.<client-slug>.v1",

    "current_phase_id": "Phase.<client-slug>.Discovery",
    "overall_progress_pct": 0,
    "phases_completed_count": 1,
    "phases_blocked_count": 0,
    "phases_paused_count": 0,
    "forecasted_cutover_date": "...",
    "deviation_from_plan_days": 0,
    "freshness_score": 1.0,
    "time_in_state_paused": 0,
    "pause_count": 0,

    "phase_actor_ids": [
      "Phase.<client-slug>.Discovery",
      "Phase.<client-slug>.Profiling",
      "..."
    ]
  },
  "events": [
    {"ts": "...", "type": "Migration.Started"},
    {"ts": "...", "type": "Roadmap.Generated"}
  ]
}
```

---

## 8. Phase

**Form ref:** `Phase`
**Cardinality:** 9-15 per Migration (varies by track and custom phases)
**Created at:** Phase 5 Step 6, in bulk
**Linked to:** Migration (parent) + other Phase actors (via `depends_on`)

```json
{
  "id": "Phase.<client-slug>.<phase-name-slug>",
  "type": "Phase",
  "ref": "phase-<phase-name-slug>-<client-slug>",
  "title": "Phase: <phase_name>",
  "form_ref": "Phase",
  "parent_migration_id": "Migration.<client-slug>",
  "state": "pending | ready | active | blocked | paused | completed | skipped | reworked",
  "created_at": "...",
  "updated_at": "...",
  "accounts": {
    "phase_name": "Profiling",
    "phase_type": "sense | model | migrate | validate | evolve | custom",
    "owner_agent": "SystemProfilerAgent",
    "depends_on": ["Phase.<client-slug>.Discovery"],
    "is_parallel_with": [],
    "is_critical_path": true,
    "target_start_date": "...",
    "target_end_date": "...",
    "signoffs_required": [
      {"role": "SA", "person_id": null},
      {"role": "product_owner", "person_id": null}
    ],
    "current_assignee_id": null,
    "blocked_reason": null,
    "progress_pct": 0,
    "time_spent_hours": 0,
    "time_budget_hours": 16,
    "deviation_from_plan_days": 0,
    "artifacts_produced_count": 0,
    "signoffs_collected_count": 0,
    "rework_count": 0
  },
  "events": []
}
```

### Standard phases per track

**L1 Greenfield:** Discovery, Pack Selection, Data Init, Setup, Training,
Go-Live, Operate, Evolve (~8)

**L2 Single replacement:** Discovery, Profiling, Mapping, Data Migration,
Process Migration, Validation, Cutover, Operate, Evolve (~9)

**L3 Multi-system:** Discovery, Profiling × N, Synthesis, Mapping,
Data Migration, Process Migration, Validation (cross-system), Staged Cutover,
Operate, Evolve (~10-13)

**L4 Enterprise:** L3 set + Pre-engagement Security Review, On-prem Setup,
Compliance Audit, Extended Validation, Board Approval Phase (~14-16)

### Custom phases — add when client signals them

| Client signal | Custom Phase | Phase type | Where to insert |
|---|---|---|---|
| "одобрение совета директоров" | BoardApproval | custom | before Cutover |
| "регуляторная проверка в <quarter>" | RegulatoryReviewWindow | custom | as blocked phase in that period |
| "летние каникулы", "зимняя пауза" | PlannedPause | custom | activates PAUSED state |
| "фазами по модулям", "wave-based cutover" | StagedCutover.<module> × N | custom | replaces single Cutover |
| "security audit обязателен" | SecurityReview | custom | before Profiling (L4 default) |
| "международные дочки" | MultiEntitySetup | custom | after Mapping |
| "дедлайн НДС 1 января" | NDSReadiness | custom | hard deadline anchor |

---

## 9. Integration (catalog actor, NOT created during Discovery)

Mentioned here for completeness. **DiscoveryAgent does NOT create
Integration actors** — they live at workspace catalog level and are referenced
from IntegrationMention via `catalog_match_id`. Creation/matching is the job
of a downstream agent or a manual SA review.

---

## 10. PackPattern (catalog actor, NOT created during Discovery)

Same as Integration — catalog-level, not created by DiscoveryAgent. Referenced
from CaseProfile / Requirement via `matched_pack_patterns[]` /
`matched_pattern_id`. Filled later by SA review or pack-evolution agent.

---

## Combined `actor-graph.json` structure — CANONICAL

This is the single source of truth for the output file format. SKILL.md
references this section.

```json
{
  "meta": {
    "client_slug": "golden-house",
    "client_name": "Golden House",
    "agent_version": "software-migration-onramp@1.0.0",
    "output_path_absolute": "/Users/.../project/discovery-output/golden-house/",
    "session_started_at": "2026-05-19T14:23:00Z",
    "session_last_updated": "2026-05-19T15:55:00Z",
    "session_paused_at": null,
    "phase_completed": ["profile", "flow", "volumes", "specifics", "presentation"],
    "phase_current": "presentation",
    "language": "uk",
    "source": "operator_session",
    "industry_pack_matched": "furniture_retail",
    "industry_pack_coverage_pct": 92,
    "predicted_track": "L2",
    "deployment_mode": "cloud",
    "target_cutover_date": "2026-07-07",
    "escalation_triggered": false,
    "quality_gates": {
      "g1_identity":              "passed | partial | failed | not_yet",
      "g2_cases_coverage":        "passed | partial | failed | not_yet",
      "g3_requirements_coverage": "passed | partial | failed | not_yet",
      "g4_volumes":               "passed | partial | failed | not_yet",
      "g5_decision":              "passed | partial | failed | not_yet",
      "g6_pack_match":            "passed | partial | failed | not_yet",
      "g7_no_escalation":         "passed | partial | failed | not_yet",
      "g8_roadmap_approved":      "passed | partial | failed | not_yet"
    },
    "gate_notes": []
  },
  "actors": {
    "lead": { /* one Lead actor — see § Lead above */ },
    "case_profiles": [ /* CaseProfile actors */ ],
    "integration_mentions": [ /* IntegrationMention actors */ ],
    "requirements": [ /* Requirement actors */ ],
    "prior_attempts": [ /* PriorAttempt actors (0-3) */ ],
    "prototype": { /* one Prototype actor — created in Phase 5 */ },
    "migration": { /* one Migration container — created in Phase 5 */ },
    "phases": [ /* Phase actors — created in Phase 5 */ ]
  },
  "events": [
    /* chronological meta-log: Lead.Created, Phase.X.Completed,
       Prototype.Created, Brief.Generated, Roadmap.Approved,
       EscalationSignal.Triggered, Brief.Signed */
  ]
}
```

**Order of creation throughout the dialog:**

| When | What appears |
|---|---|
| Session start (loadClient equivalent) | `lead` with starter accounts (language, source, discovery_started_at) |
| End of Phase 1 (form confirm) | `lead` accounts updated with identity, profile fields |
| During Phase 2 (per case) | `case_profiles[]` grows; `lead.industry_pack_match_pct`, `lead.matched_industry_pack` set |
| End of Phase 3 (form submit) | `lead` volumes accounts updated; `integration_mentions[]` populated |
| End of Phase 4 | `lead` specifics accounts updated; `requirements[]` and possibly `prior_attempts[]` populated |
| Phase 5 Step 5 | `prototype` appears |
| Phase 5 Step 6 | `migration` and `phases[]` appear |
| Phase 5 Step 7 | `lead.state = AWAITING_HUMAN_HANDOFF`, signoff event recorded |

---

**Version:** see `../SKILL.md` frontmatter (single source).

**Source spec mapping:**
- Lead actor schema — `source-spec/02_discovery_agent_spec.md` §6 «Схема актора Lead»
- DiscoveryAgent actor schema — `source-spec/02_discovery_agent_spec.md` §7
- Lead state machine — `source-spec/02_discovery_agent_spec.md` §3 «Граф состояний актора Lead»
- Migration actor (overall) — `source-spec/01_master_spec.md` Part 1.2 «Migration as Actor Graph»
- Phase actor (sub-of-Migration) — `source-spec/01_master_spec.md` Part 1.2.1 «Phase actor — детальная структура»
- Pause/resume accounts on Migration (time_in_state_paused, pause_count) — `source-spec/01_master_spec.md` Part 3.7 «pause/resume» + Part 1.2 accumulator section
- Roadmap forecasting (forecasted_cutover_date) — `source-spec/01_master_spec.md` Part 3.8.4
- 4 tracks with their standard phase sets — `source-spec/01_master_spec.md` Part 3.8.1 «Roadmap templates per track»
- Custom phases (Board approval / Regulatory etc.) — `source-spec/03_discovery_agent_kb_bundle.md` §1.2a «Сигналы клиента, которые означают custom phases»
- Furniture Retail Pack as concrete example — `source-spec/02_discovery_agent_spec.md` §5
- CaseProfile / IntegrationMention / Requirement / PriorAttempt — **proposed decomposition** (not in original spec — added during prototype architecture work). In source spec these live as fields directly on Lead (§6 case_1..case_8_flow / must_keep / banks[] / etc.). Sub-actor decomposition is an optional architectural enhancement, justified by the rationale in `discovery_form_tree.html` artifact
