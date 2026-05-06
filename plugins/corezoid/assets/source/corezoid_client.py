#!/usr/bin/env python3
import hashlib
import time
import json
import re
import urllib.request
import urllib.error

class CorezoidClient:
    def __init__(self, api_login, secret, base_url, company_id):
        self.api_login = api_login
        self.secret = secret
        self.base_url = base_url
        self.company_id = company_id
    
    def generate_signature(self, timestamp, content):
        """Generate SHA-1 signature for Corezoid API authentication"""
        signature_string = f"{timestamp}{self.secret}{content}{self.secret}"
        return hashlib.sha1(signature_string.encode()).hexdigest()
    
    def make_request(self, payload, endpoint="json"):
        """Make authenticated request to Corezoid API.

        Args:
            payload: The JSON payload to send.
            endpoint: API endpoint suffix. Default ``"json"``.
                      Use ``"copy"`` for process copy operations.
        """
        # Replace trailing path segment with the requested endpoint
        base = self.base_url
        if endpoint != "json":
            base = base.rsplit("/", 1)[0] + "/" + endpoint

        timestamp = int(time.time())
        content = json.dumps(payload, separators=(',', ':'))
        signature = self.generate_signature(timestamp, content)
        
        url = f"{base}/{self.api_login}/{timestamp}/{signature}"
        
        req = urllib.request.Request(
            url,
            data=content.encode('utf-8'),
            headers={'Content-Type': 'application/json'}
        )
        
        try:
            with urllib.request.urlopen(req) as response:
                return json.loads(response.read().decode('utf-8'))
        except urllib.error.HTTPError as e:
            return {"error": f"HTTP {e.code}", "message": e.read().decode('utf-8')}
    
    def get_process_details(self, process_id):
        """Fetch details of a specific process"""
        payload = {
            "ops": [{
                "type": "show",
                "obj": "conv",
                "obj_id": process_id,
                "company_id": self.company_id
            }]
        }
        return self.make_request(payload)
    
    def list_folder_contents(self, folder_id=0):
        """List contents of a folder (0 = root)"""
        payload = {
            "ops": [{
                "type": "list",
                "obj": "folder",
                "obj_id": folder_id,
                "company_id": self.company_id
            }]
        }
        return self.make_request(payload)

    def list_projects(self, sort="title", limit=200, offset=0):
        """List all projects (top-level workspaces) visible to the current API key.

        Args:
            sort:   Field to sort by — "title" (default) or "date".
            limit:  Max number of projects to return (default 200).
            offset: Pagination offset (default 0).

        Returns:
            Raw API response. The project list is at ops[0]["list"]; each entry
            contains obj_id (project_id), title, short_name, childs, size, privs.
        """
        payload = {
            "ops": [{
                "type": "list",
                "obj": "projects",
                "obj_id": 0,
                "id": self.company_id,
                "company_id": self.company_id,
                "sort": sort,
                "params": {"limit": limit, "offset": offset},
            }]
        }
        return self.make_request(payload)

    def list_aliases(self, project_id, stage_id):
        """List all aliases in a stage.
        
        Args:
            project_id: Numeric project ID
            stage_id: Numeric stage ID
            
        Returns:
            List of alias dicts with keys: short_name, obj_to_id, obj_to_type, title, etc.
        """
        payload = {
            "ops": [{
                "sort": "date",
                "order": "asc",
                "company_id": self.company_id,
                "project_id": project_id,
                "stage_id": stage_id,
                "obj": "aliases",
                "type": "list",
                "location": "stage",
                "id": self.company_id
            }]
        }
        result = self.make_request(payload)
        if result.get("request_proc") == "ok" and result["ops"][0].get("proc") == "ok":
            return result["ops"][0].get("list", [])
        return []

    def resolve_aliases(self, project_id, stage_id, alias_names):
        """Resolve a list of alias short_names to their target process IDs.
        
        Args:
            project_id: Numeric project ID
            stage_id: Numeric stage ID
            alias_names: List of alias short_names (without @)
            
        Returns:
            Dict mapping alias short_name -> {obj_to_id, obj_to_type, title}
        """
        all_aliases = self.list_aliases(project_id, stage_id)
        result = {}
        for a in all_aliases:
            sn = a.get("short_name", "")
            if sn in alias_names:
                result[sn] = {
                    "obj_to_id": a.get("obj_to_id"),
                    "obj_to_type": a.get("obj_to_type"),
                    "title": a.get("title", "")
                }
        return result

    def create_alias(self, project_id, stage_id, short_name, title=None, description=""):
        """Create an alias for a process.
        
        Args:
            project_id: Numeric project ID
            stage_id: Numeric stage ID
            short_name: Alias short name (used as @short_name in process references)
            title: Display title (defaults to short_name)
            description: Optional description
            
        Returns:
            API response dict
        """
        payload = {
            "ops": [{
                "obj": "alias",
                "title": title or short_name,
                "short_name": short_name,
                "description": description,
                "company_id": self.company_id,
                "stage_id": stage_id,
                "project_id": project_id,
                "type": "create"
            }]
        }
        return self.make_request(payload)

    def link_alias(self, alias_id, process_id):
        """Link an alias to a process.
        
        Args:
            alias_id: ID of the alias (obj_id from create_alias response)
            process_id: ID of the process to link to
            
        Returns:
            API response dict
        """
        payload = {
            "ops": [{
                "link": True,
                "obj_id": alias_id,
                "obj_to_id": process_id,
                "obj_to_type": "conv",
                "type": "link",
                "obj": "alias",
                "company_id": self.company_id
            }]
        }
        return self.make_request(payload)

    def create_and_link_alias(self, project_id, stage_id, process_id, short_name=None, title=None, description=""):
        """Create an alias and link it to a process in one call.
        
        If short_name is not provided, it will be generated from the process title.
        
        Args:
            project_id: Numeric project ID
            stage_id: Numeric stage ID
            process_id: ID of the process to link to
            short_name: Alias short name (auto-generated from process title if omitted)
            title: Display title (defaults to short_name)
            description: Optional description
            
        Returns:
            Dict with alias_id, short_name, and link result
        """
        if not short_name:
            details = self.get_process_details(process_id)
            if details.get("request_proc") == "ok" and details["ops"][0].get("proc") == "ok":
                process_title = details["ops"][0].get("title", "")
                short_name = self.generate_alias_name(process_title)
            else:
                raise ValueError(f"Cannot get process details for {process_id}")

        create_result = self.create_alias(project_id, stage_id, short_name, title or short_name, description)
        if create_result.get("request_proc") != "ok" or create_result["ops"][0].get("proc") != "ok":
            return {"error": "create failed", "detail": create_result}

        alias_id = create_result["ops"][0]["obj_id"]
        link_result = self.link_alias(alias_id, process_id)

        return {
            "alias_id": alias_id,
            "short_name": short_name,
            "link_result": link_result
        }

    @staticmethod
    def generate_alias_name(title):
        """Generate an alias short_name from a process title.
        
        Lowercase, replace spaces and underscores with hyphens,
        remove special characters.
        """
        import re
        name = title.lower().strip()
        name = re.sub(r'[_\s]+', '-', name)
        name = re.sub(r'[^a-z0-9\-]', '', name)
        name = re.sub(r'-+', '-', name)
        return name.strip('-')

    def crawl_dependencies(self, root_pid, project_id, stage_id):
        """BFS crawl all process dependencies starting from root_pid.

        Returns:
            dict mapping pid -> {title, alias, node_count, sub_deps, is_state_diagram}
        """
        all_aliases = self.list_aliases(project_id, stage_id)
        alias_to_pid = {}
        pid_to_alias = {}
        for a in all_aliases:
            sn = a.get('short_name', '')
            obj_to_id = a.get('obj_to_id')
            if sn and obj_to_id:
                alias_to_pid[sn] = obj_to_id
                pid_to_alias[obj_to_id] = sn

        visited = {root_pid}
        queue = [root_pid]
        all_deps = {}

        while queue:
            pid = queue.pop(0)
            try:
                details = self.get_process_details(pid)
                if details.get('request_proc') != 'ok' or details['ops'][0].get('proc') != 'ok':
                    all_deps[pid] = {'title': 'ERROR', 'alias': pid_to_alias.get(pid, ''),
                                     'node_count': 0, 'sub_deps': set(), 'is_state_diagram': False}
                    continue

                title = details['ops'][0].get('title', '')
                alias = pid_to_alias.get(pid, '')

                scheme = self.get_process_scheme(pid)
                if scheme.get('request_proc') != 'ok' or scheme['ops'][0].get('proc') != 'ok':
                    all_deps[pid] = {'title': title, 'alias': alias,
                                     'node_count': 0, 'sub_deps': set(), 'is_state_diagram': False}
                    continue

                scheme_list = scheme['ops'][0].get('scheme', [])
                if not scheme_list:
                    all_deps[pid] = {'title': title, 'alias': alias,
                                     'node_count': 0, 'sub_deps': set(), 'is_state_diagram': True}
                    continue

                proc = scheme_list[0]
                if 'scheme' not in proc or not isinstance(proc.get('scheme'), dict):
                    all_deps[pid] = {'title': title, 'alias': alias,
                                     'node_count': 0, 'sub_deps': set(), 'is_state_diagram': True}
                    continue

                nodes = proc['scheme'].get('nodes', [])
                sub_deps = set()
                for n in nodes:
                    for logic in n.get('condition', {}).get('logics', []):
                        cid = logic.get('conv_id', '')
                        if cid and logic.get('type') in ('api_rpc', 'api_copy'):
                            if isinstance(cid, int):
                                sub_deps.add(cid)
                            elif isinstance(cid, str):
                                if cid.isdigit():
                                    sub_deps.add(int(cid))
                                elif cid.startswith('@'):
                                    r = alias_to_pid.get(cid[1:])
                                    if r:
                                        sub_deps.add(r)
                                elif not cid.startswith('{{'):
                                    r = alias_to_pid.get(cid)
                                    if r:
                                        sub_deps.add(r)

                full_json = json.dumps(nodes, separators=(',', ':'), ensure_ascii=False)
                for sr in re.findall(r'conv\[@([^\]]+)\]', full_json):
                    r = alias_to_pid.get(sr)
                    if r:
                        sub_deps.add(r)

                all_deps[pid] = {'title': title, 'alias': alias,
                                 'node_count': len(nodes), 'sub_deps': sub_deps,
                                 'is_state_diagram': False}

                for sp in sub_deps:
                    if sp not in visited:
                        visited.add(sp)
                        queue.append(sp)
            except Exception:
                all_deps[pid] = {'title': 'EXCEPTION', 'alias': pid_to_alias.get(pid, ''),
                                 'node_count': 0, 'sub_deps': set(), 'is_state_diagram': False}

        return all_deps

    def generate_dependency_graph(self, root_pid, project_id, stage_id, output_path=None):
        """Generate a dependency graph JSON file for a process.

        The JSON can be loaded into dependency_graph.html (drag & drop or file picker).

        Args:
            root_pid: Root process ID to start crawling from
            project_id: Numeric project ID
            stage_id: Numeric stage ID
            output_path: Output JSON file path (default: dependency_graph_{root_pid}.json)

        Returns:
            dict with output_path, process_count, edge_count, total_process_nodes
        """
        if output_path is None:
            output_path = f"dependency_graph_{root_pid}.json"

        all_deps = self.crawl_dependencies(root_pid, project_id, stage_id)

        root_title = all_deps.get(root_pid, {}).get('title', str(root_pid))
        graph_nodes = []
        graph_edges = []
        for pid, info in all_deps.items():
            label = f"@{info['alias']}" if info['alias'] else str(pid)
            graph_nodes.append({
                'id': pid, 'label': label, 'title': info['title'],
                'nodeCount': info['node_count'], 'hasAlias': bool(info['alias'])
            })
            for dep in info['sub_deps']:
                if dep in all_deps:
                    graph_edges.append({'from': pid, 'to': dep})

        graph_data = {
            'rootPid': root_pid,
            'rootTitle': root_title,
            'nodes': graph_nodes,
            'edges': graph_edges
        }

        with open(output_path, 'w') as f:
            json.dump(graph_data, f, separators=(',', ':'))

        total_nodes = sum(i['node_count'] for i in all_deps.values())
        return {
            'output_path': output_path,
            'process_count': len(graph_nodes),
            'edge_count': len(graph_edges),
            'total_process_nodes': total_nodes
        }

    def get_process_scheme(self, process_id):
        """Get the full scheme/structure of a process"""
        payload = {
            "ops": [{
                "type": "get",
                "obj": "obj_scheme",
                "obj_id": process_id,
                "obj_type": "conv",
                "company_id": self.company_id
            }]
        }
        return self.make_request(payload)
    
    def _get_version(self, process_id):
        """Get the current version (change_time) of a process."""
        info = self.get_process_details(process_id)
        if info.get("request_proc") != "ok":
            return None
        return info["ops"][0].get("change_time", 0)

    def _parse_extra(self, extra):
        """Parse the extra field from a node (may be JSON string or dict)."""
        if isinstance(extra, str):
            try:
                return json.loads(extra)
            except Exception:
                return {"modeForm": "expand", "icon": ""}
        return extra if extra else {"modeForm": "expand", "icon": ""}

    def _find_node(self, process_id, node_id):
        """Find a node by ID in a process scheme. Returns (node_info, scheme) or (None, None)."""
        result = self.get_process_scheme(process_id)
        if result.get("request_proc") != "ok" or "scheme" not in result["ops"][0]:
            return None, None
        scheme = result["ops"][0]["scheme"][0]
        for node in scheme["scheme"]["nodes"]:
            if node["id"] == node_id:
                return node, scheme
        return None, scheme

    def modify_node(self, process_id, node_id, title=None, description=None,
                    position=None, logics=None, version=None):
        """
        Generic node modification — updates any combination of title, description,
        position, or logics while preserving all other properties.

        Args:
            process_id: Process ID
            node_id: Node ID to modify
            title: New title (None = keep existing)
            description: New description (None = keep existing)
            position: New [x, y] (None = keep existing)
            logics: New logics list (None = keep existing)
            version: Process version (None = auto-fetch)

        Returns:
            dict with 'success', 'message', and optionally 'version'
        """
        if version is None:
            version = self._get_version(process_id)
            if version is None:
                return {"success": False, "message": "Failed to get process version"}

        node_info, _ = self._find_node(process_id, node_id)
        if not node_info:
            return {"success": False, "message": f"Node {node_id} not found"}

        payload = {
            "ops": [{
                "type": "modify",
                "obj": "node",
                "obj_id": node_id,
                "company_id": self.company_id,
                "conv_id": process_id,
                "title": title if title is not None else node_info.get("title", ""),
                "description": description if description is not None else node_info.get("description", ""),
                "obj_type": node_info["obj_type"],
                "options": None,
                "logics": logics if logics is not None else node_info["condition"]["logics"],
                "semaphors": node_info["condition"].get("semaphors", []),
                "position": position if position is not None else [node_info.get("x", 0), node_info.get("y", 0)],
                "extra": self._parse_extra(node_info.get("extra", "{}")),
                "version": version
            }]
        }

        result = self.make_request(payload)
        ok = result.get("request_proc") == "ok" and result["ops"][0].get("proc") == "ok"
        if not ok:
            return {"success": False, "message": "Failed to modify node", "result": result}

        new_version = self._get_version(process_id)
        return {"success": True, "message": "Node modified", "version": new_version}

    def commit(self, process_id, version=None):
        """
        Commit pending changes to a process.

        Args:
            process_id: Process ID
            version: Process version (None = auto-fetch)

        Returns:
            dict with 'success' and 'message'
        """
        if version is None:
            version = self._get_version(process_id)
            if version is None:
                return {"success": False, "message": "Failed to get process version"}

        payload = {
            "ops": [{
                "type": "confirm",
                "obj": "commit",
                "conv_id": process_id,
                "company_id": self.company_id,
                "version": version
            }]
        }
        result = self.make_request(payload)
        ok = result.get("request_proc") == "ok" and result["ops"][0].get("proc") == "ok"
        return {"success": ok, "message": "Committed" if ok else "Commit failed", "result": result}

    def batch_modify_nodes(self, process_id, modifications, commit=True):
        """
        Apply multiple node modifications sequentially, then optionally commit.

        Args:
            process_id: Process ID
            modifications: list of dicts, each with 'node_id' and any of:
                           'title', 'description', 'position', 'logics'
            commit: Whether to commit after all modifications (default: True)

        Returns:
            dict with 'success', 'total', 'succeeded', 'failed', and 'results'
        """
        version = self._get_version(process_id)
        if version is None:
            return {"success": False, "message": "Failed to get process version"}

        results = []
        for mod in modifications:
            node_id = mod["node_id"]
            res = self.modify_node(
                process_id, node_id,
                title=mod.get("title"),
                description=mod.get("description"),
                position=mod.get("position"),
                logics=mod.get("logics"),
                version=version,
            )
            results.append({"node_id": node_id, **res})
            if res["success"]:
                version = res.get("version", version)

        succeeded = sum(1 for r in results if r["success"])
        failed = len(results) - succeeded

        commit_result = None
        if commit and succeeded > 0:
            commit_result = self.commit(process_id, version)

        return {
            "success": failed == 0 and (not commit or (commit_result and commit_result["success"])),
            "total": len(results),
            "succeeded": succeeded,
            "failed": failed,
            "results": results,
            "commit": commit_result,
        }

    def modify_node_code(self, process_id, node_id, new_code, title=None, description=None, commit=True):
        """
        Modify the JavaScript code in a Corezoid node.
        
        Args:
            process_id: Process ID
            node_id: Node ID to modify
            new_code: New JavaScript code to set
            title: Node title (optional, will fetch if not provided)
            description: Node description (optional, will fetch if not provided)
            commit: Whether to commit changes after modification (default: True)
            
        Returns:
            dict: Result with 'success' boolean and 'message' string
        """
        # Get current process version
        process_info = self.get_process_details(process_id)
        if process_info.get("request_proc") != "ok":
            return {"success": False, "message": f"Failed to get process info"}
        
        version = process_info["ops"][0].get("change_time", 0)
        
        # Get node information if title/description not provided
        if title is None or description is None:
            result = self.get_process_scheme(process_id)
            if result.get("request_proc") != "ok" or "scheme" not in result["ops"][0]:
                return {"success": False, "message": "Failed to get process scheme"}
            
            scheme = result["ops"][0]["scheme"][0]
            node_info = None
            
            for node in scheme["scheme"]["nodes"]:
                if node["id"] == node_id:
                    node_info = node
                    break
            
            if not node_info:
                return {"success": False, "message": f"Node {node_id} not found"}
            
            if title is None:
                title = node_info.get("title", "")
            if description is None:
                description = node_info.get("description", "")
            
            # Get existing logics
            err_node_id = None
            to_node_id = None
            obj_type = node_info.get("obj_type", 0)
            position = [node_info.get("x", 0), node_info.get("y", 0)]
            extra = node_info.get("extra", "{}")
            
            if isinstance(extra, str):
                try:
                    extra = json.loads(extra)
                except:
                    extra = {"modeForm": "expand", "icon": ""}
            
            for logic in node_info["condition"]["logics"]:
                if logic.get("type") == "api_code":
                    err_node_id = logic.get("err_node_id")
                elif logic.get("type") == "go":
                    to_node_id = logic.get("to_node_id")
        else:
            obj_type = 3
            position = [852, 916]
            extra = {"modeForm": "expand", "icon": ""}
            err_node_id = None
            to_node_id = None
        
        # Build logics
        logics = [{"type": "api_code", "lang": "js", "src": new_code}]
        if err_node_id:
            logics[0]["err_node_id"] = err_node_id
        if to_node_id:
            logics.append({"type": "go", "to_node_id": to_node_id})
        
        # Modify node
        modify_payload = {
            "ops": [{
                "type": "modify",
                "obj": "node",
                "obj_id": node_id,
                "company_id": self.company_id,
                "conv_id": process_id,
                "title": title,
                "description": description,
                "obj_type": obj_type,
                "options": None,
                "logics": logics,
                "semaphors": [],
                "position": position,
                "extra": extra,
                "version": version
            }]
        }
        
        result = self.make_request(modify_payload)
        
        if result.get("request_proc") != "ok" or result["ops"][0].get("proc") != "ok":
            return {"success": False, "message": "Failed to modify node", "result": result}
        
        # Commit if requested
        if commit:
            commit_payload = {
                "ops": [{
                    "type": "confirm",
                    "obj": "commit",
                    "conv_id": process_id,
                    "company_id": self.company_id,
                    "version": version
                }]
            }
            
            commit_result = self.make_request(commit_payload)
            
            if commit_result.get("request_proc") != "ok" or commit_result["ops"][0].get("proc") != "ok":
                return {"success": False, "message": "Modified but commit failed", "commit_result": commit_result}
        
        return {
            "success": True,
            "message": "Node code modified and committed" if commit else "Node code modified",
            "process_id": process_id,
            "node_id": node_id,
            "code_length": len(new_code)
        }


    # ------------------------------------------------------------------
    # Internal helpers for analysis tools
    # ------------------------------------------------------------------

    def _get_nodes(self, process_id):
        """Fetch and return (nodes_list, process_title) or raise on failure."""
        scheme = self.get_process_scheme(process_id)
        if scheme.get("request_proc") != "ok" or scheme["ops"][0].get("proc") != "ok":
            raise RuntimeError(f"Failed to fetch scheme for process {process_id}")
        proc = scheme["ops"][0]["scheme"][0]
        return proc["scheme"]["nodes"], proc.get("title", "")

    # ------------------------------------------------------------------
    # Analysis tools
    # ------------------------------------------------------------------

    def analyze_process_structure(self, process_id):
        """Return structural summary: node counts by type, logic distribution,
        untitled nodes, and basic stats.

        Returns:
            dict with keys: process_id, title, total_nodes, node_types,
            logic_types, untitled_count, untitled_nodes (list of {id, obj_type, logics, semaphors}).
        """
        nodes, title = self._get_nodes(process_id)
        type_names = {0: "standard", 1: "start", 2: "final", 3: "escalation"}

        node_types = {}
        for n in nodes:
            key = type_names.get(n["obj_type"], f"unknown_{n['obj_type']}")
            node_types[key] = node_types.get(key, 0) + 1

        logic_types = {}
        for n in nodes:
            for lg in n["condition"]["logics"]:
                lt = lg["type"]
                logic_types[lt] = logic_types.get(lt, 0) + 1

        untitled = []
        for n in nodes:
            if not n.get("title", "").strip():
                untitled.append({
                    "id": n["id"],
                    "obj_type": n["obj_type"],
                    "logics": [lg["type"] for lg in n["condition"]["logics"]],
                    "semaphors": [s["type"] for s in n["condition"]["semaphors"]],
                })

        return {
            "process_id": process_id,
            "title": title,
            "total_nodes": len(nodes),
            "node_types": node_types,
            "logic_types": logic_types,
            "untitled_count": len(untitled),
            "untitled_nodes": untitled,
        }

    def find_nodes_missing_semaphors(self, process_id):
        """Find nodes that should have a timeout semaphor but don't.

        Groups results by severity:
        - critical: api_callback without semaphor (tasks hang forever)
        - high: api (outbound HTTP) without semaphor
        - low: api_rpc / api_copy is_sync without semaphor

        Returns:
            dict with keys: process_id, title, critical, high, low.
            Each severity is a list of {id, title, logic_type, description}.
        """
        nodes, title = self._get_nodes(process_id)
        critical, high, low = [], [], []

        for n in nodes:
            has_semaphor = len(n["condition"]["semaphors"]) > 0
            if has_semaphor:
                continue

            logic_set = set()
            for lg in n["condition"]["logics"]:
                logic_set.add(lg["type"])

            entry = {
                "id": n["id"],
                "title": n.get("title", ""),
                "description": n.get("description", ""),
            }

            if "api_callback" in logic_set:
                critical.append({**entry, "logic_type": "api_callback"})
            elif "api" in logic_set:
                high.append({**entry, "logic_type": "api"})
            elif "api_rpc" in logic_set:
                # api_rpc target handles own timeouts — low severity
                low.append({**entry, "logic_type": "api_rpc"})
            elif "api_copy" in logic_set:
                is_sync = any(
                    lg.get("is_sync") for lg in n["condition"]["logics"]
                    if lg["type"] == "api_copy"
                )
                if is_sync:
                    low.append({**entry, "logic_type": "api_copy (sync)"})

        return {
            "process_id": process_id,
            "title": title,
            "critical": critical,
            "high": high,
            "low": low,
        }

    def extract_code_nodes(self, process_id):
        """Extract all JS/Erlang code nodes with source and quality flags.

        Returns:
            dict with keys: process_id, title, code_nodes (list).
            Each code_node: {id, title, lang, src, length, has_try_catch,
            has_commented_code, hardcoded_urls}.
        """
        nodes, title = self._get_nodes(process_id)
        results = []

        for n in nodes:
            for lg in n["condition"]["logics"]:
                if lg["type"] not in ("api_code", "code"):
                    continue
                src = lg.get("src", "")
                has_try = "try" in src and "catch" in src
                has_commented = bool(re.search(r'^\s*//', src, re.MULTILINE))
                hardcoded_urls = re.findall(r'https?://[^\s\'"]+', src)
                results.append({
                    "id": n["id"],
                    "title": n.get("title", ""),
                    "lang": lg.get("lang", "unknown"),
                    "src": src,
                    "length": len(src),
                    "has_try_catch": has_try,
                    "has_commented_code": has_commented,
                    "hardcoded_urls": hardcoded_urls,
                })

        return {"process_id": process_id, "title": title, "code_nodes": results}

    def list_external_dependencies(self, process_id):
        """List all outbound process references: api_rpc, api_copy, and state store reads.

        Returns:
            dict with keys: process_id, title, dependencies (list), state_store_refs (list).
            Each dependency: {conv_id, call_type, mode, count, node_titles}.
            Each state_store_ref: the raw conv[@alias].ref[...] expression.
        """
        nodes, title = self._get_nodes(process_id)

        deps = {}  # (conv_id, call_type, mode) -> {count, node_titles}
        state_refs = set()

        for n in nodes:
            for lg in n["condition"]["logics"]:
                if lg["type"] in ("api_rpc", "api_copy"):
                    cid = str(lg.get("conv_id", ""))
                    mode = str(lg.get("mode", ""))
                    key = (cid, lg["type"], mode)
                    if key not in deps:
                        deps[key] = {"count": 0, "node_titles": []}
                    deps[key]["count"] += 1
                    deps[key]["node_titles"].append(n.get("title", "") or n["id"])

                if lg["type"] == "set_param":
                    for _k, v in lg.get("extra", {}).items():
                        if isinstance(v, str) and "conv[" in v:
                            state_refs.add(v)

        dep_list = []
        for (cid, ctype, mode), info in sorted(deps.items()):
            dep_list.append({
                "conv_id": cid,
                "call_type": ctype,
                "mode": mode or None,
                "count": info["count"],
                "node_titles": info["node_titles"],
                "is_numeric": cid.isdigit(),
            })

        return {
            "process_id": process_id,
            "title": title,
            "total_outbound_calls": sum(d["count"] for d in dep_list),
            "unique_dependencies": len(dep_list),
            "dependencies": dep_list,
            "state_store_refs": sorted(state_refs),
        }

    def find_hardcoded_values(self, process_id):
        """Scan for hardcoded URLs, numeric conv_ids, and suspicious literals.

        Returns:
            dict with keys: process_id, title, hardcoded_urls, numeric_conv_ids,
            hardcoded_api_extras.
        """
        nodes, title = self._get_nodes(process_id)

        hardcoded_urls = []
        numeric_conv_ids = []
        hardcoded_extras = []

        for n in nodes:
            for lg in n["condition"]["logics"]:
                # Direct API calls with hardcoded URLs
                if lg["type"] == "api" and "url" in lg:
                    url = lg["url"]
                    if not url.startswith("{{"):
                        hardcoded_urls.append({
                            "id": n["id"],
                            "title": n.get("title", ""),
                            "url": url,
                        })

                # Numeric conv_ids (should be aliases)
                if lg["type"] in ("api_rpc", "api_copy"):
                    cid = lg.get("conv_id", "")
                    is_numeric = (isinstance(cid, int) or
                                  (isinstance(cid, str) and cid.isdigit()))
                    if is_numeric:
                        numeric_conv_ids.append({
                            "id": n["id"],
                            "title": n.get("title", ""),
                            "conv_id": cid,
                            "call_type": lg["type"],
                        })

                # Hardcoded values in api/api_rpc extra fields
                if lg["type"] in ("api", "api_rpc", "api_copy"):
                    for k, v in lg.get("extra", {}).items():
                        if isinstance(v, str) and not v.startswith("{{") and v:
                            # Skip known safe literal values
                            if v.lower() in ("true", "false", "string", "object",
                                             "number", "boolean"):
                                continue
                            hardcoded_extras.append({
                                "id": n["id"],
                                "title": n.get("title", ""),
                                "field": k,
                                "value": v,
                                "call_type": lg["type"],
                            })

        return {
            "process_id": process_id,
            "title": title,
            "hardcoded_urls": hardcoded_urls,
            "numeric_conv_ids": numeric_conv_ids,
            "hardcoded_api_extras": hardcoded_extras,
        }

    # todo improve logic
    def find_duplicate_patterns(self, process_id):
        """Detect duplicate set_param patterns and duplicate @send-message signatures.

        Returns:
            dict with keys: process_id, title, duplicate_set_params,
            duplicate_send_messages.
        """
        nodes, title = self._get_nodes(process_id)

        # Duplicate set_param extras
        sp_patterns = {}  # json_pattern -> [node_titles]
        for n in nodes:
            for lg in n["condition"]["logics"]:
                if lg["type"] == "set_param":
                    pattern = json.dumps(lg.get("extra", {}),sort_keys=True,separators=(',', ':'),ensure_ascii=False)
                    sp_patterns.setdefault(pattern, []).append(
                        n.get("title", "") or n["id"]
                    )

        dup_set_params = []
        for pattern, names in sp_patterns.items():
            if len(names) > 1:
                dup_set_params.append({
                    "pattern": json.loads(pattern),
                    "count": len(names),
                    "nodes": names,
                })

        # Duplicate @send-message calls (same text_id + attachment_id)
        sm_patterns = {}  # (text_id, attachment_id) -> [node_titles]
        for n in nodes:
            for lg in n["condition"]["logics"]:
                if lg["type"] == "api_rpc" and lg.get("conv_id") == "@send-message":
                    text_id = lg.get("extra", {}).get("text_id", "")
                    attach_id = lg.get("extra", {}).get("attachment_id", "")
                    key = (text_id, attach_id)
                    sm_patterns.setdefault(key, []).append(
                        n.get("title", "") or n["id"]
                    )

        dup_send_msgs = []
        for (text_id, attach_id), names in sm_patterns.items():
            if len(names) > 1:
                dup_send_msgs.append({
                    "text_id": text_id,
                    "attachment_id": attach_id or None,
                    "count": len(names),
                    "nodes": names,
                })

        return {
            "process_id": process_id,
            "title": title,
            "duplicate_set_params": dup_set_params,
            "duplicate_send_messages": dup_send_msgs,
        }

    def find_untitled_nodes(self, process_id):
        """Return untitled nodes grouped by obj_type with flow context.

        For each untitled node, includes what feeds into it and what it feeds.

        Returns:
            dict with keys: process_id, title, total_nodes, untitled_count,
            by_type (dict of obj_type -> list of node info).
        """
        nodes, title = self._get_nodes(process_id)
        node_map = {n["id"]: n for n in nodes}
        type_names = {0: "standard", 1: "start", 2: "final", 3: "escalation"}

        # Build reverse index: node_id -> list of source node titles
        sources_map = {}  # target_id -> [source descriptions]
        for n in nodes:
            for lg in n["condition"]["logics"]:
                tid = lg.get("to_node_id")
                if tid:
                    sources_map.setdefault(tid, []).append(
                        n.get("title", "") or n["id"]
                    )
                eid = lg.get("err_node_id")
                if eid:
                    sources_map.setdefault(eid, []).append(
                        f"(err) {n.get('title', '') or n['id']}"
                    )
            for sem in n["condition"]["semaphors"]:
                tid = sem.get("to_node_id")
                if tid:
                    sources_map.setdefault(tid, []).append(
                        f"(timeout) {n.get('title', '') or n['id']}"
                    )

        by_type = {}
        for n in nodes:
            if n.get("title", "").strip():
                continue
            type_key = type_names.get(n["obj_type"], f"unknown_{n['obj_type']}")

            # What this node goes to
            targets = []
            for lg in n["condition"]["logics"]:
                tid = lg.get("to_node_id")
                if tid and tid in node_map:
                    targets.append(node_map[tid].get("title", "") or tid)

            entry = {
                "id": n["id"],
                "logics": [lg["type"] for lg in n["condition"]["logics"]],
                "semaphors": [
                    f"{s.get('value', '?')}{s.get('dimension', 's')}"
                    for s in n["condition"]["semaphors"]
                ],
                "sources": sources_map.get(n["id"], []),
                "targets": targets,
            }
            by_type.setdefault(type_key, []).append(entry)

        return {
            "process_id": process_id,
            "title": title,
            "total_nodes": len(nodes),
            "untitled_count": sum(len(v) for v in by_type.values()),
            "by_type": by_type,
        }

    def validate_escalation_chains(self, process_id):
        """Verify every escalation node (obj_type 3) routes correctly.

        Checks:
        - crash_api / rpc_task_fatal_error → retry delay (node with semaphor)
        - access_denied / rpc_reply_exception → final error (obj_type 2)
        - default fallthrough → final error (obj_type 2)

        Returns:
            dict with keys: process_id, title, total_escalation_nodes,
            issues (list of {id, title, issue}).
        """
        nodes, title = self._get_nodes(process_id)
        node_map = {n["id"]: n for n in nodes}

        RETRY_ERRORS = {"crash_api", "rpc_task_fatal_error", "copy_task_timeout",
                        "copy_task_fatal_error", "hardware"}
        FINAL_ERRORS = {"access_denied", "rpc_reply_exception",
                        "not_unical_ref", "software"}

        escalation_nodes = [n for n in nodes if n["obj_type"] == 3]
        issues = []

        for n in escalation_nodes:
            logics = n["condition"]["logics"]
            if not logics:
                issues.append({
                    "id": n["id"],
                    "title": n.get("title", ""),
                    "issue": "Escalation node has no logics — dead end",
                })
                continue

            has_default = False
            for lg in logics:
                if lg["type"] == "go":
                    has_default = True
                    target = node_map.get(lg.get("to_node_id"))
                    if target and target["obj_type"] != 2:
                        issues.append({
                            "id": n["id"],
                            "title": n.get("title", ""),
                            "issue": f"Default fallthrough goes to non-final node "
                                     f"'{target.get('title', '')}' ({target['id']}, "
                                     f"obj_type={target['obj_type']})",
                        })

                elif lg["type"] == "go_if_const":
                    conditions = lg.get("conditions", [])
                    target_id = lg.get("to_node_id")
                    target = node_map.get(target_id)
                    if not target:
                        issues.append({
                            "id": n["id"],
                            "title": n.get("title", ""),
                            "issue": f"go_if_const points to missing node {target_id}",
                        })
                        continue

                    # Determine if this is a retry or final route
                    matched_consts = {c.get("const", "") for c in conditions}
                    is_retry_route = bool(matched_consts & RETRY_ERRORS)
                    is_final_route = bool(matched_consts & FINAL_ERRORS)

                    if is_retry_route:
                        # Retry target should be a delay node (has semaphor)
                        has_sem = len(target["condition"]["semaphors"]) > 0
                        if not has_sem and target["obj_type"] != 2:
                            issues.append({
                                "id": n["id"],
                                "title": n.get("title", ""),
                                "issue": f"Retry route for {matched_consts & RETRY_ERRORS} "
                                         f"goes to node without semaphor "
                                         f"'{target.get('title', '')}' ({target['id']})",
                            })
                    elif is_final_route:
                        if target["obj_type"] != 2:
                            issues.append({
                                "id": n["id"],
                                "title": n.get("title", ""),
                                "issue": f"Final error route for {matched_consts & FINAL_ERRORS} "
                                         f"goes to non-final node "
                                         f"'{target.get('title', '')}' ({target['id']}, "
                                         f"obj_type={target['obj_type']})",
                            })

                # Also check api_copy inside escalation (e.g. @login-session fallback)
                elif lg["type"] == "api_copy":
                    pass  # Valid pattern — escalation creates a session as fallback

            if not has_default:
                issues.append({
                    "id": n["id"],
                    "title": n.get("title", ""),
                    "issue": "Escalation node has no default 'go' fallthrough",
                })

        return {
            "process_id": process_id,
            "title": title,
            "total_escalation_nodes": len(escalation_nodes),
            "issues": issues,
        }


    def find_orphaned_nodes(self, process_id):
        """BFS from Start node to find unreachable (orphaned) nodes.

        Builds a directed graph from all outbound edges (go, go_if_const,
        err_node_id, semaphor to_node_id) and walks from the Start node.
        Any node not visited is orphaned dead code.

        Returns:
            dict with keys: process_id, title, total_nodes,
            reachable_count, orphaned_count, orphaned_nodes list.
        """
        from collections import deque

        nodes, title = self._get_nodes(process_id)
        node_map = {n["id"]: n for n in nodes}
        all_ids = set(node_map.keys())
        type_labels = {0: "standard", 1: "start", 2: "final", 3: "escalation"}

        # Build adjacency list
        adj = {nid: set() for nid in all_ids}
        for n in nodes:
            nid = n["id"]
            cond = n.get("condition", {})
            for logic in cond.get("logics", []):
                if "to_node_id" in logic:
                    adj[nid].add(logic["to_node_id"])
                if "err_node_id" in logic:
                    adj[nid].add(logic["err_node_id"])
            for sem in cond.get("semaphors", []):
                if "to_node_id" in sem:
                    adj[nid].add(sem["to_node_id"])

        # Find start node
        start_id = None
        for n in nodes:
            if n["obj_type"] == 1:
                start_id = n["id"]
                break

        if not start_id:
            return {
                "process_id": process_id,
                "title": title,
                "error": "No start node (obj_type=1) found",
            }

        # BFS
        visited = set()
        queue = deque([start_id])
        visited.add(start_id)
        while queue:
            cur = queue.popleft()
            for nb in adj.get(cur, set()):
                if nb not in visited and nb in all_ids:
                    visited.add(nb)
                    queue.append(nb)

        orphaned = all_ids - visited
        orphaned_list = []
        for oid in sorted(orphaned):
            n = node_map[oid]
            orphaned_list.append({
                "id": oid,
                "title": n.get("title", "") or "(untitled)",
                "obj_type": type_labels.get(n["obj_type"], str(n["obj_type"])),
            })

        return {
            "process_id": process_id,
            "title": title,
            "total_nodes": len(all_ids),
            "reachable_count": len(visited),
            "orphaned_count": len(orphaned),
            "orphaned_nodes": orphaned_list,
        }

    def check_variable_usage(self, process_id, variable_names):
        """Check if variables are referenced anywhere in a process.

        Scans all node logics, extras, semaphors, and conditions for
        {{variable_name}} references. Returns per-variable results with
        the list of nodes where each variable is used and how.

        Args:
            process_id: Process to scan.
            variable_names: Single name (str) or list of variable names
                            without {{ }} brackets.

        Returns:
            dict with process_id, title, and variables dict mapping each
            name to {referenced, reference_count, references}.
        """
        if isinstance(variable_names, str):
            variable_names = [variable_names]

        nodes, title = self._get_nodes(process_id)
        patterns = {v: "{{" + v + "}}" for v in variable_names}

        # Per-variable results
        results = {v: [] for v in variable_names}

        for n in nodes:
            for var, pattern in patterns.items():
                node_refs = []
                for lg in n["condition"]["logics"]:
                    lg_str = str(lg)
                    if pattern in lg_str:
                        lg_type = lg.get("type", "unknown")
                        if lg_type == "set_param":
                            extra = lg.get("extra", {})
                            extra_vals = str(extra.values())
                            if pattern in extra_vals:
                                node_refs.append(
                                    f"set_param reads {pattern} in value")
                            if var in extra:
                                node_refs.append(
                                    f"set_param writes to '{var}'")
                        elif lg_type == "go_if_const":
                            for cond in lg.get("conditions", []):
                                if pattern in str(cond):
                                    node_refs.append(
                                        f"go_if_const condition: "
                                        f"{cond.get('param','')} "
                                        f"{cond.get('fun','')} "
                                        f"'{cond.get('const','')}'")
                        elif lg_type in ("api_rpc", "api_copy", "api"):
                            extra = lg.get("extra", {})
                            for k, v in extra.items():
                                if pattern in str(v):
                                    node_refs.append(
                                        f"{lg_type} extra.{k} = {v}")
                        elif lg_type == "code":
                            node_refs.append(
                                f"code node references {pattern}")
                        else:
                            node_refs.append(
                                f"{lg_type} logic references {pattern}")

                for sem in n["condition"]["semaphors"]:
                    if pattern in str(sem):
                        node_refs.append(
                            f"semaphor references {pattern}")

                if node_refs:
                    results[var].append({
                        "id": n["id"],
                        "title": n.get("title", "") or "(untitled)",
                        "usages": node_refs,
                    })

        variables = {}
        for var in variable_names:
            refs = results[var]
            variables[var] = {
                "referenced": len(refs) > 0,
                "reference_count": len(refs),
                "references": refs,
            }

        return {
            "process_id": process_id,
            "title": title,
            "variables": variables,
        }

    def find_noop_nodes(self, process_id):
        """Detect functionally useless nodes in a process.

        Finds two patterns:
        1. No-op conditions: nodes where ALL branches (go_if_const + go)
           route to the same destination. The condition evaluates but the
           outcome is irrelevant.
        2. Unused set_param: nodes that set variables via set_param, but
           the variable is never referenced ({{var}}) in any other node's
           logics or extras — or only referenced inside a no-op condition.

        Returns:
            dict with process_id, title, noop_conditions list,
            unused_set_params list.
        """
        nodes, title = self._get_nodes(process_id)
        node_map = {n["id"]: n for n in nodes}

        # --- Pattern 1: No-op conditions ---
        noop_conditions = []
        noop_node_ids = set()
        for n in nodes:
            logics = n["condition"]["logics"]
            if not logics:
                continue
            # Collect all to_node_id targets from routing logics
            targets = set()
            has_routing = False
            for lg in logics:
                if lg["type"] in ("go", "go_if_const"):
                    has_routing = True
                    tid = lg.get("to_node_id")
                    if tid:
                        targets.add(tid)
            # A no-op is when there are 2+ routing logics but all go
            # to the same single destination
            if has_routing and len(targets) == 1:
                routing_count = sum(
                    1 for lg in logics
                    if lg["type"] in ("go", "go_if_const")
                )
                if routing_count >= 2:
                    dest = list(targets)[0]
                    dest_title = node_map.get(dest, {}).get("title", "") or "(untitled)"
                    noop_conditions.append({
                        "id": n["id"],
                        "title": n.get("title", "") or "(untitled)",
                        "routing_count": routing_count,
                        "single_destination": dest,
                        "destination_title": dest_title,
                        "issue": f"All {routing_count} branches route to "
                                 f"the same node '{dest_title}' ({dest})",
                    })
                    noop_node_ids.add(n["id"])

        # --- Pattern 2: Unused set_param ---
        # Collect all variables set by set_param logics
        set_param_nodes = []
        for n in nodes:
            for lg in n["condition"]["logics"]:
                if lg["type"] == "set_param":
                    extra = lg.get("extra", {})
                    if extra:
                        set_param_nodes.append({
                            "node": n,
                            "variables": list(extra.keys()),
                        })

        # Build a text blob of all node logics/extras for reference
        # scanning, excluding no-op condition nodes
        ref_text_parts = []
        for n in nodes:
            if n["id"] in noop_node_ids:
                continue
            for lg in n["condition"]["logics"]:
                if lg["type"] == "set_param":
                    continue
                ref_text_parts.append(str(lg))
            for sem in n["condition"]["semaphors"]:
                ref_text_parts.append(str(sem))
        local_ref_blob = " ".join(ref_text_parts)

        # Collect group:"all" forwarding nodes — these pass the entire
        # task data to a target process, so we need to check there too.
        group_all_targets = []  # list of (conv_id_raw, node_info)
        group_all_nodes = []
        for n in nodes:
            if n["id"] in noop_node_ids:
                continue
            for lg in n["condition"]["logics"]:
                if lg["type"] in ("api_rpc", "api_copy") and \
                        lg.get("group") == "all":
                    conv_id_raw = lg.get("conv_id", "")
                    group_all_targets.append(conv_id_raw)
                    group_all_nodes.append({
                        "id": n["id"],
                        "title": n.get("title", "") or "(untitled)",
                        "call_type": lg["type"],
                        "conv_id": conv_id_raw,
                    })

        # For variables not found locally, scan target processes
        # reachable via group:"all". Build a combined ref blob from
        # all target process nodes (fetched lazily, cached).
        remote_ref_blob = None  # built on demand
        remote_resolved = []    # track what was actually resolved

        def _build_remote_ref_blob():
            """Fetch schemes of group:all target processes and build
            a combined text blob of their logics for {{var}} scanning."""
            parts = []
            resolved_pids = set()
            for conv_raw in group_all_targets:
                pid = None
                if isinstance(conv_raw, int) or \
                        (isinstance(conv_raw, str) and conv_raw.isdigit()):
                    pid = int(conv_raw)
                elif isinstance(conv_raw, str) and conv_raw.startswith("@"):
                    # Resolve alias — need project_id/stage_id
                    try:
                        details = self.get_process_details(process_id)
                        proj = details["ops"][0]["project_id"]
                        stg = details["ops"][0]["stage_id"]
                        alias_name = conv_raw.lstrip("@")
                        resolved = self.resolve_aliases(proj, stg, [alias_name])
                        if alias_name in resolved:
                            pid = resolved[alias_name].get("obj_to_id")
                            remote_resolved.append({
                                "alias": conv_raw,
                                "resolved_process_id": pid,
                            })
                        else:
                            remote_resolved.append({
                                "alias": conv_raw,
                                "resolved_process_id": None,
                                "error": "alias not found in project stage",
                            })
                    except Exception as e:
                        remote_resolved.append({
                            "alias": conv_raw,
                            "resolved_process_id": None,
                            "error": str(e),
                        })
                        continue
                if pid and pid not in resolved_pids and pid != process_id:
                    resolved_pids.add(pid)
                    try:
                        target_nodes, _ = self._get_nodes(pid)
                        for tn in target_nodes:
                            for lg in tn["condition"]["logics"]:
                                parts.append(str(lg))
                            for sem in tn["condition"]["semaphors"]:
                                parts.append(str(sem))
                    except Exception as e:
                        remote_resolved.append({
                            "process_id": pid,
                            "error": f"failed to fetch scheme: {e}",
                        })
                        continue
            return " ".join(parts)

        unused_set_params = []
        for sp in set_param_nodes:
            n = sp["node"]
            unreferenced = []
            for var in sp["variables"]:
                pattern = "{{" + var + "}}"
                # Check local references first
                if pattern in local_ref_blob:
                    continue
                # Not found locally — check remote if group:all exists
                if group_all_targets:
                    if remote_ref_blob is None:
                        remote_ref_blob = _build_remote_ref_blob()
                    if pattern in remote_ref_blob:
                        continue
                unreferenced.append(var)
            if unreferenced:
                unused_set_params.append({
                    "id": n["id"],
                    "title": n.get("title", "") or "(untitled)",
                    "unused_variables": unreferenced,
                    "issue": f"set_param sets {unreferenced} but "
                             f"no downstream node references them "
                             f"(including group:all target processes)",
                })

        return {
            "process_id": process_id,
            "title": title,
            "noop_conditions": noop_conditions,
            "unused_set_params": unused_set_params,
            "group_all_forwards": group_all_nodes,
            "remote_scanned": remote_resolved,
        }

    # ------------------------------------------------------------------
    # Task Management
    # ------------------------------------------------------------------

    def create_task(self, process_id, data, ref=None):
        """Send a new task to a process.

        Args:
            process_id: Target process (conv_id).
            data: dict of task parameters.
            ref: Optional unique reference string for the task.

        Returns:
            API response dict.  On success ``ops[0]['obj_id']`` is the task_id.
        """
        op = {
            "type": "create",
            "obj": "task",
            "conv_id": process_id,
            "data": data,
        }
        if ref is not None:
            op["ref"] = ref
        return self.make_request({"ops": [op]})

    def show_task(self, process_id, task_id=None, ref=None):
        """Retrieve a task's current state by task_id and/or ref.

        At least one of *task_id* or *ref* must be provided.

        Returns:
            API response dict.  ``ops[0]['data']`` contains the full task
            data, ``ops[0]['node_id']`` the current node, and
            ``ops[0]['status']`` the task status.
        """
        if task_id is None and ref is None:
            raise ValueError("At least one of task_id or ref must be provided")
        op = {
            "type": "show",
            "obj": "task",
            "conv_id": process_id,
        }
        if task_id is not None:
            op["obj_id"] = task_id
        if ref is not None:
            op["ref"] = ref
        return self.make_request({"ops": [op]})

    def list_task_history(self, process_id, task_id):
        """Return the execution history (node path) for a task.

        Only *task_id* (obj_id) is supported — ``ref`` cannot be used here.

        Note:
            The ``data`` field in each history entry is always ``null``;
            use :meth:`show_task` to get the task's current data.

        Returns:
            API response dict.  ``ops[0]['list']`` is an ordered array of
            node transitions with ``node_id``, ``node_prev_id``,
            ``create_time_ms``, ``user_name``, etc.
        """
        return self.make_request({
            "ops": [{
                "type": "list",
                "obj": "task_history",
                "conv_id": process_id,
                "obj_id": task_id,
            }]
        })

    def list_node_tasks(self, process_id, node_id, limit=50, offset=0):
        """Return tasks currently sitting in a specific node of a process.

        Args:
            process_id: Numeric process (conv) ID.
            node_id: 24-character hex node ID.
            limit: Maximum number of tasks to return (default 50).
            offset: Zero-based pagination offset (default 0).

        Returns:
            API response dict.  ``ops[0]['list']`` is an array of task objects,
            each containing ``task_id``, ``ref``, and ``data``.
            ``ops[0]['count']`` is the total number of tasks in the node.
        """
        return self.make_request({
            "ops": [{
                "type": "list",
                "obj": "node",
                "company_id": self.company_id,
                "conv_id": process_id,
                "obj_id": node_id,
                "limit": limit,
                "offset": offset,
            }]
        })

    def modify_task(self, process_id, data, task_id=None, ref=None):
        """Modify an existing task's data to continue going through the process.

        At least one of *task_id* or *ref* must be provided.
        The *data* dict is merged into the task — only the keys you pass
        are updated; existing keys you omit are preserved.

        The task will continue the flow from the node where it was modified with the updated data.

        Returns:
            API response dict.
        """
        if task_id is None and ref is None:
            raise ValueError("At least one of task_id or ref must be provided")
        op = {
            "type": "modify",
            "obj": "task",
            "conv_id": process_id,
            "data": data,
        }
        if task_id is not None:
            op["obj_id"] = task_id
        if ref is not None:
            op["ref"] = ref
        return self.make_request({"ops": [op]})

    def delete_task(self, process_id, task_id=None, ref=None):
        """Delete a task from a process.

        At least one of *task_id* or *ref* must be provided.
        If only *ref* is given, the method calls :meth:`show_task` first
        to resolve the ``task_id`` and ``node_id`` required by the API.

        Returns:
            API response dict.
        """
        if task_id is None and ref is None:
            raise ValueError("At least one of task_id or ref must be provided")

        node_id = None

        if task_id is None:
            show_resp = self.show_task(process_id, ref=ref)
            op_result = show_resp.get("ops", [{}])[0]
            if op_result.get("proc") != "ok":
                return show_resp
            task_id = op_result["obj_id"]
            node_id = op_result["node_id"]
        else:
            show_resp = self.show_task(process_id, task_id=task_id)
            op_result = show_resp.get("ops", [{}])[0]
            if op_result.get("proc") != "ok":
                return show_resp
            node_id = op_result["node_id"]

        return self.make_request({
            "ops": [{
                "type": "delete",
                "obj": "task",
                "conv_id": process_id,
                "obj_id": task_id,
                "node_id": node_id,
            }]
        })

    def copy_process(self, process_id, title=None, folder_id=None):
        """Copy (clone) a process.

        Uses the ``/api/2/copy`` endpoint.

        Args:
            process_id: Source process to copy.
            title: Title for the new copy. Defaults to
                   ``"<original_title> (Copy)"``.
            folder_id: Destination folder ID. Defaults to the same folder
                       as the source process.

        Returns:
            API response dict.  On success ``ops[0]['obj_id']`` is the new
            process ID.
        """
        details = self.get_process_details(process_id)
        info = details["ops"][0]
        company_id = info["company_id"]

        if title is None:
            title = f"{info['title']} (Copy)"
        if folder_id is None:
            folder_id = info.get("parent_obj_id", 0)

        return self.make_request({
            "ops": [{
                "type": "create",
                "obj": "obj_copy",
                "obj_type": "conv",
                "obj_id": process_id,
                "obj_to_id": folder_id,
                "obj_to_type": "folder",
                "title": title,
                "company_id": company_id,
                "from_company_id": company_id,
                "to_company_id": company_id,
                "async": True,
                "ignore_errors": True,
            }]
        }, endpoint="copy")


if __name__ == "__main__":
    client = CorezoidClient(api_login="", secret="", base_url="", company_id="")
    print("CorezoidClient ready.")
