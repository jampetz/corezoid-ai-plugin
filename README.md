# Corezoid — Claude Code & Codex Plugin

A plugin for [Claude Code](https://claude.ai/code) and [Codex](https://codex.openai.com) that connects the [Corezoid](https://corezoid.com) BPM platform to Claude via MCP. Claude gets direct access to Corezoid processes and deep platform knowledge to create, edit, review, and deploy workflows through natural conversation.

## What it does

The plugin bundles a Go MCP server that exposes Corezoid operations as MCP tools and provides specialist skills that teach Claude the platform model and common workflows:

| Skill                          | Activate with                            | Covers                                            |
|--------------------------------|------------------------------------------|---------------------------------------------------|
| `corezoid`                     | "Corezoid", "process", "conv.json"       | Full platform overview, all node types, MCP tools |
| `corezoid-init`                | "set up", "login", "pull workspace"      | OAuth login, workspace pull, environment setup    |
| `corezoid-create`              | "create a process", "new process"        | Building processes from scratch                   |
| `corezoid-edit`                | "edit", "modify", "update" a process     | Modifying existing `.conv.json` files             |
| `corezoid-review`              | "review", "audit", "check" a process     | Analysis, dead code, best-practice violations     |
| `corezoid-project-review`      | "review project", "audit folder"         | Cross-process audit of an entire folder           |
| `corezoid-dashboard-manager`   | "create dashboard", "add chart", "visualize metrics" | Dashboards, charts, node metrics, real-time monitoring |
| `corezoid-process-tech-writer` | "document", "write docs", "describe process" | Markdown docs + enriched JSON with node descriptions |

## Requirements

- [Claude Code](https://claude.ai/code) or [Codex](https://codex.openai.com) installed
- [Go 1.21+](https://go.dev/dl/) available in `PATH` (the MCP server runs via `go run`, no build step needed)
  ```bash
  brew install golang        # macOS
  sudo apt install golang    # Ubuntu/Debian
  ```
- A Corezoid account

## Installation

### Claude Code
 
**From the GitHub marketplace:**

```bash
claude plugin marketplace add corezoid/corezoid-ai-plugin
claude plugin install corezoid@corezoid
```

**Or from a local clone:**

```bash
git clone https://github.com/corezoid/corezoid-ai-plugin
claude plugin marketplace add ./corezoid-ai-plugin
claude plugin install corezoid@corezoid
```

### Codex

**From the GitHub marketplace:**

```bash
codex plugin marketplace add corezoid/corezoid-ai-plugin
codex plugin install corezoid@corezoid
```

**Or from a local clone:**

```bash
git clone https://github.com/corezoid/corezoid-ai-plugin
codex plugin marketplace add ./corezoid-ai-plugin
codex plugin install corezoid@corezoid
```

No build step, no extra setup. The MCP server starts automatically on first use.

### Updating

```bash
claude plugin update corezoid   # Claude Code
codex plugin update corezoid    # Codex
```

Restart Claude Code / Codex after updating to apply the new version.

## Authentication

On the first Corezoid operation Claude detects that no token is present and runs the `login` tool automatically — your browser opens for OAuth2 sign-in and the session continues without interruption.

The token is saved to `.env` in your working directory and reused on every subsequent session. When it expires, the login flow triggers again automatically.

You can also trigger login manually at any time:

```
log in to Corezoid
```

### Static token (optional)

If you prefer to manage the token yourself, set it in `.env` or export it before starting Claude Code or Codex:

```bash
export SIMULATOR_TOKEN=your_token_here
```

## Configuration

| Environment variable       | Required | Description                                       |
|----------------------------|----------|---------------------------------------------------|
| `SIMULATOR_TOKEN`          | No       | Static token — overrides OAuth2 saved credentials |
| `COREZOID_API_URL`         | No       | Override the default Corezoid API base URL        |
| `COREZOID_WORKSPACE_ID`    | No       | Default workspace ID                              |
| `COREZOID_STAGE_ID`        | No       | Default stage ID                                  |
| `COREZOID_APIGW_URL`       | No       | Override the API Gateway URL                      |

## Usage

Once installed, just talk to Claude naturally:

```
Pull my Corezoid workspace and show me what processes are in the Payments folder.
```

```
Create a process that calls any weather API, handles errors,
and sends the forecast back to the caller.
```

```
Create a folder named "Services" in Corezoid.
```

```
Edit the "payment" process — add retry logic on API timeout
with exponential backoff up to 3 attempts.
```

```
Edit process with id 1278273  — add retry logic
on API timeout with exponential backoff up to 3 attempts.
```

```
Review process ID 2778176 for dead nodes, missing error handlers,
and hardcoded values.
```

```
Review the process at 1278273_Business.folder/2778176_payment.conv.json
for dead nodes, missing error handlers, and hardcoded values.
```

```
Push the updated payment process to Corezoid and run a test task with
{"amount": 100, "currency": "USD"}.
```

```
Audit the entire Payments folder — list all processes, check for
validation errors, and summarize what each process does.
```

## MCP Tools

| Tool                | Description                                        |
|---------------------|----------------------------------------------------|
| `login`             | Authenticate via OAuth2 (opens browser)            |
| `logout`            | Remove saved credentials                           |
| `list-workspaces`   | List available workspaces and stages               |
| `list-stages`       | List stages in a workspace                         |
| `list-projects`     | List folders and processes in a stage              |
| `pull-folder`       | Export an entire folder/stage to local files       |
| `pull-process`      | Export a single process to a `.conv.json` file     |
| `push-process`      | Validate and deploy a `.conv.json` to Corezoid     |
| `lint-process`      | Validate process structure locally (no API call)   |
| `run-task`          | Send a task to a deployed process                  |
| `list-node-tasks`   | List tasks currently sitting in a node             |
| `list-task-history` | Show task execution history                        |
| `delete-task`       | Remove a task from a node                          |
| `modify-task`       | Update task parameters                             |
| `create-process`    | Create a new empty process in a folder             |
| `create-folder`     | Create a new subfolder                             |
| `create-alias`      | Create a short alias for a process                 |
| `create-variable`   | Create a Corezoid environment variable             |
| `create-dashboard`  | Create a new dashboard for visualizing node metrics |
| `get-dashboard`     | Get a dashboard with its charts and series         |
| `add-chart`         | Add a chart (column, pie, funnel, table) to a dashboard |
| `modify-chart`      | Modify an existing chart (full series replace)     |
| `get-chart`         | Get a single chart with its series data            |
| `set-dashboard-layout` | Save chart positions on a dashboard grid        |

## Architecture

```
Claude Code / Codex
  └── corezoid MCP server (go run .)
        ├── Auth          login, logout
        ├── Workspace     list-workspaces, list-stages, list-projects
        ├── Processes     pull-process, pull-folder, push-process, lint-process
        │                 create-process, create-folder, create-alias, create-variable
        ├── Tasks         run-task, list-node-tasks, list-task-history
        │                 modify-task, delete-task
        └── Dashboards    create-dashboard, get-dashboard, add-chart,
                          modify-chart, get-chart, set-dashboard-layout
```

## Project structure

```
corezoid-ai-plugin/
├── .claude-plugin/
│   └── marketplace.json         # Claude Code marketplace listing (points to plugins/corezoid)
├── .agents/
│   └── plugins/
│       └── marketplace.json     # Codex marketplace listing (points to plugins/corezoid)
├── plugins/corezoid/            # Plugin root (CLAUDE_PLUGIN_ROOT for both Claude Code and Codex)
│   ├── .claude-plugin/
│   │   └── plugin.json          # Claude Code plugin manifest
│   ├── .codex-plugin/
│   │   └── plugin.json          # Codex plugin manifest
│   ├── .mcp.json                # MCP server configuration
│   ├── mcp-server/              # Go MCP server source
│   ├── skills/
│   │   ├── corezoid/                    # Universal assistant skill
│   │   ├── corezoid-init/               # Environment setup skill
│   │   ├── corezoid-create/             # Process creation skill
│   │   ├── corezoid-edit/               # Process editing skill
│   │   ├── corezoid-review/             # Process review skill
│   │   ├── corezoid-project-review/     # Project audit skill
│   │   ├── corezoid-dashboard-manager/  # Dashboard & chart management skill
│   │   └── corezoid-process-tech-writer/ # Process documentation skill
│   ├── docs/                    # Node and process documentation
│   └── samples/                 # Example .conv.json processes
```

## Links

- [Corezoid](https://corezoid.com)
- [Claude Code](https://claude.ai/code)

## License

ISC
