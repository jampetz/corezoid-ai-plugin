"""Corezoid MCP Server — exposes CorezoidClient methods as MCP tools."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Any

# Resolve workspace root (parent of this server directory) for imports and .env
_workspace_root = str(Path(__file__).resolve().parent.parent)
sys.path.insert(0, _workspace_root)

from dotenv import load_dotenv
from mcp.server.fastmcp import FastMCP

from corezoid_client import CorezoidClient

# Load .env from workspace root
_env_path = Path(_workspace_root) / ".env"
load_dotenv(_env_path)

_api_login = os.environ["API_LOGIN"]
_secret = os.environ["SECRET"]
_base_url = os.environ["BASE_URL"]
_company_id = os.environ["COMPANY_ID"]

client = CorezoidClient(
    api_login=_api_login,
    secret=_secret,
    base_url=_base_url,
    company_id=_company_id,
)

mcp = FastMCP("corezoid")


def _minify(obj: Any) -> str:
    """Return a compact JSON string — avoids FastMCP's indent=2 re-serialization."""
    return json.dumps(obj, separators=(",", ":"), ensure_ascii=False)


def _error_response(exc: Exception) -> str:
    """Return a compact JSON error string without leaking internals."""
    return _minify({"error": type(exc).__name__, "message": str(exc)})


def _serialise_deps(deps: dict) -> str:
    """Convert sets inside crawl_dependencies output to lists and minify."""
    out: dict[str, Any] = {}
    for pid, info in deps.items():
        entry = dict(info)
        if isinstance(entry.get("sub_deps"), set):
            entry["sub_deps"] = sorted(entry["sub_deps"])
        out[str(pid)] = entry
    return _minify(out)


# ---------------------------------------------------------------------------
# MCP Tools
# ---------------------------------------------------------------------------


@mcp.tool()
def get_process_details(process_id: int) -> str:
    """Get metadata for a Corezoid process by its ID."""
    try:
        return _minify(client.get_process_details(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def get_process_scheme(process_id: int) -> str:
    """Get the full process scheme (nodes, connections, logics) for a process."""
    try:
        return _minify(client.get_process_scheme(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_folder_contents(folder_id: int = 0) -> str:
    """List contents of a Corezoid folder. Use folder_id=0 for root."""
    try:
        return _minify(client.list_folder_contents(folder_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_projects(sort: str = "title") -> str:
    """List all projects (top-level workspaces) visible to the current API key.

    Returns each project's obj_id (project_id), title, short_name, childs count,
    size (process count), and privs. Use the returned obj_id as the folder_id in
    list_folder_contents to browse inside a project.

    Args:
        sort: Sort field — "title" (default, alphabetical) or "date".
    """
    try:
        all_projects: list = []
        offset = 0
        limit = 200
        while True:
            result = client.list_projects(sort=sort, limit=limit, offset=offset)
            batch = result["ops"][0].get("list", [])
            all_projects.extend(batch)
            if len(batch) < limit:
                break
            offset += limit
        return _minify({"projects": all_projects, "total": len(all_projects)})
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_aliases(project_id: int, stage_id: int) -> str:
    """List all aliases in a project stage."""
    try:
        return _minify(client.list_aliases(project_id, stage_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def resolve_aliases(project_id: int, stage_id: int, alias_names: list[str]) -> str:
    """Resolve alias short_names to their target process IDs."""
    try:
        return _minify(client.resolve_aliases(project_id, stage_id, alias_names))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def create_alias(
    project_id: int,
    stage_id: int,
    short_name: str,
    title: str | None = None,
    description: str = "",
) -> str:
    """Create a new alias in a project stage."""
    try:
        return _minify(client.create_alias(project_id, stage_id, short_name, title, description))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def link_alias(alias_id: int, process_id: int) -> str:
    """Link an existing alias to a process."""
    try:
        return _minify(client.link_alias(alias_id, process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def create_and_link_alias(
    project_id: int,
    stage_id: int,
    process_id: int,
    short_name: str | None = None,
    title: str | None = None,
    description: str = "",
) -> str:
    """Create an alias and link it to a process in one step."""
    try:
        return _minify(client.create_and_link_alias(
            project_id, stage_id, process_id, short_name, title, description,
        ))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def modify_node(
    process_id: int,
    node_id: str,
    title: str | None = None,
    description: str | None = None,
    logics: list | None = None,
) -> str:
    """Modify a node's title, description, or logics in a process."""
    try:
        return _minify(client.modify_node(
            process_id, node_id, title=title, description=description, logics=logics,
        ))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def modify_node_code(
    process_id: int,
    node_id: str,
    new_code: str,
    title: str | None = None,
    description: str | None = None,
    commit: bool = True,
) -> str:
    """Modify the JS/Erlang code inside a Corezoid node."""
    try:
        return _minify(client.modify_node_code(
            process_id, node_id, new_code,
            title=title, description=description, commit=commit,
        ))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def batch_modify_nodes(
    process_id: int,
    modifications: list,
    commit: bool = True,
) -> str:
    """Apply multiple node modifications sequentially, then optionally commit."""
    try:
        return _minify(client.batch_modify_nodes(process_id, modifications, commit=commit))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def commit(process_id: int) -> str:
    """Commit pending changes to a process."""
    try:
        return _minify(client.commit(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def crawl_dependencies(root_pid: int, project_id: int, stage_id: int) -> str:
    """BFS-crawl all process dependencies starting from a root process ID."""
    try:
        raw = client.crawl_dependencies(root_pid, project_id, stage_id)
        return _serialise_deps(raw)
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def make_request(payload: dict) -> str:
    """Send a raw authenticated payload to the Corezoid API."""
    try:
        return _minify(client.make_request(payload))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def create_task(process_id: int, data: dict, ref: str | None = None) -> str:
    """Create (send) a new task to a Corezoid process.
    Provide input parameters in 'data'. Optionally set a unique 'ref' for later lookup."""
    try:
        return _minify(client.create_task(process_id, data, ref=ref))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def show_task(process_id: int, task_id: str | None = None, ref: str | None = None) -> str:
    """Retrieve a task's current state (data, node, status) by task_id and/or ref.
    At least one of task_id or ref must be provided."""
    try:
        return _minify(client.show_task(process_id, task_id=task_id, ref=ref))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_task_history(process_id: int, task_id: str) -> str:
    """Return the execution history (node path) for a task.
    Only task_id (obj_id) is supported — ref cannot be used.
    Note: 'data' in each history entry is always null; use show_task for current data."""
    try:
        return _minify(client.list_task_history(process_id, task_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_node_tasks(process_id: int, node_id: str, limit: int = 50, offset: int = 0) -> str:
    """Return tasks currently sitting in a specific node of a process.
    ops[0]['list'] contains task objects with task_id, ref, and data.
    ops[0]['count'] is the total number of tasks in the node."""
    try:
        return _minify(client.list_node_tasks(process_id, node_id, limit=limit, offset=offset))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def modify_task(process_id: int, data: dict, task_id: str | None = None, ref: str | None = None) -> str:
    """Modify an existing task's data to continue flow through the process.
    At least one of task_id or ref must be provided.
    Only the keys in 'data' are updated; existing keys are preserved.
    The task continues the flow from the node where it was modified with the updated data."""
    try:
        return _minify(client.modify_task(process_id, data, task_id=task_id, ref=ref))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def delete_task(process_id: int, task_id: str | None = None, ref: str | None = None) -> str:
    """Delete a task from a process.
    At least one of task_id or ref must be provided.
    If only ref is given, the task is looked up first to resolve task_id and node_id."""
    try:
        return _minify(client.delete_task(process_id, task_id=task_id, ref=ref))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def copy_process(process_id: int, title: str | None = None, folder_id: int | None = None) -> str:
    """Copy (clone) a Corezoid process.
    Defaults to same folder and '<title> (Copy)' if not specified."""
    try:
        return _minify(client.copy_process(process_id, title=title, folder_id=folder_id))
    except Exception as exc:
        return _error_response(exc)


# ---------------------------------------------------------------------------
# Analysis Tools
# ---------------------------------------------------------------------------


@mcp.tool()
def analyze_process_structure(process_id: int) -> str:
    """Analyze process structure: node counts by type, logic type distribution,
    untitled node list, and basic stats. Use as a first step in any process review."""
    try:
        return _minify(client.analyze_process_structure(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_nodes_missing_semaphors(process_id: int) -> str:
    """Find nodes that should have a timeout semaphor but don't.
    Groups by severity: critical (api_callback — tasks hang forever),
    high (api — HTTP calls), low (api_rpc/api_copy sync)."""
    try:
        return _minify(client.find_nodes_missing_semaphors(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def extract_code_nodes(process_id: int) -> str:
    """Extract all JS/Erlang code nodes with source code and quality flags:
    has_try_catch, has_commented_code, hardcoded_urls."""
    try:
        return _minify(client.extract_code_nodes(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def list_external_dependencies(process_id: int) -> str:
    """List all outbound process references: api_rpc conv_ids, api_copy conv_ids
    with modes, and conv[@alias] state store reads from set_param."""
    try:
        return _minify(client.list_external_dependencies(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_hardcoded_values(process_id: int) -> str:
    """Scan for hardcoded URLs in api nodes, numeric conv_ids that should be
    aliases, and literal string values in api/rpc extra fields."""
    try:
        return _minify(client.find_hardcoded_values(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_duplicate_patterns(process_id: int) -> str:
    """Detect duplicate set_param patterns and duplicate @send-message call
    signatures (same text_id + attachment_id) across nodes."""
    try:
        return _minify(client.find_duplicate_patterns(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_untitled_nodes(process_id: int) -> str:
    """Return untitled nodes grouped by obj_type with flow context: what feeds
    into each node and what it feeds into. Useful for naming audit."""
    try:
        return _minify(client.find_untitled_nodes(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def validate_escalation_chains(process_id: int) -> str:
    """Verify every escalation node (obj_type 3) routes correctly:
    crash/timeout errors to retry delays, access_denied to final error nodes,
    and that all have a default fallthrough."""
    try:
        return _minify(client.validate_escalation_chains(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_orphaned_nodes(process_id: int) -> str:
    """BFS from Start node to find unreachable (orphaned) nodes.
    Detects dead code that no task can ever reach. Returns reachable count,
    orphaned count, and list of orphaned nodes with id/title/type."""
    try:
        return _minify(client.find_orphaned_nodes(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def find_noop_nodes(process_id: int) -> str:
    """Detect functionally useless nodes: (1) no-op conditions where all
    branches route to the same destination, (2) set_param nodes that set
    variables never referenced by any downstream node."""
    try:
        return _minify(client.find_noop_nodes(process_id))
    except Exception as exc:
        return _error_response(exc)


@mcp.tool()
def check_variable_usage(process_id: int, variable_names: list[str]) -> str:
    """Check if variables (e.g. ['user_id', 'channel']) are referenced
    anywhere in a process. Scans all node logics, extras, conditions, and
    semaphors for {{variable_name}}. Returns per-variable results showing
    which nodes use each variable and how."""
    try:
        return _minify(client.check_variable_usage(process_id, variable_names))
    except Exception as exc:
        return _error_response(exc)


if __name__ == "__main__":
    mcp.run()
