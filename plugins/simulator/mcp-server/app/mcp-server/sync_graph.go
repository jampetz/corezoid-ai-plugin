package mcpserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

// ---- YAML data model ----

type GraphFile struct {
	LayerID string       `yaml:"layerId"`
	Actors  []GraphActor `yaml:"actors"`
	Edges   []GraphEdge  `yaml:"edges"`
}

type GraphActor struct {
	ID          string                 `yaml:"id"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description,omitempty"`
	FormID      int                    `yaml:"formId,omitempty"`
	FormName    string                 `yaml:"formName,omitempty"`
	Picture     string                 `yaml:"picture,omitempty"`
	Color       string                 `yaml:"color,omitempty"`
	Data        map[string]interface{} `yaml:"data,omitempty"`
	Position    struct {
		X int `yaml:"x"`
		Y int `yaml:"y"`
	} `yaml:"position"`
}

type GraphEdge struct {
	Source      string `yaml:"source"`
	Target      string `yaml:"target"`
	SourceTitle string `yaml:"source_title,omitempty"`
	TargetTitle string `yaml:"target_title,omitempty"`
}

// ---- Server response types ----

type layerActor struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Color       string                 `json:"color"`
	FormID      int                    `json:"formId"`
	Picture     string                 `json:"picture"`
	LaID        int                    `json:"laId"`
	Data        map[string]interface{} `json:"data"`
	Position    struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

// formIDFromLayerActor returns the formId to use for API calls.
// Prefers the id found in data keys like "__form__408962:view" over the top-level formId.
func formIDFromLayerActor(sa layerActor) int {
	for key := range sa.Data {
		if strings.HasPrefix(key, "__form__") {
			rest := strings.TrimPrefix(key, "__form__")
			if idx := strings.Index(rest, ":"); idx > 0 {
				if id, err := strconv.Atoi(rest[:idx]); err == nil && id > 0 {
					return id
				}
			}
		}
	}
	return sa.FormID
}

type layerEdge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	LaID       int    `json:"laId"`
	LaIDSource int    `json:"laIdSource"`
	LaIDTarget int    `json:"laIdTarget"`
}

// ---- Helpers ----

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isUUID(s string) bool {
	return uuidRe.MatchString(s)
}

// buildBaseURL returns the same base URL used by all other MCP tools.
func buildBaseURL() string {
	switch {
	case globalApiConfig.Url != "":
		return strings.TrimSuffix(globalApiConfig.Url, "/")
	case globalApiConfig.BaseUrl != "":
		return strings.TrimSuffix(globalApiConfig.BaseUrl, "/")
	case len(globalSwaggerSpec.Servers) > 0:
		return strings.TrimSuffix(globalSwaggerSpec.Servers[0].URL, "/")
	default:
		return "https://api.simulator.company/v/1.0"
	}
}

func papiGET(apiURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", globalApiConfig.Authorization)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func fetchLayerActors(layerID string) ([]layerActor, error) {
	base := buildBaseURL()
	var all []layerActor
	limit, offset := 50, 0
	for {
		u := fmt.Sprintf("%s/graph_layers/paginated/%s?type=nodes&limit=%d&offset=%d", base, layerID, limit, offset)
		body, err := papiGET(u)
		if err != nil {
			return nil, err
		}
		var page struct {
			Data []layerActor `json:"data"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse layer actors: %w (body: %.200s)", err, body)
		}
		all = append(all, page.Data...)
		if len(page.Data) < limit {
			break
		}
		offset += limit
	}
	return all, nil
}

func fetchLayerEdges(layerID string) ([]layerEdge, error) {
	base := buildBaseURL()
	var all []layerEdge
	limit, offset := 50, 0
	for {
		u := fmt.Sprintf("%s/graph_layers/paginated/%s?type=edges&limit=%d&offset=%d", base, layerID, limit, offset)
		body, err := papiGET(u)
		if err != nil {
			return nil, err
		}
		var page struct {
			Data []layerEdge `json:"data"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse layer edges: %w (body: %.200s)", err, body)
		}
		all = append(all, page.Data...)
		if len(page.Data) < limit {
			break
		}
		offset += limit
	}
	return all, nil
}

// manageLayerItem is a single create/delete action for the manageLayer API.
type manageLayerItem struct {
	Action string `json:"action"`
	Data   struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		LaID     int    `json:"laId,omitempty"`
		LaIDSrc  int    `json:"laIdSource,omitempty"`
		LaIDTgt  int    `json:"laIdTarget,omitempty"`
		Position struct {
			X int `json:"x"`
			Y int `json:"y"`
		} `json:"position"`
	} `json:"data"`
}

// callManageLayer sends manageLayer requests in batches of up to 50 items.
func callManageLayer(ctx context.Context, layerID string, items []manageLayerItem) error {
	var manageOp *Operation
	for i, op := range globalOperations {
		if operationToolName(op) == "manageLayer" {
			manageOp = &globalOperations[i]
			break
		}
	}
	if manageOp == nil {
		return fmt.Errorf("manageLayer operation not found")
	}

	const batchSize = 50
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		bodyBytes, err := json.Marshal(batch)
		if err != nil {
			return err
		}
		innerArgs := map[string]interface{}{"body": string(bodyBytes)}
		if injErr := injectManageLayerData(ctx, innerArgs); injErr != nil {
			log.Printf("Warning: callManageLayer injectManageLayerData: %v", injErr)
		}
		innerReq := mcp.CallToolRequest{}
		innerReq.Params.Arguments = innerArgs
		result, err := executeOperation(ctx, *manageOp,
			map[string]interface{}{"layerId": layerID},
			nil, nil, innerReq)
		if err != nil {
			return fmt.Errorf("manageLayer batch %d: %w", i/batchSize, err)
		}
		if result.IsError {
			for _, c := range result.Content {
				if tc, ok := c.(mcp.TextContent); ok {
					return fmt.Errorf("manageLayer batch %d: %s", i/batchSize, tc.Text)
				}
			}
		}
	}
	return nil
}

// createGraphActor creates a single actor and returns its server UUID.
func createGraphActor(ctx context.Context, a GraphActor) (string, error) {
	// formName takes priority over formId when both are specified.
	formID := a.FormID
	if a.FormName != "" {
		if _, loadErr := loadSysForms(); loadErr != nil {
			return "", fmt.Errorf("load sys forms: %w", loadErr)
		}
		if id := resolveFormNameToID(a.FormName); id != 0 {
			formID = id
		} else {
			return "", fmt.Errorf("form name %q not found", a.FormName)
		}
	}
	if formID == 0 {
		return "", fmt.Errorf("actor %q: formId or formName required", a.Title)
	}

	var createOp *Operation
	for i, op := range globalOperations {
		if operationToolName(op) == "createActor" {
			createOp = &globalOperations[i]
			break
		}
	}
	if createOp == nil {
		return "", fmt.Errorf("createActor operation not found")
	}

	body := map[string]interface{}{
		"title":       a.Title,
		"description": a.Description,
		"color":       a.Color,
		"picture":     a.Picture,
	}
	if a.Data != nil {
		body["data"] = a.Data
	}
	omitEmptyFields(body)

	bodyBytes, _ := json.Marshal(body)
	actorArgs := map[string]interface{}{"body": string(bodyBytes)}
	qp := map[string]interface{}{"formId": float64(formID)}

	childFormID, injErr := injectCreateActorData(ctx, actorArgs, qp)
	if injErr != nil {
		log.Printf("Warning: syncGraph createActor data injection: %v", injErr)
	}

	innerReq := mcp.CallToolRequest{}
	innerReq.Params.Arguments = map[string]interface{}{"body": actorArgs["body"]}

	result, err := executeOperation(ctx, *createOp, qp, nil, nil, innerReq)
	if err != nil {
		return "", err
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			if result.IsError {
				return "", fmt.Errorf("createActor: %s", tc.Text)
			}
			cacheActorFormIDFromResult(tc.Text)
			if childFormID != 0 {
				overrideActorFormID(tc.Text, childFormID)
			}
			var resp struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if jsonErr := json.Unmarshal([]byte(tc.Text), &resp); jsonErr == nil && resp.Data.ID != "" {
				return resp.Data.ID, nil
			}
		}
	}
	return "", fmt.Errorf("createActor: no ID in response")
}

// updateGraphActor updates actor fields on the server if they differ. Returns true if an update was made.
func updateGraphActor(ctx context.Context, sa layerActor, fa GraphActor) (bool, error) {
	if sa.Title == fa.Title &&
		sa.Description == fa.Description &&
		sa.Color == fa.Color &&
		sa.Picture == fa.Picture {
		return false, nil
	}

	var updateOp *Operation
	for i, op := range globalOperations {
		if operationToolName(op) == "updateActor" {
			updateOp = &globalOperations[i]
			break
		}
	}
	if updateOp == nil {
		return false, fmt.Errorf("updateActor operation not found")
	}

	body := map[string]interface{}{
		"title":       fa.Title,
		"description": fa.Description,
		"color":       fa.Color,
		"picture":     fa.Picture,
	}
	if fa.Data != nil {
		body["data"] = fa.Data
	}

	bodyBytes, _ := json.Marshal(body)
	innerReq := mcp.CallToolRequest{}
	innerReq.Params.Arguments = map[string]interface{}{"body": string(bodyBytes)}

	// If the form is a child, the API requires the parent formId in the URL.
	childFormID := formIDFromLayerActor(sa)
	apiFormID := childFormID
	if sysForms, sysErr := loadSysForms(); sysErr == nil && sysForms != nil {
		if parentID, isChild, found := findFormInTree(sysForms, childFormID, 0); found && isChild {
			apiFormID = parentID
		}
	}

	result, err := executeOperation(ctx, *updateOp,
		map[string]interface{}{
			"actorId":      sa.ID,
			"formId":       float64(apiFormID),
			"replaceEmpty": false,
		},
		nil, nil, innerReq)
	if err != nil {
		return false, err
	}
	if result.IsError {
		for _, c := range result.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				return false, fmt.Errorf("updateActor %s: %s", sa.ID, tc.Text)
			}
		}
	}
	return true, nil
}

// createEdgeLink creates a directed link between two actors. Returns the link UUID.
func createEdgeLink(ctx context.Context, sourceUUID, targetUUID string) (string, error) {
	var massLinkOp *Operation
	for i, op := range globalOperations {
		if operationToolName(op) == "massLink" {
			massLinkOp = &globalOperations[i]
			break
		}
	}
	if massLinkOp == nil {
		return "", fmt.Errorf("massLink operation not found")
	}

	links := []map[string]interface{}{
		{"source": sourceUUID, "target": targetUUID},
	}
	bodyBytes, _ := json.Marshal(links)
	innerArgs := map[string]interface{}{"body": string(bodyBytes)}
	if injErr := injectMassLinkData(ctx, innerArgs); injErr != nil {
		return "", fmt.Errorf("inject massLink data: %w", injErr)
	}
	innerReq := mcp.CallToolRequest{}
	innerReq.Params.Arguments = innerArgs

	qp := map[string]interface{}{}
	if accID := os.Getenv("WORKSPACE_ID"); accID != "" {
		qp["accId"] = accID
	}

	result, err := executeOperation(ctx, *massLinkOp, qp, nil, nil, innerReq)
	if err != nil {
		return "", err
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			if result.IsError {
				return "", fmt.Errorf("massLink: %s", tc.Text)
			}
			// massLink returns {"data":[{"error":false,"data":{"id":"..."}}]}
			var resp struct {
				Data []struct {
					Error bool `json:"error"`
					Data  struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"data"`
			}
			if jsonErr := json.Unmarshal([]byte(tc.Text), &resp); jsonErr == nil && len(resp.Data) > 0 && resp.Data[0].Data.ID != "" {
				return resp.Data[0].Data.ID, nil
			}
		}
	}
	return "", fmt.Errorf("massLink: no link ID in response")
}

// ---- Main handlers ----

func handlePushGraphFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if authResult := ensureAuth(ctx); authResult != nil {
		return authResult, nil
	}

	args := req.GetArguments()
	layerID, _ := args["layerId"].(string)
	if layerID == "" {
		return mcp.NewToolResultError("[Error] layerId is required"), nil
	}
	filePath := layerID + ".yaml"

	// 1. Read and parse YAML
	rawData, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] cannot read file %s: %v", filePath, err)), nil
	}
	var graph GraphFile
	if err := yaml.Unmarshal(rawData, &graph); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] cannot parse YAML: %v", err)), nil
	}
	if graph.LayerID == "" {
		graph.LayerID = layerID
	}

	// 2. Fetch current layer state
	serverActors, err := fetchLayerActors(graph.LayerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] fetch layer actors: %v", err)), nil
	}
	serverEdges, err := fetchLayerEdges(graph.LayerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] fetch layer edges: %v", err)), nil
	}

	// Build lookup maps from server state
	serverActorByUUID := make(map[string]layerActor, len(serverActors))
	for _, a := range serverActors {
		serverActorByUUID[a.ID] = a
	}

	type edgePair struct{ src, tgt string }
	serverEdgeByPair := make(map[edgePair]layerEdge, len(serverEdges))
	for _, e := range serverEdges {
		serverEdgeByPair[edgePair{e.Source, e.Target}] = e
	}

	// 3. Sync actors
	// idMap: file id (local or UUID) → server UUID
	idMap := make(map[string]string, len(graph.Actors))
	fileUUIDs := make(map[string]bool, len(graph.Actors))

	var nodeManageItems []manageLayerItem
	var posUpdates []map[string]interface{}
	stats := struct{ created, updated, unchanged, deleted int }{}

	for i := range graph.Actors {
		a := &graph.Actors[i]
		origID := a.ID

		if isUUID(origID) {
			idMap[origID] = origID
			fileUUIDs[origID] = true

			sa, onLayer := serverActorByUUID[origID]
			if onLayer {
				changed, updateErr := updateGraphActor(ctx, sa, *a)
				if updateErr != nil {
					log.Printf("Warning: update actor %s: %v", origID, updateErr)
				}
				if changed {
					stats.updated++
				} else {
					stats.unchanged++
				}
				// Queue position update if changed
				if sa.Position.X != a.Position.X || sa.Position.Y != a.Position.Y {
					posUpdates = append(posUpdates, map[string]interface{}{
						"id":       sa.LaID,
						"position": map[string]int{"x": a.Position.X, "y": a.Position.Y},
					})
				}
			} else {
				// UUID in file but not on layer → add to layer
				var item manageLayerItem
				item.Action = "create"
				item.Data.ID = origID
				item.Data.Type = "node"
				item.Data.Position.X = a.Position.X
				item.Data.Position.Y = a.Position.Y
				nodeManageItems = append(nodeManageItems, item)
				stats.created++
			}
		} else {
			// New actor: create on server, then add to layer
			serverUUID, createErr := createGraphActor(ctx, *a)
			if createErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] create actor %q: %v", a.Title, createErr)), nil
			}
			idMap[origID] = serverUUID
			a.ID = serverUUID
			fileUUIDs[serverUUID] = true

			var item manageLayerItem
			item.Action = "create"
			item.Data.ID = serverUUID
			item.Data.Type = "node"
			item.Data.Position.X = a.Position.X
			item.Data.Position.Y = a.Position.Y
			nodeManageItems = append(nodeManageItems, item)
			stats.created++
		}
	}

	// Server actors not in file → remove from layer
	for _, sa := range serverActors {
		if !fileUUIDs[sa.ID] {
			var item manageLayerItem
			item.Action = "delete"
			item.Data.ID = sa.ID
			item.Data.Type = "node"
			item.Data.LaID = sa.LaID
			nodeManageItems = append(nodeManageItems, item)
			stats.deleted++
		}
	}

	// Apply node changes via manageLayer
	if len(nodeManageItems) > 0 {
		if err := callManageLayer(ctx, graph.LayerID, nodeManageItems); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] manageLayer nodes: %v", err)), nil
		}
	}

	// Update positions for existing actors
	if len(posUpdates) > 0 {
		var posOp *Operation
		for i, op := range globalOperations {
			if operationToolName(op) == "layerActorsPosition" {
				posOp = &globalOperations[i]
				break
			}
		}
		if posOp != nil {
			bodyBytes, _ := json.Marshal(posUpdates)
			innerReq := mcp.CallToolRequest{}
			innerReq.Params.Arguments = map[string]interface{}{"body": string(bodyBytes)}
			if _, posErr := executeOperation(ctx, *posOp,
				map[string]interface{}{"layerId": graph.LayerID},
				nil, nil, innerReq); posErr != nil {
				log.Printf("Warning: update positions: %v", posErr)
			}
		}
	}

	// 4. Write updated YAML (actor IDs and edge references updated in place)
	for i := range graph.Edges {
		if uuid, ok := idMap[graph.Edges[i].Source]; ok {
			graph.Edges[i].Source = uuid
		}
		if uuid, ok := idMap[graph.Edges[i].Target]; ok {
			graph.Edges[i].Target = uuid
		}
	}
	updatedYAML, err := yaml.Marshal(&graph)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] marshal YAML: %v", err)), nil
	}
	if err := os.WriteFile(filePath, updatedYAML, 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] write YAML: %v", err)), nil
	}

	// 5. Re-fetch layer to get laId for newly added nodes (needed for edge placement)
	updatedActors, err := fetchLayerActors(graph.LayerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] re-fetch layer actors: %v", err)), nil
	}
	laIDByUUID := make(map[string]int, len(updatedActors))
	for _, a := range updatedActors {
		laIDByUUID[a.ID] = a.LaID
	}

	// 6. Sync edges
	var edgeManageItems []manageLayerItem
	fileEdgePairs := make(map[edgePair]bool, len(graph.Edges))
	statsEdge := struct{ created, deleted int }{}

	for _, e := range graph.Edges {
		srcUUID := idMap[e.Source]
		if srcUUID == "" {
			srcUUID = e.Source
		}
		tgtUUID := idMap[e.Target]
		if tgtUUID == "" {
			tgtUUID = e.Target
		}

		pair := edgePair{srcUUID, tgtUUID}
		fileEdgePairs[pair] = true

		if _, exists := serverEdgeByPair[pair]; !exists {
			linkID, linkErr := createEdgeLink(ctx, srcUUID, tgtUUID)
			if linkErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] create link %s→%s: %v", srcUUID, tgtUUID, linkErr)), nil
			}
			srcLaID := laIDByUUID[srcUUID]
			tgtLaID := laIDByUUID[tgtUUID]
			if srcLaID != 0 && tgtLaID != 0 {
				var item manageLayerItem
				item.Action = "create"
				item.Data.ID = linkID
				item.Data.Type = "edge"
				item.Data.LaIDSrc = srcLaID
				item.Data.LaIDTgt = tgtLaID
				edgeManageItems = append(edgeManageItems, item)
			}
			statsEdge.created++
		}
	}

	// Server edges not in file → remove from layer
	for _, se := range serverEdges {
		pair := edgePair{se.Source, se.Target}
		if !fileEdgePairs[pair] {
			var item manageLayerItem
			item.Action = "delete"
			item.Data.ID = se.ID
			item.Data.Type = "edge"
			item.Data.LaID = se.LaID
			edgeManageItems = append(edgeManageItems, item)
			statsEdge.deleted++
		}
	}

	if len(edgeManageItems) > 0 {
		if err := callManageLayer(ctx, graph.LayerID, edgeManageItems); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] manageLayer edges: %v", err)), nil
		}
	}

	out, _ := json.Marshal(map[string]interface{}{
		"layerId": graph.LayerID,
		"actors": map[string]int{
			"created":   stats.created,
			"updated":   stats.updated,
			"unchanged": stats.unchanged,
			"deleted":   stats.deleted,
		},
		"edges": map[string]int{
			"created": statsEdge.created,
			"deleted": statsEdge.deleted,
		},
		"fileUpdated": true,
	})
	return mcp.NewToolResultText(string(out)), nil
}

// handlePullGraphFile fetches all actors and edges from a layer and writes
// them to <layerId>.yaml in the current working directory.
func handlePullGraphFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if authResult := ensureAuth(ctx); authResult != nil {
		return authResult, nil
	}

	args := req.GetArguments()
	layerID, _ := args["layerId"].(string)
	if layerID == "" {
		return mcp.NewToolResultError("[Error] layerId is required"), nil
	}
	filePath := layerID + ".yaml"

	// Fetch actors
	serverActors, err := fetchLayerActors(layerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] fetch layer actors: %v", err)), nil
	}

	// Fetch edges
	serverEdges, err := fetchLayerEdges(layerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] fetch layer edges: %v", err)), nil
	}

	// Build actor title lookup for edge source_title / target_title
	titleByUUID := make(map[string]string, len(serverActors))
	for _, a := range serverActors {
		titleByUUID[a.ID] = a.Title
	}

	// Build GraphFile
	graph := GraphFile{LayerID: layerID}

	// Pre-load sys forms so resolveFormIDToName works (best-effort, non-fatal).
	if _, err := loadSysForms(); err != nil {
		log.Printf("Warning: exportGraph loadSysForms: %v", err)
	}

	for _, sa := range serverActors {
		var ga GraphActor
		ga.ID = sa.ID
		ga.Title = sa.Title
		ga.Description = sa.Description
		ga.Color = sa.Color
		ga.Picture = sa.Picture
		ga.FormID = formIDFromLayerActor(sa)
		if name := resolveFormIDToName(ga.FormID); name != "" {
			ga.FormName = name
		}
		ga.Position.X = sa.Position.X
		ga.Position.Y = sa.Position.Y
		graph.Actors = append(graph.Actors, ga)
	}

	for _, se := range serverEdges {
		graph.Edges = append(graph.Edges, GraphEdge{
			Source:      se.Source,
			Target:      se.Target,
			SourceTitle: titleByUUID[se.Source],
			TargetTitle: titleByUUID[se.Target],
		})
	}

	data, err := yaml.Marshal(&graph)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] marshal YAML: %v", err)), nil
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("[Error] write file: %v", err)), nil
	}

	out, _ := json.Marshal(map[string]interface{}{
		"layerId":   layerID,
		"filePath":  filePath,
		"actors":    len(graph.Actors),
		"edges":     len(graph.Edges),
	})
	return mcp.NewToolResultText(string(out)), nil
}
