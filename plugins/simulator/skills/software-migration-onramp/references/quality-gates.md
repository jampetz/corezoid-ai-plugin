# Quality Gates — self-check before progressing or finalizing

DiscoveryAgent runs **8 Quality Gates** at specific milestones. Each gate must
pass before the next phase / Brief generation. Failed gates dictate either
**continue asking** or **escalate to SA**. **This is the operational canon for
gate logic.**

> **Provenance:** synthesized from
> `../source-spec/03_discovery_agent_kb_bundle.md` §3 (Completion Checklist),
> §4 (Quality Gates G1-G8), plus
> `../source-spec/02_discovery_agent_spec.md` §9 (escalation triggers).
> Decision tables and «soft pass vs hard pass» distinction added during
> prototype work for operational clarity. See "Source spec mapping" at the
> bottom for traceability.

---

## G1 — Identity

**When checked:** before transitioning Lead.state from `PROFILE` to `BUSINESS_FLOW`.

**Pass condition:** all required Lead identity accounts filled:

- `company_name` non-empty
- `contact_name` non-empty
- `phone` OR `email` non-empty (at least one)
- `industry` non-empty
- `outlets_count` >= 0
- `employees_count` >= 1
- `current_system` non-empty
- `migration_trigger` non-empty
- `target_deadline` is a valid future date

**On fail:** ask follow-up questions in Phase 1 for missing fields. Do not
emit `Phase.Profile.Completed` until all required identity fields filled.

---

## G2 — Cases coverage

**When checked:** before transitioning `BUSINESS_FLOW` → `VOLUMES_INFRA`.

**Pass condition:** for each relevant case type for the client's industry,
a CaseProfile exists with:

- `state ∈ {covered, n_a}`
- If `covered`: `flow_text` non-empty (1-3 sentences) AND `confidence_pct >= 50`
- If `n_a`: explicit client confirmation that this case doesn't apply

**Quick metric:** at least 6 covered+n_a cases for retail, at least 4 for
services, no `pending` or `needs_clarification` states remaining.

**On fail:**
- If some cases are `needs_clarification` → ask follow-up questions
- If `confidence_pct < 50` on a covered case → re-discuss that case
- If client refuses to discuss critical cases (e.g. refuses to talk about
  payments) → fail Gate and escalate to SA with `escalation_signals[] += "client_uncooperative_on_critical_cases"`

---

## G3 — Requirements coverage

**When checked:** before transitioning `SPECIFICS` → `PRESENTATION`.

**Pass condition:** the Lead has captured all 10 standard requirement areas
(from the Golden House `Покрытие требований клиента` checklist, adapted):

| # | Requirement area (EN / RU) | Where captured |
|---|---|---|
| 1 | Store full deal chain end-to-end / Хранить заказы со всей цепочкой | case_1, case_3, case_6, case_8 |
| 2 | Sub-orders with different lead times / Подзаказы с разными сроками | case_3, case_4 |
| 3 | Lock-down after a point of no return / Запрет корректировки после момента X | case_7 |
| 4 | Tie inbound payment / shipment to order / Связать приход/оплату/отгрузку с заказом | cases 2, 5, 6 |
| 5 | Retail vs wholesale pricing (or equivalent) / Прайс розница/опт | case_3 |
| 6 | Defined roles (sales / warehouse / procurement / logistics) / Роли | employees_count, outlets_count, internal_owner |
| 7 | Order status by stage / Статусы заказа по этапам | cases 1-7 (state machines) |
| 8 | Filter orders by vendor / Фильтр по поставщику | Requirement actors via must_keep |
| 9 | Filter orders by stage / Фильтр по стадии | Requirement actors via must_keep OR owner_reports |
| 10 | Roll-up to financial result / Стягивание в финрезультат | case_7, Requirement actors via owner_reports |

**Threshold:** **9+ из 10 закрыты** to pass. If 7-8 — flag follow-up. If <7 —
fail gate.

**On fail:** open-ended question:

> "Я хочу убедиться, что ничего не упустил. Есть ли у вас в текущей системе
> что-то, что мы не обсудили — какие-то отчёты, фильтры, специфика
> работы с клиентами/поставщиками?"

---

## G4 — Volumes filled

**When checked:** end of Phase 3.

**Pass condition:** quantitative Lead accounts have at least order-of-magnitude
values:

- `orders_per_month` — value or range (e.g. "60-80")
- `contractors_count` — integer
- `vendors_count` — integer (0 acceptable for services)
- `sku_count` — integer (0 acceptable for services)

**On fail:** client refuses to estimate → record `null` with note
`"client_declined_to_estimate"` and proceed if other gates allow. This is a
soft fail — only blocks if combined with other failures.

---

## G5 — Decision-making known

**When checked:** end of Phase 4.

**Pass condition:**

- `internal_owner` non-empty (someone is responsible inside the client company)
- `decision_maker` non-empty (someone signs contracts)
- `budget_range` set (specific number, range, or "не озвучивает")
- `success_criteria` non-empty

**On fail:**
- `internal_owner` empty → **escalate to SA** (no owner = no commitment)
- `decision_maker` empty → **escalate to SA**
- `budget_range` is "не озвучивает" → continue but flag
- `success_criteria` empty → ask "А по каким KPI через год вы будете
  считать миграцию успешной?"

---

## G6 — Industry Pack matched

**When checked:** end of Phase 2 (computed) + verified before Phase 5.

**Pass condition:** `industry_pack_match_pct >= 60`.

**On fail:** `industry_pack_match_pct < 60` means the client's cases don't fit
any existing pack well. Action:

- Set `predicted_track = "L4 Custom"` regardless of size signals
- Add to `escalation_signals[] += "low_pack_coverage"`
- Continue Phase 3-5 but warn the operator: "Этому клиенту нужен
  отдельный Industry Pack или Custom track — SA должен спроектировать
  индивидуально"

---

## G7 — No active escalation triggers

**When checked:** continuously throughout the dialog AND before Phase 5
finalization.

**Pass condition:** `escalation_signals[]` is empty OR contains only minor
signals (none with urgency `immediate`).

**Eight escalation triggers (from spec):**

| # | Trigger | Detection | Urgency | Action on fire |
|---|---|---|---|---|
| 1 | Explicit human request | Client says "хочу поговорить с человеком", "позовите менеджера" | immediate | `escalate_to_human(reason="explicit_request")`; politely transfer |
| 2 | Regulated industry | `industry ∈ {banking, insurance, pharma, healthcare, government}` | immediate | `escalate_to_human(reason="regulated_industry")`; stop discovery |
| 3 | Enterprise size | `employees_count > 200` | immediate | `escalate_to_human(reason="enterprise_size")` |
| 4 | Emotional negative | Detect anger, frustration, swearing, sarcasm in client's tone | immediate | `escalate_to_human(reason="emotional_negative")`; calm + transfer |
| 5 | Internal conflict | Client mentions disagreement between decision-makers ("директор хочет X, владелец Y") | next_business_day | flag + continue, SA reviews tomorrow |
| 6 | Previous failed migration | PriorAttempt actor with `outcome=failed` AND `escalation_weight >= 7` | next_business_day | continue, SA reviews tomorrow |
| 7 | Complex specifics / high pack delta | `industry_pack_match_pct < 60` OR `delta_required[].length > 4` | next_business_day | continue, SA designs Custom track |
| 8 | Legal-sensitive | Client asks about specific contract clauses, GDPR, data sovereignty | next_business_day | "Это к юристам, передам SA" |

**On fire — what to do (procedure for Claude):**

1. **Add an entry to** `actors.lead.accounts.escalation_signals[]`:
   ```json
   { "trigger": "<trigger_name>", "urgency": "immediate|next_business_day", "context": "<one-line description>", "ts": "<ISO-8601 now>" }
   ```
2. **Append to** `actors.lead.events[]`:
   ```json
   { "type": "EscalationSignal.Triggered", "ts": "<ISO-8601 now>", "signal": "<trigger_name>" }
   ```
3. **Write** `actor-graph.json` immediately with the updated state.
4. **If `urgency == "immediate"`:**
   - Also create a sibling file `escalation.json` in the same output directory
     summarizing the trigger + context + client name for SA hand-off.
   - Set `actors.lead.state = "ESCALATED"`.
   - Tell the client calmly (adapt phrase to session language):
     > «Понимаю. Сейчас передам ваш кейс solution architect'у — он
     > перезвонит вам в течение часа.»
   - Stop the dialog. Do not attempt to fill remaining Quality Gates.
   - End your turn with the structured summary message (per SKILL.md spec)
     showing what was collected before the escalation.
5. **If `urgency == "next_business_day"`:**
   - Continue the dialog — the signal will be picked up by SA in their daily
     review of the actor-graph.json.
   - Note in `_meta.gate_notes[]`: `"G7 partial — next_business_day signal queued for SA review"`.

**Pass condition for Brief generation:** no `immediate` signals in
`escalation_signals[]`. `next_business_day` signals are allowed but warn
operator.

---

## G8 — Roadmap approved (Phase 5 final)

**When checked:** at the very end of Phase 5, before writing final
`actor-graph.json`.

**Pass condition:**

- `Prototype` actor created
- `Migration` actor created with valid `track`, `deployment_mode`,
  `target_cutover_date`
- `phase_actors[].length >= 8` (minimum for L1) up to 15+ (L4)
- Phase actor dependencies form a valid DAG (no cycles)
- `forecasted_cutover_date` within ±10% deviation from `target_cutover_date`
  (or explicit warning if more)
- Client confirmed roadmap (event `Roadmap.Approved` with signer
  `product_owner`)

**On fail:**
- If roadmap dates infeasible (e.g. client demands cutover in 30 days for L2)
  → renegotiate; if client insists → escalate to SA, do not approve
- If client refuses to sign Roadmap.Approved → write current state, mark
  Lead state as `BRIEF_GENERATED` but NOT `BRIEF_SIGNED`. Note in summary
  that approval is pending.

---

## Quality Gate decision table — what to do at each milestone

| Milestone | Required gates | Action if all pass | Action if any fail |
|---|---|---|---|
| End of Phase 1 | G1 | Move to Phase 2 | Ask follow-ups in Phase 1 |
| End of Phase 2 | G2, G6 (initial) | Move to Phase 3 | Re-discuss cases or flag |
| End of Phase 3 | G4 | Move to Phase 4 | Soft warn; continue if other gates fine |
| End of Phase 4 | G3, G5, G7 | Move to Phase 5 | Add follow-up Qs or escalate |
| Phase 5 final | G6 verified, G8 | Write final artifact, exit | Don't finalize; flag for SA |

---

## Soft pass vs hard pass

- **Hard pass** = all required conditions met. Move forward.
- **Soft pass** = some conditions partial (e.g. budget_range = "не озвучивает").
  Move forward but flag in `_meta.gate_notes[]`.
- **Hard fail** = critical condition missing (e.g. no decision_maker).
  Either ask follow-ups or escalate. Don't move forward.

---

**Version:** see `../SKILL.md` frontmatter (single source).

**Source spec mapping:**
- 8 Quality Gates G1-G8 (canonical) — `source-spec/03_discovery_agent_kb_bundle.md` §4 «Quality Gates перед handoff»
- G3 «9+/10 requirements» methodology — `source-spec/03_discovery_agent_kb_bundle.md` §3.1 «Покрытие требований клиента»
- Контрагенты checklist (G2) — `source-spec/03_discovery_agent_kb_bundle.md` §3.2 «Контрагенты и отделы»
- Dashboards by role (referenced in G3) — `source-spec/03_discovery_agent_kb_bundle.md` §3.3 «Дашборды по ролям»
- 8 escalation triggers (G7) — `source-spec/02_discovery_agent_spec.md` §9 «Триггеры эскалации»
- G8 Roadmap approval (new in v1.1) — `source-spec/03_discovery_agent_kb_bundle.md` §4 «Gate 8 — Roadmap graph approved»
- Track classification thresholds (G6) — `source-spec/02_discovery_agent_spec.md` §8 «Логика классификации трека»
