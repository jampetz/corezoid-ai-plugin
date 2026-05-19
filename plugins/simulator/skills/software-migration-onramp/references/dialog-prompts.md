# Dialog Prompts — 5-phase Discovery Procedure

Concrete dialog instructions per phase. **This is the operational canon for
the 5-phase Discovery procedure.** Read the relevant section at the start of
each phase. Free-form within the guidelines; don't read questions verbatim.

> **Provenance:** synthesized from
> `../source-spec/02_discovery_agent_spec.md` §4 (canonical phase prompts),
> §5 (Industry Pack pattern), §8 (classify_track logic), §10 (Brief template),
> plus operational additions made during prototype work:
> hybrid form+dialog approach (Phase 1/3 use confirmation cards, Phases 2/4
> stay narrative), explicit sub-actor creation per phase, custom-phase signal
> handling in Phase 5. For an end-to-end worked example see
> `../source-spec/05_golden_house_simulation.md`. See "Source spec mapping"
> at the bottom for line-by-line provenance.

---

## Phase 1 — Profile (10-15 min)

**Goal:** identity / current system / migration trigger / target deadline.
**Lead.state on entry:** `INITIAL` → on exit: `PROFILE` (after embedded form
confirmation), then `BUSINESS_FLOW`.
**Exit event:** `Phase.Profile.Completed`.

### Opening (greeting + setting expectations)

Use this template (adapt to language):

> "Добрый день! Я — Discovery-консультант компании Simulator. За ближайший
> час (можем разбить на 2-3 захода, если удобнее) я разберусь в вашем бизнесе
> настолько, чтобы понять, подойдёт ли вам наша платформа для замены текущей
> учётной системы. На выходе у вас будет персональный демо-стенд с вашими
> данными и предварительное коммерческое предложение. Начнём с базового —
> расскажите, чем занимается ваша компания?"

### Key questions and account mappings

Ask conversationally, not in this order. Accounts to fill on Lead:

| Question theme | Lead account | Follow-up trigger |
|---|---|---|
| Чем занимаетесь, давно на рынке? | `industry`, `company_age_years` | если производство → «своё или импорт?» |
| Сколько точек/салонов/магазинов? | `outlets_count` | если >5 — уточнить сетевую логику |
| Готовая продукция / на заказ / смешанный? | `assortment_type` | «на заказ» → копать индивидуальные сделки в Phase 2 |
| Сколько сотрудников всего? | `employees_count` | если >100 — флаг Custom; >200 — autoescalate immediate |
| Какая система сейчас? | `current_system`, `current_system_version` | «1С» → украинская или российская? БАС или классика? |
| Что в ней нравится / не нравится? | `current_system_pain` | слушать внимательно |
| Почему меняете сейчас? | `migration_trigger` | НДС/ФОП-2027 → связать с дедлайном |
| Дедлайн перехода? | `target_deadline` | <60 дней — autoescalate (нереально для L2+) |

### Closing — embedded confirmation form

After 4-5 turns of dialogue, propose the confirmation form:

> "Понял. Зафиксируем коротко — подтвердите или поправьте эти значения,
> они пойдут в карточку лида:"

Then emit a structured block the operator UI renders as a form
(or, in markdown mode, just enumerate the prefilled values):

```
Профиль <CompanyName>:
- Точек/салонов:           <outlets_count>
- Сотрудников:             <employees_count>
- Лет на рынке:            <company_age_years>
- Ассортимент:             <assortment_type>
- Текущая система:         <current_system>
- Дедлайн перехода:        <target_deadline>
- Триггер миграции:        <migration_trigger>
```

Wait for client confirmation. If they edit anything — update accounts. Then:

> "Спасибо. Теперь самое важное — давайте разберём, как устроена одна
> типичная сделка."

→ Phase 2.

---

## Phase 2 — Business Flow (20-30 min, main phase)

**Goal:** reconstruct the actor graph of a typical deal at the client. Compute
Industry Pack match %. Identify delta.
**Lead.state:** `BUSINESS_FLOW`.
**Exit event:** `Phase.Flow.Completed` + create 6-12 CaseProfile actors + run
`match_industry_pack` logic.

### Opening

> "Теперь самое важное — давайте разберём, как устроена одна типичная
> сделка. Расскажите по шагам, не торопясь: клиент пришёл / прислал заявку
> / открыл сайт — что происходит дальше? Я буду переспрашивать там, где
> нужно уточнить. В конце нарисую вам граф состояний — посмотрите,
> правильно ли я понял."

### Listen first, then probe

Let the client describe in their own words for 5-10 minutes. **Don't
interrupt** unless they go completely off topic. Take mental notes on:

- Who is the customer (B2C / B2B / mixed)
- Channels of acquisition (showroom / online / phone / referral)
- Payment timing (prepayment / postpayment / split)
- Vendors/suppliers (if physical goods)
- Production lead times (if custom or made-to-order)
- Logistics (own / outsourced / mixed)
- Fiscal docs (ПРРО / receipt / invoice)
- Returns / disputes

### After their initial story — draw the ASCII state graph

Reconstruct what you heard as a state machine:

```
NEW → PREPAID → SUPPLIER_NOTIFIED → IN_PRODUCTION → READY → 
  → FULLY_PAID → AT_WAREHOUSE → SHIPPED → DELIVERED → CLOSED
```

(Adapt to the actual client's flow — services may have `INQUIRY → BRIEF →
IN_PROGRESS → REVIEW → DELIVERED → INVOICED`; B2B may have `LEAD → QUALIFIED
→ NEGOTIATION → CONTRACT → DELIVERY → CLOSED_WON`).

Show it to the client:

> "Вот как я понял ваш типичный заказ. Где я ошибся или упростил?"

Iterate 1-3 times until client confirms.

### Probe by industry — the 8-case frame

Use these 8 case patterns as a checklist of what to ask about (skip
non-applicable, add custom cases as needed):

| Case | What to clarify | Account to fill |
|---|---|---|
| 1. Customer-facing entry point | Кто оформляет заказ, какие связи между менеджером / точкой / клиентом? | `case_1_flow` (CaseProfile actor) |
| 2. Initial payment / prepayment | Когда и как берётся первый платёж? ПРРО? | `case_2_flow` + `prro_provider` |
| 3. Vendor / supplier interaction | Как отправляется заказ поставщику? Какая цена/срок? | `case_3_flow` + `vendor_communication_pattern` |
| 4. Lead time management | Кто следит за сроком? Что при просрочке? | `case_4_flow` |
| 5. Final payment | Когда берётся доплата? Уведомление? | `case_5_flow` |
| 6. Logistics & warehouse | Кто везёт? Транзитный склад или постоянный? | `case_6_flow` + `warehouse_model` |
| 7. Deal closing / invoicing | Какие документы подписываются? Кто? | `case_7_flow` |
| 8. Returns / disputes | Как часто? Через ПРРО или отдельно? | `case_8_flow` + `returns_frequency` |

If a case **doesn't apply** to the client's industry (e.g. services have no
vendor / warehouse / returns) — explicitly note: `case_X_flow: "n/a"` rather
than leaving empty. This is important for Quality Gate 2.

If client describes **a case not in this list** (e.g. installation,
contract renewals, subscription billing) — create a CaseProfile with a
custom `case_name` and number it sequentially. Don't force-fit.

### Create CaseProfile actors

For each case the client described (covered, n/a, or custom):

```json
{
  "id": "CaseProfile.<client-slug>.<case_slug>",
  "type": "CaseProfile",
  "parent_lead_id": "Lead.<client-slug>",
  "case_number": 1,
  "case_name": "Розничная продажа в салоне",
  "status": "covered",          // or "n_a" or "needs_clarification"
  "flow_text": "клиент в салоне → менеджер оформляет Order → ...",
  "confidence_pct": 85,
  "identified_state_machine": "NEW → PREPAID → ...",
  "delta_required": []
}
```

### Compute Industry Pack match

After all cases collected:

1. Determine which existing pack best fits (`furniture_retail`,
   `services_studio`, `auto_parts`, `wholesale_fmcg`, `horeca`,
   `light_manufacturing`, `apparel`, `pharmacy`, `construction_materials`,
   etc.).
2. Compute `industry_pack_match_pct` = `(covered_cases / total_pack_cases) * 100`.
3. If novel actors are identified — add to `delta_required[]` on Lead.

> "Ваш кейс хорошо ложится на наш Industry Pack `<pack_name>` — покрытие
> ~<X>%. Дельта (то, что нужно докрутить под вас): <delta>. На финальной
> встрече solution architect это финализирует."

→ Phase 3.

---

## Phase 3 — Volumes & Infra (10 min)

**Goal:** объёмы для sizing, интеграции для DataMigrator/ProcessMigrator.
**Lead.state:** `VOLUMES_INFRA`.
**Exit event:** `Phase.Volumes.Completed` + create IntegrationMention actors.

### Opening — embedded input form

This phase is **mostly quantitative**. Don't ask each question one by one —
emit a batch form:

> "Объёмы и инфраструктура. Чтобы не переспрашивать каждое поле — заполните
> блок ниже, потом продолжим разговор."

```
Volumes & Infrastructure:
- Заказов/сделок в месяц:        [___]
- Активных контрагентов:         [___]
- Активных поставщиков:          [___]
- SKU в каталоге:                [___]
- Одновременных пользователей:   [___]
- ПРРО:                          [___]
- Банки (через запятую):         [___]
- Логистика (через запятую):     [___]
- Выгрузка БД возможна?          [Да полная | Да CSV/dump | Частично | Нет]
- Есть ли дубли в базе?          [Нет | Немного | Да много]
- Кто администрирует?            [Внутренний IT | Аутсорс | Никто]
```

After they fill — confirm and create IntegrationMention actors for each
external system mentioned.

### Create IntegrationMention actors

For each provider mentioned (ПРРО, каждый банк, каждый перевозчик,
M.E.Doc/Вчасно/etc.):

```json
{
  "id": "IntegrationMention.<client-slug>.<provider-slug>",
  "type": "IntegrationMention",
  "parent_lead_id": "Lead.<client-slug>",
  "provider_name": "Cash Online",
  "provider_type": "prro",          // or "bank", "carrier", "accounting", "messaging"
  "status": "mentioned",
  "integration_pattern": "file",    // or "api", "manual", "webhook"
  "criticality": "critical"         // or "important", "nice"
}
```

### Follow-up on signals

If `data_dirtiness_flag` = "Да много" — ask one follow-up:
> "А почему так много дублей? Менеджеры не используют поиск?"

This typically reveals a process issue → flag for DataMigrator dedupe phase
recommendation.

→ Phase 4.

---

## Phase 4 — Specifics (15-20 min)

**Goal:** боли, ограничения, decision-maker, бюджет, эскалационные сигналы.
**Lead.state:** `SPECIFICS`.
**Exit event:** `Phase.Specifics.Completed` + create Requirement and PriorAttempt
actors.

### Opening

> "Теперь поговорим про специфику. Что в текущей системе вы НЕ хотите
> потерять при переходе?"

Listen. For each item → create a Requirement actor:

```json
{
  "id": "Requirement.<client-slug>.<req-slug>",
  "type": "Requirement",
  "parent_lead_id": "Lead.<client-slug>",
  "requirement_text": "фильтрация заказов по поставщикам",
  "source": "must_keep",            // or "owner_report", "success_criteria"
  "criticality": "must",            // or "should", "nice"
  "status": "novel"                 // pending similarity match later
}
```

### Then ask top_pain

> "Что вас в текущей системе бесит больше всего?"

Update Lead: `top_pain`.

### Owner reports

> "Какие отчёты собственник смотрит сейчас? Воронка, выручка, P&L?"

Each report → Requirement actor with `source: "owner_report"`.

### Decision making

| Question | Account |
|---|---|
| Кто owner проекта внутри компании? | `internal_owner` |
| Кто принимает финальное решение (подписывает контракт)? | `decision_maker` |
| Бюджет — какой диапазон? | `budget_range` |
| Были ли прежние попытки миграции? | `previous_attempts` |
| Если да — как закончилось? | → PriorAttempt actor |
| Критерий успеха через год? | `success_criteria` |

### PriorAttempt actor (if failed migration mentioned)

```json
{
  "id": "PriorAttempt.<client-slug>.<sys-slug>",
  "type": "PriorAttempt",
  "parent_lead_id": "Lead.<client-slug>",
  "system_name": "BAS Бухгалтерія",
  "attempted_at": "2025-Q1",
  "outcome": "failed",
  "reason_failed": "слишком похоже на 1С, менеджеры не приняли",
  "lessons_learned": "нужна смена UX paradigm",
  "escalation_weight": 7
}
```

If `escalation_weight >= 7` — set Lead `escalation_signals[] += "previous_failed_migration"` and flag for SA review (next BD).

### Watch for escalation signals throughout

See `quality-gates.md` §G7 for the 8 triggers. Common ones:

- Regulated industry (banking, healthcare, gov) → escalate immediately
- >200 employees → escalate immediately
- Emotional negative / frustrated tone → escalate immediately
- Internal conflict between decision-makers → next BD
- Complex specifics (>40% pack delta) → next BD
- Legal questions about contracts → next BD

→ Phase 5.

---

## Phase 5 — Presentation (5-10 min)

**Goal:** показать результат, классифицировать трек, создать Prototype,
сгенерировать roadmap_graph, собрать signoff.
**Lead.state:** `PRESENTATION` → `BRIEF_GENERATED` → `AWAITING_HUMAN_HANDOFF`.
**Exit events:** `Prototype.Created`, `Brief.Generated`, `Roadmap.Approved`.

### Step 1 — Summary

> "Спасибо за подробный разговор. Резюме:
> • <Industry>, <outlets_count> точек/филиалов, <employees_count> сотрудников
> • <orders_per_month> сделок/мес, <vendors_count> поставщиков
> • Текущая система: <current_system>
> • Главная боль: <top_pain>
> • Дедлайн: <target_deadline>"

### Step 2 — Classify track

Run the classification logic:

```
if employees > 200 OR regulated_industry OR multi_legal_entity:
    track = "L4 Custom"
elif employees > 100 OR industry_pack_match_pct < 60 OR has_complex_specifics:
    track = "L3 Custom"
elif outlets <= 5 AND employees <= 30 AND industry_pack_match_pct >= 85 AND has_internal_it_owner:
    track = "L1 Express"
elif no_legacy_system_to_replace:  # Greenfield
    track = "L1 Greenfield"
else:
    track = "L2 Standard"
```

Present:

> "Я отношу вас к треку **<L1/L2/L3/L4>** — это <2-4 / 6-10 / 10-16 / 16+>
> недель, команда <2-3 / 3-4 / 5+ человек>, <X> EUR onboarding + <Y> EUR/мес
> подписки."

### Step 3 — Confirm deployment mode

> "Какой деплоймент — Cloud, Hybrid, On-premise или Air-gapped?"

If regulated industry / banking → recommend Hybrid+. Otherwise default Cloud.
If user says Hybrid+ — `DataMigrator` and `ValidationAgent` MUST run on-prem
(per Master Spec v1.4 critical rule).

Lead: `deployment_mode`, refine `target_cutover_date`.

### Step 4 — Custom phase signals

Ask one last open-ended question:

> "Есть ли особые обстоятельства, которые повлияют на план миграции?
> Например: одобрение совета директоров, регуляторная проверка, летние
> каникулы, фазированный cut-over по модулям?"

Each signal → adds a custom Phase actor to the roadmap. See
`actor-schemas.md` → Phase for the patterns:
- "Board approval" → custom Phase before Cutover
- "Regulatory review" → blocked Phase in time window
- "Planned pause" → PlannedPause Phase
- "Staged cutover" → multiple Cutover phases instead of one
- "Security audit" → SecurityReview Phase before Profiling

### Step 5 — Create Prototype

```json
{
  "id": "Prototype.<client-slug>",
  "type": "Prototype",
  "parent_lead_id": "Lead.<client-slug>",
  "industry_pack_actor_id": "<matched_industry_pack>",
  "stand_url": "demo.simulator.company/p/<slug>-<random>",
  "expires_at": "<today + 14 days>",
  "status": "active",
  "sample_data_hint": "5 sample customers, 3 sample vendors, 15 SKU"
}
```

> "Создал ваш персональный демо-стенд: <stand_url>. TTL 14 дней."

### Step 6 — Generate roadmap_graph

Based on track, create:

1. **Migration container actor** (1):
```json
{
  "id": "Migration.<client-slug>",
  "type": "Migration",
  "client_name": "...",
  "track": "L2",
  "deployment_mode": "cloud",
  "start_date": "...",
  "target_cutover_date": "...",
  "overall_status": "ROADMAP_GENERATED"
}
```

2. **Phase actors** (9-15, depending on track):

| Track | Phases |
|---|---|
| L1 Greenfield | Discovery, Pack Selection, Data Init, Setup, Training, Go-Live, Operate, Evolve |
| L2 Single replacement | Discovery, Profiling, Mapping, Data Migration, Process Migration, Validation, Cutover, Operate, Evolve |
| L3 Multi-system | + N parallel Profiling phases, Synthesis, Staged Cutover |
| L4 Enterprise | + Pre-engagement Security Review, On-prem setup, Compliance audit, Extended validation, Board approval |

Add any **custom phases** from Step 4 signals.

Each Phase actor:
```json
{
  "id": "Phase.<client-slug>.<phase-slug>",
  "type": "Phase",
  "parent_migration_id": "Migration.<client-slug>",
  "phase_name": "Profiling",
  "owner_agent": "SystemProfilerAgent",
  "depends_on": ["Phase.<client-slug>.Discovery"],
  "target_start_date": "...",
  "target_end_date": "...",
  "signoffs_required": [{"role": "SA", "person": null}],
  "status": "pending"   // first phase is "ready", Discovery is "completed"
}
```

### Step 7 — Show roadmap and collect signoff

> "Создал ваш персональный Migration Roadmap. <N> фаз, target cutover
> <date>. Каждая фаза — отдельный актор с owner-агентом и signoffs.
> Подтвердите — и я передаю вас solution architect'у."

If client confirms → emit event `Roadmap.Approved` (signer: product_owner).
Set Lead `state: AWAITING_HUMAN_HANDOFF`.

### Step 8 — Final write

Write final `actor-graph.json` with all actors. Output summary message (per
SKILL.md spec) and stop.

> "Discovery завершён. Solution architect <SA_name> получит уведомление и
> свяжется с вами в течение 1 рабочего дня. Хорошего дня!"

---

## Cross-cutting: when things go off-script

- **Client takes a pause** — write current state, set
  `_meta.session_paused_at`, tell client they can resume via the same link.
- **Client says "я не AI клиент, я хочу пройти симуляцию"** — switch to
  symbolic mode: you play both sides using sensible defaults.
- **Client refuses to share specific data** (e.g. employees_count, banks) —
  record as `null` with note `"client_declined_to_share"` and proceed if
  Quality Gates allow.
- **Operator interrupts you** — pause, listen, then resume from where you
  stopped.
- **Major escalation trigger fires** — stop the flow, write
  `escalation.json`, tell client SA will call them, exit.

---

**Version:** see `../SKILL.md` frontmatter (single source).

**Source spec mapping:**
- Phase 1-5 detailed prompts and account mappings — `source-spec/02_discovery_agent_spec.md` §4 «Детальные промпты по этапам»
- 8-case probing frame — `source-spec/02_discovery_agent_spec.md` §4 (Этап 2 «Опросный фрейм по 8 кейсам Golden House»)
- Industry Pack matching with delta — `source-spec/02_discovery_agent_spec.md` §5, plus `source-spec/03_discovery_agent_kb_bundle.md` §1.2 (Pack catalog) and §1.2a (Onramp Process pack signals)
- Hybrid form vs free dialog — operational decision based on hybrid analysis (added in prototype work; see also `discovery_form_tree.html` artifact for full form structure)
- classify_track logic — `source-spec/02_discovery_agent_spec.md` §8 «Логика классификации трека»
- Phase 5 roadmap_graph generation — `source-spec/01_master_spec.md` Part 3.8 «Roadmap mechanics» + Part 1.2.1 (Phase actor schema)
- Custom phase triggers (Board approval / NDS / staged cutover etc.) — `source-spec/03_discovery_agent_kb_bundle.md` §1.2a (signals list)
- Brief template — `source-spec/02_discovery_agent_spec.md` §10 «Шаблон Discovery Brief»
- Walked example — `source-spec/05_golden_house_simulation.md` end-to-end
