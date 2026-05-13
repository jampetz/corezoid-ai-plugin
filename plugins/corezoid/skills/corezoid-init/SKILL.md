---
name: corezoid-init
description: >
  Corezoid environment setup specialist. Use when the user wants to connect to
  Corezoid, set up credentials, authenticate, pull a project, configure the
  environment, or start working with a Corezoid project for the first time.
  Activate when the user says "init", "setup", "connect to corezoid", "login",
  "pull workspace", "configure environment", or "get started".
---

# Initialize Corezoid Environment

You are a specialist in setting up the Corezoid working environment using the `corezoid` MCP server.

## Step 1 — Call `login`

Call MCP tool **`login`** with no arguments. It will guide setup in one of two modes depending on whether the client supports MCP elicitation.

---

## Mode A — Elicitation supported (interactive forms)

The `login` tool handles everything automatically in sequence:

1. **API URL prompt** — interactive form asking for `COREZOID_ACCOUNT_URL`
2. **OAuth2** — browser window opens for authentication, token saved to `.env`
3. **Workspace picker** — fetches available workspaces and shows a dropdown, saves `COREZOID_WORKSPACE_ID` to `.env`
4. **Stage picker** — lists projects then stages for selection, saves `COREZOID_STAGE_ID` to `.env`

When `login` returns "Setup complete", proceed to **Step 2**.

---

## Mode B — Elicitation not supported (chat-based collection)

When elicitation is unavailable, drive the setup yourself using explicit tool calls. Follow this sequence:

### B1 — Collect Account URL

→ Ask the user: **"What is your Corezoid Account URL? (e.g. https://account.corezoid.com)"**

→ Call `login(account_url=<value>)`

The tool opens a browser for OAuth2 authentication and saves the token.

### B2 — Select Workspace

→ Call **`list-workspaces`**

→ Show the workspace list to the user and ask which one to use.

### B3 — Select Project

→ Call **`list-projects(company_id=<workspace_id>)`**

→ Show the project list to the user and ask which project they want to use.

### B4 — Select Stage

→ Call **`list-stages(project_id=<id>, company_id=<workspace_id>)`**

→ Show the stage list to the user and ask which stage (root folder) to use.

### B5 — Commit selection

→ Call `login(workspace_id=<workspace_id>, stage_id=<stage_id>)`

When `login` returns "Setup complete", proceed to **Step 2**.

---

## Step 2 — Pull the project

After `login` returns "Setup complete", call MCP tool **`pull-folder`** with:
- `folder_id`: value of `COREZOID_STAGE_ID` (now set in `.env`)

Do not proceed until the tool returns successfully.

---

## Exception: user provides values directly

If the user explicitly pastes values, write them to `.env` and skip the corresponding prompts:

```
COREZOID_API_URL=<value>
COREZOID_WORKSPACE_ID=<value>
COREZOID_STAGE_ID=<value>
```

Then call `login` — it will skip already-set values and only prompt for what's missing.

---

## Variables reference

| Variable | Set during |
|---|---|
| `COREZOID_ACCOUNT_URL` | login step 1 — API URL prompt |
| `SIMULATOR_TOKEN` | login step 2 — OAuth2 |
| `COREZOID_WORKSPACE_ID` | login step 3 — workspace selection |
| `COREZOID_STAGE_ID` | login step 4 — stage selection |
