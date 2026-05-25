package main

import (
	"fmt"
)

// wsItem holds a single workspace entry returned by the list-workspaces API.
type wsItem struct {
	companyID string
	title     string
	role      string // "owner", "admin", or "member"
}

type projectItem struct {
	projectID int64
	title     string
	shortName string
}

type stageItem struct {
	stageID   int64
	title     string
	shortName string
	immutable bool
}

// fetchWorkspaceList calls the Corezoid API and returns the list of workspaces
// available to the authenticated user. Requires apiURL and apiToken to be set.
func fetchWorkspaceList() ([]wsItem, error) {
	v := NewValidator(0)
	ops := []map[string]any{{"type": "list", "obj": "company"}}
	resp, err := v.req("list_workspaces", ops)
	if err != nil {
		return nil, err
	}
	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	list, _ := opMap["list"].([]interface{})

	var result []wsItem
	for _, item := range list {
		ws, _ := item.(map[string]interface{})
		companyID, _ := ws["company_id"].(string)
		title, _ := ws["title"].(string)
		isOwner, _ := ws["is_owner"].(bool)
		isAdmin, _ := ws["is_admin"].(bool)
		if companyID == "" {
			continue
		}
		role := "member"
		if isOwner {
			role = "owner"
		} else if isAdmin {
			role = "admin"
		}
		result = append(result, wsItem{companyID: companyID, title: title, role: role})
	}
	return result, nil
}

// fetchProjectList calls the Corezoid API and returns the projects in a workspace.
func fetchProjectList(companyID string) ([]projectItem, error) {
	v := NewValidator(0)
	ops := []map[string]any{
		{"type": "list", "obj": "projects", "obj_id": 0, "id": companyID, "company_id": companyID, "sort": "title"},
	}
	resp, err := v.req("list_projects", ops)
	if err != nil {
		return nil, err
	}
	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	if proc, _ := opMap["proc"].(string); proc != "ok" {
		desc, _ := opMap["description"].(string)
		return nil, fmt.Errorf("%s", desc)
	}
	list, _ := opMap["list"].([]interface{})

	var result []projectItem
	for _, item := range list {
		p, _ := item.(map[string]interface{})
		projectID := int64(0)
		if f, ok := p["project_id"].(float64); ok {
			projectID = int64(f)
		}
		if projectID == 0 {
			continue
		}
		title, _ := p["title"].(string)
		shortName, _ := p["short_name"].(string)
		result = append(result, projectItem{projectID: projectID, title: title, shortName: shortName})
	}
	return result, nil
}

// fetchStageList calls the Corezoid API and returns the stages (folders) in a project.
func fetchStageList(companyID string, projectID int64) ([]stageItem, error) {
	v := NewValidator(0)
	ops := []map[string]any{
		{"type": "list", "obj": "project", "obj_id": projectID, "id": companyID, "company_id": companyID, "sort": "date", "order": "asc"},
	}
	resp, err := v.req("list_stages", ops)
	if err != nil {
		return nil, err
	}
	opsArr, _ := resp["ops"].([]interface{})
	if len(opsArr) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	opMap, _ := opsArr[0].(map[string]interface{})
	if proc, _ := opMap["proc"].(string); proc != "ok" {
		desc, _ := opMap["description"].(string)
		return nil, fmt.Errorf("%s", desc)
	}
	list, _ := opMap["list"].([]interface{})

	var result []stageItem
	for _, item := range list {
		s, _ := item.(map[string]interface{})
		sid := int64(0)
		if f, ok := s["obj_id"].(float64); ok {
			sid = int64(f)
		}
		if sid == 0 {
			continue
		}
		title, _ := s["title"].(string)
		shortName, _ := s["short_name"].(string)
		immutable, _ := s["immutable"].(bool)
		result = append(result, stageItem{stageID: sid, title: title, shortName: shortName, immutable: immutable})
	}
	return result, nil
}

// ensureTokenAuth checks that a valid API token is present.
func ensureTokenAuth() error {
	if apiToken == "" {
		creds, err := loadCredentials()
		if err == nil && creds != nil && !isCredentialsExpired(creds) {
			apiToken = creds.AccessToken
		}
	}
	if apiToken == "" {
		return fmt.Errorf("[Error] Not authenticated: missing [ACCESS_TOKEN]. Invoke the 'corezoid-init' skill to set up credentials (use the Skill tool with skill=\"corezoid-init\").")
	}
	return nil
}

// ensureAuth checks that all required credentials are set.
// Returns an error with instructions if any value is missing.
func ensureAuth() error {
	if err := ensureTokenAuth(); err != nil {
		return err
	}

	var missing []string
	if accountURL == "" {
		missing = append(missing, "ACCOUNT_URL")
	}
	// WORKSPACE_ID is optional: personal-workspace accounts have no companyID.
	// `Executor.req` strips the empty placeholder from outbound ops in that case.
	if stageID == 0 {
		missing = append(missing, "COREZOID_STAGE_ID")
	}

	if len(missing) > 0 {
		return fmt.Errorf("[Error] Not authenticated: missing %v. Invoke the 'corezoid-init' skill to set up credentials (use the Skill tool with skill=\"corezoid-init\").", missing)
	}
	return nil
}
