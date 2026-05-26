package main

import (
	"context"
	"fmt"
	"strings"
)

// handleListWorkspaces prints the workspaces the authenticated user can see.
// Used during login when ACCOUNT_URL is set but WORKSPACE_ID hasn't been
// picked yet.
func handleListWorkspaces(ctx context.Context, _ map[string]interface{}) (string, bool) {
	v := NewValidator(ctx, 0)
	ops := []map[string]any{
		{
			"type": "list",
			"obj":  "company",
		},
	}
	resp, err := v.req("list_workspaces", ops)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), true
	}

	// Extract the workspace list from ops[0].list
	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return "No workspaces found", false
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	list, _ := opMap["list"].([]interface{})
	if len(list) == 0 {
		return "No workspaces found", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Workspaces (%d total):\n\n", len(list)))
	for _, item := range list {
		ws, _ := item.(map[string]interface{})
		companyID, _ := ws["company_id"].(string)
		title, _ := ws["title"].(string)
		isOwner, _ := ws["is_owner"].(bool)
		isAdmin, _ := ws["is_admin"].(bool)

		role := "member"
		if isOwner {
			role = "owner"
		} else if isAdmin {
			role = "admin"
		}

		sb.WriteString(fmt.Sprintf("  %-45s  %s  [%s]\n", companyID, title, role))
	}
	return sb.String(), false
}

// handleListProjects prints the projects in a given workspace.
func handleListProjects(ctx context.Context, args map[string]interface{}) (string, bool) {
	companyID, err := strArg(args, "company_id")
	if err != nil {
		return "Error: " + err.Error(), true
	}

	v := NewValidator(ctx, 0)
	ops := []map[string]any{
		{
			"type":       "list",
			"obj":        "projects",
			"obj_id":     0,
			"id":         companyID,
			"company_id": companyID,
			"sort":       "title",
		},
	}
	resp, err := v.req("list_projects", ops)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), true
	}

	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return "No projects found", false
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	if proc, _ := opMap["proc"].(string); proc != "ok" {
		desc, _ := opMap["description"].(string)
		return fmt.Sprintf("Error: %s", desc), true
	}
	list, _ := opMap["list"].([]interface{})
	if len(list) == 0 {
		return "No projects found", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Projects in workspace %s (%d total):\n\n", companyID, len(list)))
	sb.WriteString(fmt.Sprintf("  %-10s  %-35s  %-30s  %s\n", "ID", "Title", "Short name", "Owner"))
	sb.WriteString(fmt.Sprintf("  %s\n", strings.Repeat("-", 95)))
	for _, item := range list {
		p, _ := item.(map[string]interface{})
		projectID := int64(0)
		if f, ok := p["project_id"].(float64); ok {
			projectID = int64(f)
		}
		title, _ := p["title"].(string)
		shortName, _ := p["short_name"].(string)
		ownerLogin, _ := p["owner_login"].(string)
		undeployed := int(0)
		if f, ok := p["undeployed"].(float64); ok {
			undeployed = int(f)
		}
		undeployedStr := ""
		if undeployed > 0 {
			undeployedStr = fmt.Sprintf(" [%d undeployed]", undeployed)
		}
		sb.WriteString(fmt.Sprintf("  %-10d  %-35s  %-30s  %s%s\n",
			projectID, title, shortName, ownerLogin, undeployedStr))
	}
	return sb.String(), false
}

// handleListStages prints the stages (root folders) in a project.
func handleListStages(ctx context.Context, args map[string]interface{}) (string, bool) {
	projectID, err := intArg(args, "project_id")
	if err != nil {
		return "Error: " + err.Error(), true
	}
	companyID, err := strArg(args, "company_id")
	if err != nil {
		return "Error: " + err.Error(), true
	}

	v := NewValidator(ctx, 0)
	ops := []map[string]any{
		{
			"type":       "list",
			"obj":        "project",
			"obj_id":     projectID,
			"id":         companyID,
			"company_id": companyID,
			"sort":       "date",
			"order":      "asc",
		},
	}
	resp, err := v.req("list_stages", ops)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), true
	}

	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return "No stages found", false
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	if proc, _ := opMap["proc"].(string); proc != "ok" {
		desc, _ := opMap["description"].(string)
		return fmt.Sprintf("Error: %s", desc), true
	}
	list, _ := opMap["list"].([]interface{})
	if len(list) == 0 {
		return "No stages found", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Stages in project %d (%d total):\n\n", projectID, len(list)))
	sb.WriteString(fmt.Sprintf("  %-10s  %-20s  %-20s  %s\n", "ID", "Title", "Short name", "Immutable"))
	sb.WriteString(fmt.Sprintf("  %s\n", strings.Repeat("-", 70)))
	for _, item := range list {
		s, _ := item.(map[string]interface{})
		stageID := int64(0)
		if f, ok := s["obj_id"].(float64); ok {
			stageID = int64(f)
		}
		title, _ := s["title"].(string)
		shortName, _ := s["short_name"].(string)
		immutable, _ := s["immutable"].(bool)
		immutableStr := "no"
		if immutable {
			immutableStr = "yes"
		}
		sb.WriteString(fmt.Sprintf("  %-10d  %-20s  %-20s  %s\n", stageID, title, shortName, immutableStr))
	}
	return sb.String(), false
}
