package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var reEnvVar = regexp.MustCompile(`\{\{env_var\[@([a-z0-9-]+)]}}`)

type Executor struct {
	NodeResponses []map[string]interface{} // Store responses from node creation
	ProcessID     int                      // Store the process ID
	APILogin      string                   // API Login for authentication
	Token         string                   // API Token for authentication
	APISecret     string                   // API Secret for authentication
	APIUrl        string                   // API URL for requests
	NodeIDMap     map[string]NodeInfo      // Maps node ID to server-assigned obj_id and other info
	Debug         bool                     // Enable debug logging
	Version       int
	NewProc       bool
}

func NewValidator(inProcessID int) *Executor {
	v := &Executor{
		APILogin:  "",
		APISecret: "",
		APIUrl:    apiURL,
		Token:     apiToken,
		NodeIDMap: make(map[string]NodeInfo),
		Debug:     debug,
		Version:   int(time.Now().Unix()),
		ProcessID: inProcessID,
	}
	return v

}

func (v *Executor) PullZip(id int, objType string) ([]byte, error) {
	ops := []map[string]any{
		{
			"type":       "create",
			"obj":        "obj_scheme",
			"obj_id":     id,
			"company_id": workspaceID,
			"obj_type":   objType,
			"with_alias": true,
			"async":      false,
			"format":     "zip",
		},
	}

	// Send the request to share access
	response, _ := v.req("export_process", ops)

	if response["request_proc"] != "ok" {
		return nil, fmt.Errorf("failed to export process: %v, %v, %v", response, workspaceID, id)
	}
	if response["ops"] == nil {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	ops1 := response["ops"].([]any)
	if len(ops1) == 0 {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	downloadURL := ops1[0].(map[string]any)["download_url"]

	if downloadURL == nil {
		fmt.Println("rsp", ops1)
		fmt.Println("req", ops)
		return nil, fmt.Errorf("failed to export process: no download_url in response")
	}
	downloadURL1 := downloadURL.(string)

	// download file
	// Create HTTP client with insecure TLS configuration for download
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("GET", downloadURL1, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Simulator %s", v.Token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download process: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read process: %v", err)
	}

	return body, nil
}

func (v *Executor) PullFolder(id int, objType string) ([]any, error) {
	ops := []map[string]any{
		{
			"type":       "create",
			"obj":        "obj_scheme",
			"obj_id":     id,
			"company_id": workspaceID,
			"obj_type":   objType,
			"with_alias": true,
			"async":      false,
			"format":     "json",
		},
	}

	// Send the request to share access
	response, _ := v.req("export_process", ops)

	if response["request_proc"] != "ok" {
		return nil, fmt.Errorf("failed to export process: %v", response)
	}
	if response["ops"] == nil {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	ops1 := response["ops"].([]any)
	if len(ops1) == 0 {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	downloadURL := ops1[0].(map[string]any)["download_url"]
	if downloadURL == nil {
		return nil, fmt.Errorf("failed to export process: no download_url in response")
	}
	downloadURL1 := downloadURL.(string)

	// download file
	// Create HTTP client with insecure TLS configuration for download
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("GET", downloadURL1, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Simulator %s", v.Token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download process: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read process: %v", err)
	}
	var processes []any
	err = json.Unmarshal(body, &processes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal process: %v", err)
	}

	return processes, nil
}

func (v *Executor) ExportProcess() (any, error) {
	ops := []map[string]any{
		{
			"type":       "create",
			"obj":        "obj_scheme",
			"obj_id":     v.ProcessID,
			"company_id": workspaceID,
			"obj_type":   "conv",
			"with_alias": true,
			"async":      false,
			"format":     "json",
		},
	}

	// Send the request to share access
	response, _ := v.req("export_process", ops)

	if response["request_proc"] != "ok" {
		return nil, fmt.Errorf("failed to export process: %v", response)
	}
	if response["ops"] == nil {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	ops1 := response["ops"].([]any)
	if len(ops1) == 0 {
		return nil, fmt.Errorf("failed to export process: no ops in response")
	}
	downloadURL := ops1[0].(map[string]any)["download_url"]
	if downloadURL == nil {
		return nil, fmt.Errorf("failed to export process: no download_url in response")
	}
	downloadURL1 := downloadURL.(string)

	// download file
	// Create HTTP client with insecure TLS configuration for download
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("GET", downloadURL1, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Simulator %s", v.Token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download process: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read process: %v", err)
	}
	var process []any
	err = json.Unmarshal(body, &process)
	if err != nil {
		fmt.Println(string(body))
		return nil, fmt.Errorf("failed to unmarshal process: %v", err)
	}

	return process, nil
}

// ProcessJSONContent is a convenience function that processes JSON content,
// creates nodes on the server, modifies them with content, and returns the validator
// with the responses along with the enhanced processData.
func (validator *Executor) ProcessJSON(filePath, jsonContent string) (newProcessData map[string]interface{}, err error) {
	defer func() {
		if err != nil {
			if validator != nil {
				validator.DeleteVersion(validator.Version)
			}
		}
	}()

	// Process the JSON content
	// Create a validator and process the JSON

	var processDataOfAI map[string]interface{}
	err = json.Unmarshal([]byte(jsonContent), &processDataOfAI)
	if err != nil {
		err = fmt.Errorf("error parsing JSON: %v", err)
		return nil, err
	}

	if validator.Debug {
		logger.Debug("JSON parsed, processDataLength=%d", len(processDataOfAI))
	}
	// First create an empty process
	newProcessData = processDataOfAI
	nodes, err := getNodes(processDataOfAI)
	if err != nil {
		err = fmt.Errorf("error getting nodes: %v", err)
		return nil, err
	}
	if validator.ProcessID == 0 {
		validator.NewProc = true
		validator.ProcessID = validator.CreateEmptyProcess(0, processDataOfAI["title"].(string), "")
		if validator.ProcessID == 0 {
			err = fmt.Errorf("failed to create process")
			return nil, err
		}
	} else {
		oldProcessData, err := validator.GetProcessByID(validator.ProcessID)
		if err != nil {
			logger.Error("Failed to get process: %v", err)
			return nil, fmt.Errorf("error getting process: %v", err)
		}
		if commits, ok := oldProcessData["commits"].(map[string]interface{}); ok && commits != nil {
			if oldVer, ok := commits["version"]; ok && oldVer != nil {
				oldVer1 := int(oldVer.(float64))
				if oldVer1 > 0 {
					//need to reset version
					validator.DeleteVersion(oldVer1)
				}
			}
		}

		oldNodes, ok := oldProcessData["list"].([]interface{})
		if !ok {
			oldProcessDataBin, _ := json.Marshal(oldProcessData)
			fmt.Printf("COREZOID_PROC_ID: %d, server error rsp: %s\n", validator.ProcessID, oldProcessDataBin)
			err = fmt.Errorf("error getting nodes")
			return nil, err
		}
		nodes = validator.DeleteNotUsedNodes(oldNodes, nodes)
		for _, oldNode := range oldNodes {
			if int(oldNode.(map[string]interface{})["obj_type"].(float64)) == 1 {
				oldNodeID := oldNode.(map[string]interface{})["obj_id"].(string)
				newNodeID := ""
				for i, newNode := range nodes {
					if int(newNode.(map[string]interface{})["obj_type"].(float64)) == 1 {
						newNodeID = newNode.(map[string]interface{})["id"].(string)
						newNode.(map[string]interface{})["existed"] = true
						newNode.(map[string]interface{})["id"] = oldNodeID
						validator.NodeIDMap[newNodeID] = NodeInfo{
							Type:     1,
							Name:     "Start",
							Icon:     "",
							ServerID: oldNodeID,
						}
						nodes[i] = newNode
						break
					}
				}
				if newNodeID == "" {
					panic("No start node found")
				}
				break
			}
		}
	}

	// Check if the process data is valid
	if len(newProcessData) == 0 {
		err = fmt.Errorf("no process data found in JSON")
		return nil, err
	}

	// Then create nodes from the JSON

	err = validator.CreateNodesFromJSON(nodes)
	if err != nil {
		err = fmt.Errorf("error creating nodes: %v", err)
		return nil, err
	}
	changed := false
	for inID, extInfo := range validator.NodeIDMap {
		if inID == extInfo.ServerID {
			continue
		}
		changed = true
		jsonContent = strings.Replace(jsonContent, "\""+inID+"\"", "\""+extInfo.ServerID+"\"", -1)
	}
	if changed {
		// save changes
		err = ioutil.WriteFile(filePath, []byte(jsonContent), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write file: %v", err)
		}
		err = json.Unmarshal([]byte(jsonContent), &newProcessData)
		if err != nil {
			logger.Error("Failed to parse1 JSON: %v", err)
			return nil, fmt.Errorf("error parsing1 JSON: %v", err)
		}
	}

	nodes, err = getNodes(newProcessData)
	if err != nil {
		err = fmt.Errorf("error getting nodes: %v", err)
		return nil, err
	}
	params, _ := newProcessData["params"].([]interface{})
	err = validator.SetParams(params)
	if err != nil {
		err = fmt.Errorf("error setting params: %v", err)
		return nil, err
	}

	// Modify the nodes with their content
	//time.Sleep(10 * time.Second)
	err = validator.ModifyNodes(nodes)
	if err != nil {
		return nil, err
	}

	// Compile API code nodes
	err = validator.CompileAPICode(nodes)
	if err != nil {
		err = fmt.Errorf("failed to compile API code: %v", err)
		return nil, err
		// Continue execution even if compilation fails
	}

	// Commit the changes
	commitResponse := validator.Commit()

	// Check if the commit was successful
	if commitResponse == nil {
		return nil, fmt.Errorf("failed to commit changes: no response from server")
	}

	// Check if the request was processed successfully
	if requestProc, ok := commitResponse["request_proc"].(string); !ok || requestProc != "ok" {
		err = fmt.Errorf("failed to commit changes: request_proc not ok")
		return nil, err
	}

	// Check each operation result
	if opsArray, ok := commitResponse["ops"].([]interface{}); ok {
		var errorMsgs []string

		for i, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					// Get node information if available
					nodeInfo := fmt.Sprintf("Operation %d", i+1)
					if objID, ok := opMap["obj_id"].(string); ok {
						// Try to map obj_id back to node ID
						nodeInfo = fmt.Sprintf("Node with obj_id %s", objID)
					}

					// Check for description field (new format)
					if description, ok := opMap["description"].(string); ok {
						errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", nodeInfo, description))
						continue
					}

					// Check for errors structure (old format)
					if errors, ok := opMap["errors"].(map[string]interface{}); ok {
						// Build error message with mapped node IDs
						for objID, errMsgs := range errors {
							// Map obj_id back to node ID
							nodeID := objID

							// Process error messages for this node
							if errArray, ok := errMsgs.([]interface{}); ok {
								for _, errMsg := range errArray {
									if msg, ok := errMsg.(string); ok {
										errorMsgs = append(errorMsgs, fmt.Sprintf("Error in the node %s: %s", nodeID, msg))
									}
								}
							}
						}
						continue
					}

					// Fallback to simple error message if no structured errors
					errorMsg := "unknown error"
					if msg, ok := opMap["message"].(string); ok {
						errorMsg = msg
					}
					errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", nodeInfo, errorMsg))
				}
			}
		}

		if len(errorMsgs) > 0 {
			err = fmt.Errorf("failed to commit changes: %s", strings.Join(errorMsgs, "\n"))
			return nil, err
		}
	}

	return newProcessData, nil
}

// CreateNodesFromJSON processes a JSON file containing process data,
// extracts all nodes, creates them on the server using req(),
// stores the server responses in the Executor structure, and
// returns the enhanced processData with obj_id added to each node.
// jsonContent is the content of the JSON file as a string.
func (v *Executor) CreateNodesFromJSON(nodes []any) error {

	// Parse the JSON content

	// Create operations for each node
	var ops []map[string]any
	for i, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			if v.Debug {
				logger.Debug("Skipping invalid node, index=%d", i)
			}
			continue
		}
		// Get node ID for logging
		nodeID, _ := nodeMap["id"].(string)
		nodeTitle, _ := nodeMap["title"].(string)
		nodeType := int(nodeMap["obj_type"].(float64))
		nodeIcon := ""
		extraStr, _ := nodeMap["extra"].(string)
		var extra map[string]interface{}
		if extraStr != "" {
			json.Unmarshal([]byte(extraStr), &extra)
		}
		if icon1, ok := extra["icon"]; ok {
			nodeIcon = icon1.(string)
		}

		if existed, _ := nodeMap["existed"].(bool); existed {
			v.NodeIDMap[nodeMap["id"].(string)] = NodeInfo{
				Type: nodeType,
				Name: nodeTitle, Icon: nodeIcon, ServerID: nodeMap["id"].(string)}
			continue
		}

		if v.Debug {
			logger.Debug("Processing node, index=%d, id=%s, title=%s", i, nodeID, nodeTitle)
		}

		// Create operation for this node
		op := map[string]any{
			"id":          nodeMap["id"],
			"type":        "create",
			"obj":         "node",
			"conv_id":     v.ProcessID,
			"title":       nodeMap["title"],
			"description": nodeMap["description"],
			"obj_type":    nodeMap["obj_type"],
			"version":     v.Version,
		}
		// Extract extra

		v.NodeIDMap[nodeMap["id"].(string)] = NodeInfo{
			Type: nodeType,
			Name: nodeTitle, Icon: nodeIcon}
		ops = append(ops, op)
	}

	// If no nodes were found, return an error
	if len(ops) == 0 {
		return nil
	}

	if v.Debug {
		logger.Debug("Created operations for nodes, operationCount=%d", len(ops))
	}

	// Send the request to create nodes
	response, _ := v.req("create_nodes", ops)

	if response == nil {
		logger.Error("Failed to create nodes: no response from server")
		return fmt.Errorf("failed to create nodes: no response from server")
	}

	// Create a map to store node UUIDs to their corresponding operations
	nodeUUIDToOp := make(map[string]map[string]any)
	for _, op := range ops {
		if id, ok := op["id"].(string); ok {
			nodeUUIDToOp[id] = op
		}
	}

	// Store the response in the Executor and populate the NodeIDMap
	if response != nil && response["ops"] != nil {
		if opsArray, ok := response["ops"].([]interface{}); ok {
			if v.Debug {
				logger.Debug("Processing node creation responses, responseCount=%d", len(opsArray))
			}

			// Convert each operation response to a map
			for i, op := range opsArray {
				if opMap, ok := op.(map[string]interface{}); ok {
					v.NodeResponses = append(v.NodeResponses, opMap)

					// Extract the node ID and UUID
					if id, ok := opMap["id"].(string); ok {
						if objID, ok := opMap["obj_id"].(string); ok {
							// Store the mapping in the Executor's NodeIDMap
							nodeInfo := v.NodeIDMap[id]
							nodeInfo.ServerID = objID
							v.NodeIDMap[id] = nodeInfo

							if v.Debug {
								logger.Debug("Mapped node ID, index=%d, nodeID=%s, objID=%s", i, id, objID)
							}
						}
					}
				}
			}

		}
	} else {
		logger.Error("Invalid response format from server")
	}

	logger.Debug("Node creation completed, nodeCount=%d", len(v.NodeIDMap))
	return nil
}

func (v *Executor) createTask(ref string, taskData map[string]interface{}) error {

	ops := []map[string]any{
		{
			"obj":     "task",
			"type":    "create",
			"action":  "user",
			"ref":     ref,
			"conv_id": v.ProcessID,
			"data":    taskData,
		},
	}

	if v.Debug {
		logger.Debug("Sending create task request")
	}

	response, _ := v.req("create_task", ops)
	// Check each operation result
	if opsArray, ok := response["ops"].([]interface{}); ok {
		for _, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					// Fallback to simple error message if no structured errors
					errorMsg := "unknown error"
					if msg, ok := opMap["description"].(string); ok {
						errorMsg = msg
					}
					return fmt.Errorf("failed to create task code: %s", errorMsg)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("failed to create task: no response from server")
	// Check if response contains the task ID
}

func (v *Executor) showTask(ref string) (data map[string]interface{}, err error) {

	ops := []map[string]any{
		{
			"obj":     "task",
			"type":    "show",
			"action":  "user",
			"ref":     ref,
			"conv_id": v.ProcessID,
		},
	}

	if v.Debug {
		logger.Debug("Sending create task request")
	}

	response, _ := v.req("show_task", ops)
	// Check each operation result
	if opsArray, ok := response["ops"].([]interface{}); ok {
		for _, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					// Fallback to simple error message if no structured errors
					errorMsg := "unknown error"
					if msg, ok := opMap["description"].(string); ok {
						errorMsg = msg
					}
					return nil, fmt.Errorf("failed to show task code: %s", errorMsg)
				}
				return opMap, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to find task: no response from server")
	// Check if response contains the task ID
}

func (v *Executor) GetProcessByID(id int) (rsp map[string]any, err error) {
	//{"ops":[{"obj_id":"1661064","type":"show","obj":"conv","company_id":"28eecad5-ebd4-4621-ae21-af4f568bcd94"}]}
	ops := []map[string]any{
		{
			"obj_id":     id,
			"type":       "list",
			"obj":        "conv",
			"company_id": workspaceID,
		},
	}

	if v.Debug {
		logger.Debug("Sending get process request")
	}

	response, _ := v.req("get_process", ops)

	// Check if response contains the process data
	if response != nil && response["ops"] != nil {
		if opsArray, ok := response["ops"].([]interface{}); ok && len(opsArray) > 0 {
			if firstOp, ok := opsArray[0].(map[string]interface{}); ok {
				return firstOp, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to get process: no response from server")
}

func (v *Executor) CreateEmptyProcess(folderID int, title, desc string) int {
	if title == "" {
		title = time.Now().String()
	}

	ops := []map[string]any{
		{
			"title":       title,
			"description": desc,
			"folder_id":   folderID,
			"company_id":  workspaceID,
			"obj":         "conv",
			"create_mode": "without_nodes",
			"conv_type":   "process",
			"type":        "create",
			"obj_type":    0,
			"status":      "active",
		},
	}

	if v.Debug {
		logger.Debug("Sending create empty process request")
	}

	response, err := v.req("create_process", ops)
	if err != nil {
		fmt.Println(response)
		logger.Error("Failed to create empty process: %v", err)
		return 0
	}

	// Check if response contains the process ID
	if response != nil && response["ops"] != nil {
		if opsArray, ok := response["ops"].([]interface{}); ok && len(opsArray) > 0 {
			if firstOp, ok := opsArray[0].(map[string]interface{}); ok {
				if objID, ok := firstOp["obj_id"].(float64); ok {
					v.ProcessID = int(objID)
					logger.Debug("Empty process created: %d", v.ProcessID)
					return v.ProcessID
				}
			}
		}
	}

	logger.Error("Failed to create empty process")
	return 0
}

func (v *Executor) SetParams(params []interface{}) error {
	ops := []map[string]any{
		{
			"obj_id":     v.ProcessID,
			"ref_mask":   true,
			"type":       "modify",
			"obj":        "conv_params",
			"params":     params,
			"company_id": workspaceID,
		},
	}

	if v.Debug {
		logger.Debug("Sending set params request")
	}

	response, _ := v.req("set_params", ops)
	if response == nil {
		return fmt.Errorf("failed to set params: no response from server")
	}
	if requestProc, ok := response["request_proc"].(string); !ok || requestProc != "ok" {
		return fmt.Errorf("failed to set params: request_proc not ok")
	}
	return nil
}

// CompileAPICode processes all nodes with api_code logic type,
// loads and compiles the code on the server.
// Returns error if any operation fails.
func (v *Executor) CompileAPICode(nodes []interface{}) error {

	for _, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}
		if condition, ok := nodeMap["condition"].(map[string]interface{}); ok {
			if logicsArray, ok := condition["logics"].([]interface{}); ok {
				for _, logic := range logicsArray {
					if logicMap, ok := logic.(map[string]interface{}); ok {
						lt, _ := logicMap["type"].(string)
						if lt == "api_code" {
							// Get the node ID, language, and source code
							nodeID, ok := nodeMap["id"].(string)
							if !ok {
								continue
							}

							lang, _ := logicMap["lang"].(string)
							if lang == "" {
								lang = "js" // Default to JavaScript if not specified
							}

							src, _ := logicMap["src"].(string)
							if src == "" {
								// Try to get code from the 'code' field as fallback
								src, _ = logicMap["code"].(string)
								if src == "" {
									continue // Skip if no source code
								}
							}

							// First API call: Load the code
							loadOps := []map[string]any{
								{
									"type":    "load",
									"obj":     "api_code",
									"conv_id": v.ProcessID,
									"node_id": nodeID,
									"lang":    lang,
									"src":     src,
									"env":     "sandbox",
								},
							}

							loadResponse, _ := v.req("load_api_code", loadOps)

							// Check if the load request was successful
							if loadResponse == nil {
								return fmt.Errorf("failed to load API code: no response from server")
							}

							// Check if the request was processed successfully
							if requestProc, ok := loadResponse["request_proc"].(string); !ok || requestProc != "ok" {
								return fmt.Errorf("failed to load API code: request_proc not ok")
							}

							// Check each operation result
							if opsArray, ok := loadResponse["ops"].([]interface{}); ok {
								for _, op := range opsArray {
									if opMap, ok := op.(map[string]interface{}); ok {
										if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
											// Check for errors structure
											if errors, ok := opMap["errors"].(map[string]interface{}); ok {
												// Create a reverse mapping from obj_id to node ID

												// Build error message with mapped node IDs
												var errorMsgs []string
												for objID, errMsgs := range errors {
													// Map obj_id back to node ID
													nodeID := objID

													// Process error messages for this node
													if errArray, ok := errMsgs.([]interface{}); ok {
														for _, errMsg := range errArray {
															if msg, ok := errMsg.(string); ok {
																errorMsgs = append(errorMsgs, fmt.Sprintf("Error in the node %s: %s", nodeID, msg))
															}
														}
													}
												}

												if len(errorMsgs) > 0 {
													return fmt.Errorf("failed to load API code:\n%s", strings.Join(errorMsgs, "\n"))
												}
											}

											// Fallback to simple error message if no structured errors
											errorMsg := "unknown error"
											if msg, ok := opMap["description"].(string); ok {
												errorMsg = msg
											}
											return fmt.Errorf("failed to load API code: %s", errorMsg)
										}
									}
								}
							}

							// Second API call: Compile the code
							compileOps := []map[string]any{
								{
									"obj":     "api_code",
									"type":    "compile",
									"conv_id": v.ProcessID,
									"node_id": nodeID,
									"lang":    lang,
									"src":     src,
								},
							}

							compileResponse, _ := v.req("compile_api_code", compileOps)

							// Check if the compile request was successful
							if compileResponse == nil {
								return fmt.Errorf("failed to compile API code: no response from server")
							}

							// Check if the request was processed successfully
							if requestProc, ok := compileResponse["request_proc"].(string); !ok || requestProc != "ok" {
								return fmt.Errorf("failed to compile API code: request_proc not ok")
							}

							// Check each operation result
							if opsArray, ok := compileResponse["ops"].([]interface{}); ok {
								for _, op := range opsArray {
									if opMap, ok := op.(map[string]interface{}); ok {
										if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
											// Check for errors structure
											if errors, ok := opMap["errors"].(map[string]interface{}); ok {
												// Create a reverse mapping from obj_id to node ID

												// Build error message with mapped node IDs
												var errorMsgs []string
												for objID, errMsgs := range errors {
													// Map obj_id back to node ID
													nodeID := objID

													// Process error messages for this node
													if errArray, ok := errMsgs.([]interface{}); ok {
														for _, errMsg := range errArray {
															if msg, ok := errMsg.(string); ok {
																errorMsgs = append(errorMsgs, fmt.Sprintf("Node %s: %s", nodeID, msg))
															}
														}
													}
												}

												if len(errorMsgs) > 0 {
													return fmt.Errorf("failed to compile API code:\n%s", strings.Join(errorMsgs, "\n"))
												}
											}

											// Fallback to simple error message if no structured errors
											errorMsg := "unknown error"
											if msg, ok := opMap["description"].(string); ok {
												errorMsg = msg
											}
											return fmt.Errorf("failed to compile API code: %s", errorMsg)
										}
									}
								}
							}
						}

					}
				}
			}
		}
	}
	logger.Debug("API code loading and compilation completed")
	return nil
}

// ModifyNodes updates the nodes with their content after they've been created.
// It takes the enhanced processData with obj_id fields and updates the nodes on the server.
// Returns error if any operation fails.
func (v *Executor) ModifyNodes(nodes []any) error {

	// Create operations for each node
	var ops []map[string]any
	for _, node := range nodes {
		//if i > 1 {
		//	break
		//}
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip nodes without obj_id
		objID, ok := nodeMap["id"].(string)
		if !ok {
			continue
		}

		// Extract node properties
		title, _ := nodeMap["title"].(string)
		description, _ := nodeMap["description"].(string)
		objType, _ := nodeMap["obj_type"].(float64)

		// Extract position
		x, xOk := nodeMap["x"].(float64)
		y, yOk := nodeMap["y"].(float64)
		position := []float64{}
		if xOk && yOk {
			position = []float64{x, y}
		}

		// Extract extra
		extraStr, _ := nodeMap["extra"].(string)
		var extra map[string]interface{}
		if extraStr != "" {
			json.Unmarshal([]byte(extraStr), &extra)
		}

		// Extract options
		optionsStr, _ := nodeMap["options"].(string)
		var options map[string]interface{}
		if optionsStr != "" {
			json.Unmarshal([]byte(optionsStr), &options)
		}

		// Extract logics
		logics := []map[string]interface{}{}
		if condition, ok := nodeMap["condition"].(map[string]interface{}); ok {
			if logicsArray, ok := condition["logics"].([]interface{}); ok {
				isGo := false
				actionCount := 0
				for _, logic := range logicsArray {
					if logicMap, ok := logic.(map[string]interface{}); ok {
						logicType, _ := logicMap["type"].(string)
						if logicType == "go" {
							isGo = true
						}
						if logicType != "go" && logicType != "go_if_const" {
							actionCount++
						}
						logics = append(logics, logicMap)
					}
				}
				if len(logicsArray) > 0 && !isGo {
					return fmt.Errorf("error in the %s node, Each node in the condition.logics array must always have a logic with \"type\": \"go\" at the end. This is necessary in case none of the conditions are met, so the request is redirected by default to the specified node.", nodeMap["id"].(string))
				}
				if actionCount > 1 {
					return fmt.Errorf("error in the %s node (%s): condition.logics must not contain more than one action logic (types other than \"go\" and \"go_if_const\"). Found %d action logics.", objID, title, actionCount)
				}
			}
		}

		// Extract semaphors
		semaphors := []map[string]interface{}{}
		if condition, ok := nodeMap["condition"].(map[string]interface{}); ok {
			if semaphorsArray, ok := condition["semaphors"].([]interface{}); ok {
				for _, semaphor := range semaphorsArray {
					if semaphorMap, ok := semaphor.(map[string]interface{}); ok {
						semaphors = append(semaphors, semaphorMap)
					}
				}
			}
		}

		// Create operation for this node
		op := map[string]any{
			"type":        "modify",
			"obj":         "node",
			"obj_id":      objID,
			"conv_id":     v.ProcessID,
			"title":       title,
			"description": description,
			"obj_type":    objType,
			"version":     v.Version,
		}

		// Add logics and semaphors
		if len(logics) > 0 {
			op["logics"] = logics
		} else {
			op["logics"] = []map[string]interface{}{}
		}

		if len(semaphors) > 0 {
			op["semaphors"] = semaphors
		} else {
			op["semaphors"] = []map[string]interface{}{}
		}

		// Add position if available
		if len(position) == 2 {
			op["position"] = position
		}

		// Add extra if available
		if extra != nil {
			op["extra"] = extra
		}

		// Add options if available
		if options != nil {
			op["options"] = options
		}

		ops = append(ops, op)
	}

	// If no nodes were found, return an error
	if len(ops) == 0 {
		return fmt.Errorf("no valid nodes found to modify")
	}

	// Send the request to modify nodes
	response, _ := v.req("modify_nodes", ops)

	// Check if the request was successful
	if response == nil {
		return fmt.Errorf("failed to modify nodes: no response from server")
	}

	// Check if the request was processed successfully
	if requestProc, ok := response["request_proc"].(string); !ok || requestProc != "ok" {
		return fmt.Errorf("failed to modify nodes: request_proc not ok")
	}

	// Check each operation result
	if opsArray, ok := response["ops"].([]interface{}); ok {
		var errorMsgs []string

		for i, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					// Get node information if available

					nodeInfo := fmt.Sprintf("Operation %d", i+1)
					nodeInfo = ops[i]["obj_id"].(string)
					if objID, ok := opMap["obj_id"].(string); ok {
						// Try to map obj_id back to node ID
						nodeInfo = fmt.Sprintf("Node with obj_id %s", objID)
					}

					// Check for description field (new format)
					if description, ok := opMap["description"].(string); ok {
						errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", nodeInfo, description))
						continue
					}

					// Check for errors structure (old format)
					if errors, ok := opMap["errors"].(map[string]interface{}); ok {
						// Build error message with mapped node IDs
						for objID, errMsgs := range errors {
							// Map obj_id back to node ID
							nodeID := objID

							// Process error messages for this node
							if errArray, ok := errMsgs.([]interface{}); ok {
								for _, errMsg := range errArray {
									if msg, ok := errMsg.(string); ok {
										errorMsgs = append(errorMsgs, fmt.Sprintf("Node %s: %s", nodeID, msg))
									}
								}
							}
						}
						continue
					}

					// Fallback to simple error message if no structured errors
					errorMsg := "unknown error"
					if msg, ok := opMap["message"].(string); ok {
						errorMsg = msg
					}
					errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", nodeInfo, errorMsg))
				}
			}
		}

		if len(errorMsgs) > 0 {
			return fmt.Errorf("%s", strings.Join(errorMsgs, "\n"))
		}

	}
	logger.Debug("Nodes were modified with their content")
	return nil
}

// Commit confirms and finalizes changes to a process
// Returns the response from the server
func (v *Executor) Commit() map[string]interface{} {

	// Create the operation for committing changes
	ops := []map[string]any{
		{
			"type":    "confirm",
			"obj":     "commit",
			"conv_id": v.ProcessID,
			"version": v.Version,
		},
	}

	if v.Debug {
		logger.Debug("Sending commit request")
	}

	// Send the request to commit changes
	response, _ := v.req("commit_process", ops)

	if response == nil {
		logger.Error("Failed to commit changes: no response from server")
	} else if v.Debug {
		if requestProc, ok := response["request_proc"].(string); ok {
			logger.Debug("Commit response received, request_proc=%s", requestProc)
		}
	}
	logger.Debug("Changes committed, processID=%d", v.ProcessID)
	return response
}

func (v *Executor) DeleteVersion(ver int) {
	//{"ops":[{"type":"delete","obj":"commits","conv_id":1662238,"company_id":"28eecad5-ebd4-4621-ae21-af4f568bcd94","version":1747658113}]}
	// Create the operation for committing changes
	ops := []map[string]any{
		{
			"type":       "delete",
			"obj":        "commits",
			"company_id": workspaceID,
			"conv_id":    v.ProcessID,
			"version":    ver,
		},
	}

	// Send the request to commit changes
	response, _ := v.req("delete_version", ops)

	if response == nil {
		logger.Error("Failed to commit changes: no response from server")
	} else if v.Debug {
		if requestProc, ok := response["request_proc"].(string); ok {
			logger.Debug("Commit response received, request_proc=%s", requestProc)
		}
	}
	logger.Debug("Delete Changes, processID=%d", v.ProcessID)
	return
}

// Share grants access to a process for a specific user
// userID is the ID of the user to grant access to
// convID is the ID of the process to grant access to
func (v *Executor) Share(userID, convID int) map[string]interface{} {
	//logger.Info("Sharing process access, processID=%d, userID=%d", convID, userID)

	// Create the operation for sharing access
	ops := []map[string]any{
		{
			"type":              "link",
			"obj":               "conv",
			"obj_id":            convID,
			"obj_to":            "user",
			"obj_to_id":         userID,
			"is_need_to_notify": true,
			"privs": []map[string]any{
				{
					"type":     "create",
					"list_obj": []string{"all"},
				},
				{
					"type":     "modify",
					"list_obj": []string{"all"},
				},
				{
					"type":     "delete",
					"list_obj": []string{"all"},
				},
				{
					"type":     "view",
					"list_obj": []string{"all"},
				},
			},
		},
	}

	if v.Debug {
		logger.Debug("Sending share request, privileges=all")
	}

	// Send the request to share access
	response, _ := v.req("share_process", ops)

	if response == nil {
		logger.Error("Failed to share process: no response from server")
	} else if v.Debug {
		if requestProc, ok := response["request_proc"].(string); ok {
			logger.Debug("Share response received, request_proc=%s", requestProc)
		}
	}

	return response
}

// req sends a request to the Corezoid API with authentication
func (v *Executor) req(method string, ops []map[string]any) (map[string]interface{}, error) {
	// Personal-workspace accounts have no companyID. The callers in this file
	// unconditionally inject `"company_id": workspaceID` into every op, so when
	// workspaceID is empty the payload carries `"company_id": ""` and Corezoid
	// rejects every request with `Value is not valid / company_id`. Drop the
	// empty placeholder (and its mirrors) so the request matches what the API
	// accepts for personal accounts. No-op for normal company workspaces.
	if strings.TrimSpace(workspaceID) == "" {
		for _, op := range ops {
			for _, k := range []string{"company_id", "from_company_id", "to_company_id"} {
				if s, ok := op[k].(string); ok && s == "" {
					delete(op, k)
				}
			}
		}
	}

	// Create the payload
	payload := map[string]any{"ops": ops}
	payloadJSON, _ := json.Marshal(payload)

	// Log request details based on debug level
	if v.Debug {
		// For debug mode, pretty print the JSON for better readability
		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, payloadJSON, "", "  ")
		logger.Debug("API Request, method=%s, payload=%s", method, prettyJSON.String())
	}
	// Generate timestamp and signature

	// Create the authenticated URL
	path := "json"
	if method == "export_process" {
		path = "download"
	}
	authURL := fmt.Sprintf("%s/api/2/%s", v.APIUrl, path)

	if v.Debug {
		logger.Debug("Request URL, url=%s", authURL)
	}
	//fmt.Println(authURL)
	// Create the request
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		logger.Error("Error creating request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Simulator %s", v.Token))
	logger.Debug("Header: %v", req.Header)

	// Create HTTP client with insecure TLS configuration to skip certificate verification
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error making request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading response: %v", err)
		return nil, err
	}

	// Log response details based on debug level
	if v.Debug {
		// For debug mode, pretty print the JSON response if possible
		var prettyJSON bytes.Buffer
		if json.Indent(&prettyJSON, body, "", "  ") == nil {
			logger.Debug("API Response, method=%s, status=%s, body=%s", method, resp.Status, prettyJSON.String())
		} else {
			// If can't pretty print, just log the raw response
			logger.Debug("API Response, method=%s, status=%s, body=%s", method, resp.Status, string(body))
		}
	}

	// Parse the response body into a map
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		logger.Error("Error parsing response: %v", err)
		return nil, err
	}
	err = v.checkError(response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (v *Executor) checkError(rsp map[string]interface{}) error {
	// Check if the compile request was successful
	if rsp == nil {
		return fmt.Errorf("failed to compile API code: no response from server")
	}

	// Check if the request was processed successfully
	if requestProc, ok := rsp["request_proc"].(string); !ok || requestProc != "ok" {
		return fmt.Errorf("failed to compile API code: request_proc not ok")
	}

	// Check each operation result
	if opsArray, ok := rsp["ops"].([]interface{}); ok {
		for _, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					// Check for errors structure
					if errors, ok := opMap["errors"].(map[string]interface{}); ok {
						// Create a reverse mapping from obj_id to node ID

						// Build error message with mapped node IDs
						var errorMsgs []string
						for objID, errMsgs := range errors {
							// Map obj_id back to node ID
							nodeID := objID

							// Process error messages for this node
							if errArray, ok := errMsgs.([]interface{}); ok {
								for _, errMsg := range errArray {
									if msg, ok := errMsg.(string); ok {
										errorMsgs = append(errorMsgs, fmt.Sprintf("Node %s: %s", nodeID, msg))
									}
								}
							}
						}

						if len(errorMsgs) > 0 {
							return fmt.Errorf("failed to compile API code:\n%s", strings.Join(errorMsgs, "\n"))
						}
					}

					// Fallback to simple error message if no structured errors
					errorMsg := "unknown error"
					if msg, ok := opMap["description"].(string); ok {
						errorMsg = msg
					}
					return fmt.Errorf("failed to execute: %s", errorMsg)
				}
			}
		}
	}
	return nil
}

func (v *Executor) DeleteNotUsedNodes(oldNodes []any, newNodes []any) []any {
	for _, oldNode := range oldNodes {
		if int(oldNode.(map[string]interface{})["obj_type"].(float64)) == 1 {
			continue
		}

		id := oldNode.(map[string]interface{})["obj_id"].(string)
		foundID := -1
		for newNodeID, newNode := range newNodes {
			if newNode.(map[string]interface{})["id"].(string) == id {
				foundID = newNodeID
				break
			}
		}
		if foundID == -1 {

			logger.Debug("Deleting node %s", id)
			v.DeleteNode(id)
		} else {
			nn := newNodes[foundID].(map[string]interface{})
			nn["existed"] = true
			newNodes[foundID] = nn
		}
	}
	return newNodes

}

func (v *Executor) DeleteNode(id string) {
	//	{"ops":[{"type":"delete","obj":"node","company_id":"28eecad5-ebd4-4621-ae21-af4f568bcd94","obj_id":"682aee5c82ba963719debc2d","conv_id":1661878,"version":1747645682}]}
	ops := []map[string]any{
		{
			"type":       "delete",
			"obj":        "node",
			"company_id": workspaceID,
			"obj_id":     id,
			"conv_id":    v.ProcessID,
			"version":    v.Version,
		},
	}

	if v.Debug {
		logger.Debug("Sending delete node request")
	}

	// Send the request to delete node
	response, _ := v.req("delete_node", ops)

	if response == nil {
		logger.Error("Failed to delete node: no response from server")
	} else if v.Debug {
		if requestProc, ok := response["request_proc"].(string); ok {
			logger.Debug("Delete node response received, request_proc=%s", requestProc)
		}
	}
}

func (v *Executor) BeforeValidation(jsonContent string, taskData map[string]interface{}) error {
	var processData map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &processData)
	if err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}
	params, _ := processData["params"].([]interface{})
	for name := range taskData {
		found := false
		for _, param := range params {
			paramMap, ok := param.(map[string]interface{})
			if !ok {
				continue
			}
			paramName, _ := paramMap["name"].(string)
			if paramName == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("task data key '%s' is not specified in the 'params' field of the process", name)
		}
	}
	nodes, err := getNodes(processData)
	if err != nil {
		return fmt.Errorf("error reading nodes: %v", err)
	}

	// Check for node overlapping
	//type nodePosition struct {
	//	x, y    float64
	//	objType int
	//	id      string // Add ID to track which node is which
	//}
	//positions := make([]nodePosition, 0, len(nodes))
	//
	//for _, node := range nodes {
	//	nodeMap, ok := node.(map[string]interface{})
	//	if !ok {
	//		continue
	//	}
	//
	//	// Extract position, type and ID
	//	x, xOk := nodeMap["x"].(float64)
	//	y, yOk := nodeMap["y"].(float64)
	//	objType := int(nodeMap["obj_type"].(float64))
	//	id, _ := nodeMap["id"].(string)
	//
	//	if xOk && yOk {
	//		positions = append(positions, nodePosition{x, y, objType, id})
	//	}
	//}
	//
	//// Check for overlapping nodes
	//for i := 0; i < len(positions); i++ {
	//	for j := i + 1; j < len(positions); j++ {
	//		// Calculate distance between centers based on obj_type
	//		dx := positions[i].x - positions[j].x
	//		dy := positions[i].y - positions[j].y
	//
	//		// Adjust coordinates based on pivot points
	//		// For START/END nodes (obj_type 1 or 2), pivot is at center
	//		// For other nodes, pivot is at top-left corner
	//		if positions[i].objType != 1 && positions[i].objType != 2 {
	//			// Add half width (100px) to get to center
	//			dx += 100
	//			// Add half height (75px) to get to center
	//			dy += 75
	//		}
	//
	//		if positions[j].objType != 1 && positions[j].objType != 2 {
	//			// Add half width (100px) to get to center
	//			dx -= 100
	//			// Add half height (75px) to get to center
	//			dy -= 75
	//		}
	//
	//		distance := math.Sqrt(dx*dx + dy*dy)
	//
	//		// Calculate minimum required distance based on node types
	//		minDistance := 100.0 // Default for two regular nodes (half width + half width)
	//
	//		if positions[i].objType == 1 || positions[i].objType == 2 {
	//			if positions[j].objType == 1 || positions[j].objType == 2 {
	//				// Both are START/END nodes (radius + radius)
	//				minDistance = 50.0
	//			} else {
	//				// One START/END, one regular (radius + half width)
	//				minDistance = 100.0
	//			}
	//		}
	//
	//		if distance < minDistance {
	//			return fmt.Errorf("nodes with IDs '%s' and '%s' at positions (%.1f, %.1f) and (%.1f, %.1f) are too close to each other (distance: %.1f px, minimum required: %.1f px)",
	//				positions[i].id, positions[j].id, positions[i].x, positions[i].y, positions[j].x, positions[j].y, distance, minDistance)
	//		}
	//	}
	//}

	for _, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}
		if condition, ok := nodeMap["condition"].(map[string]interface{}); ok {
			if logicsArray, ok := condition["logics"].([]interface{}); ok {
				actionCount := 0
				for _, logic := range logicsArray {
					if logicMap, ok := logic.(map[string]interface{}); ok {
						type1, _ := logicMap["type"].(string)
						if type1 == "api_copy" || type1 == "api_rpc" {
							if convID, ok := logicMap["conv_id"].(float64); ok && int(convID) == v.ProcessID {
								return fmt.Errorf("node with type %s and conv_id %d must not have the same process", type1, int(convID))
							}
						}
						if type1 != "go" && type1 != "go_if_const" {
							actionCount++
						}
					}
				}
				if actionCount > 1 {
					nodeID, _ := nodeMap["id"].(string)
					nodeTitle, _ := nodeMap["title"].(string)
					return fmt.Errorf("node '%s' (%s): condition.logics must not contain more than one action logic (types other than \"go\" and \"go_if_const\"). Found %d action logics.", nodeID, nodeTitle, actionCount)
				}
			}
		}
	}

	// Check that alias references in conv_id actually exist.
	// Collect all unique alias names first, then do a single list call.
	type aliasRef struct {
		name     string
		nodeType string
	}
	var aliasRefs []aliasRef
	seenAlias := make(map[string]bool)
	for _, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}
		if condition, ok := nodeMap["condition"].(map[string]interface{}); ok {
			if logicsArray, ok := condition["logics"].([]interface{}); ok {
				for _, logic := range logicsArray {
					if logicMap, ok := logic.(map[string]interface{}); ok {
						type1, _ := logicMap["type"].(string)
						if type1 != "api_copy" && type1 != "api_rpc" {
							continue
						}
						convID, ok := logicMap["conv_id"].(string)
						if !ok || !strings.HasPrefix(convID, "@") {
							continue
						}
						aliasName := convID[1:] // strip leading @
						// skip dynamic references like @{{variable}} — resolved at runtime
						if strings.Contains(aliasName, "{{") {
							continue
						}
						if !seenAlias[aliasName] {
							seenAlias[aliasName] = true
							aliasRefs = append(aliasRefs, aliasRef{name: aliasName, nodeType: type1})
						}
					}
				}
			}
		}
	}
	if len(aliasRefs) > 0 && stageID != 0 {
		existingAliases, err := v.listAliasesByStage(stageID)
		if err != nil {
			return fmt.Errorf("alias validation failed: %v", err)
		}
		for _, ref := range aliasRefs {
			if _, exists := existingAliases[ref.name]; !exists {
				return fmt.Errorf("node with type %s references alias '@%s' which does not exist", ref.nodeType, ref.name)
			}
		}
	}

	// Check that all {{env_var[@name]}} references point to existing env variables.
	// Requires COREZOID_STAGE_ID to be set; skipped silently if not available.
	if stageID != 0 {
		matches := reEnvVar.FindAllStringSubmatch(jsonContent, -1)
		seen := make(map[string]bool)
		var uniqueVarNames []string
		for _, m := range matches {
			varName := m[1]
			if !seen[varName] {
				seen[varName] = true
				uniqueVarNames = append(uniqueVarNames, varName)
			}
		}
		if len(uniqueVarNames) > 0 {
			existingVars, err := v.listEnvVarsByStage(stageID)
			if err != nil {
				return fmt.Errorf("env var validation failed: %v", err)
			}
			for _, varName := range uniqueVarNames {
				if _, exists := existingVars[varName]; !exists {
					return fmt.Errorf("env variable '@%s' referenced in process does not exist", varName)
				}
			}
		}
	}

	return nil
}

// resolveFolderPathFromAPI builds a relative path from the stage root down to folderID
// by walking up the folder hierarchy via ShowFolder API calls.
// Returns "" when folderID is the stage root itself (process lives at the top level).
// The stage root is identified by: folderID == stageID (global) OR folder.ObjType == 3.
func (v *Executor) resolveFolderPathFromAPI(folderID int) (string, error) {
	const maxDepth = 20
	type segment struct {
		id    int
		title string
	}
	var segments []segment
	currentID := folderID
	for i := 0; i < maxDepth; i++ {
		if stageID != 0 && currentID == stageID {
			break
		}
		info, err := v.ShowFolder(currentID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve folder path at id %d: %w", currentID, err)
		}
		if info.ObjType == 3 || (stageID != 0 && info.ObjID == stageID) {
			break
		}
		safeName := strings.ReplaceAll(info.Title, " ", "_")
		segments = append(segments, segment{id: info.ObjID, title: safeName})
		if info.ParentObjID == 0 || info.ParentObjID == currentID {
			break
		}
		currentID = info.ParentObjID
	}
	// collected bottom-up; reverse to get top-down
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}
	parts := make([]string, len(segments))
	for i, s := range segments {
		parts[i] = fmt.Sprintf("%d_%s", s.id, s.title)
	}
	return strings.Join(parts, string(os.PathSeparator)), nil
}

// CreateVariable creates a new environment variable using the provided parameters
func (v *Executor) CreateVariable(rootFolderIDBin, name, description, value string) error {
	rootFolderID, _ := strconv.Atoi(rootFolderIDBin)
	ops := []map[string]any{
		{
			"obj":          "env_var",
			"data_type":    "raw",
			"title":        description,
			"short_name":   name,
			"company_id":   workspaceID,
			"stage_id":     rootFolderID,
			"project_id":   v.GetProjectIDByStageID(rootFolderID),
			"env_var_type": "visible",
			"type":         "create",
			"obj_type":     0,
			"status":       "active",
			"scopes":       []map[string]string{{"type": "*", "fields": "*"}},
			"value":        value,
		},
	}

	if v.Debug {
		logger.Debug("Sending create variable request")
	}

	response, _ := v.req("create-variable", ops)

	if response["request_proc"] != "ok" {
		return fmt.Errorf("failed to create variable: %v", response)
	}

	// Check if the operation was successful
	if opsArray, ok := response["ops"].([]interface{}); ok {
		for _, op := range opsArray {
			if opMap, ok := op.(map[string]interface{}); ok {
				if proc, ok := opMap["proc"].(string); !ok || proc != "ok" {
					errorMsg := "unknown error"
					if msg, ok := opMap["description"].(string); ok {
						errorMsg = msg
					}
					return fmt.Errorf("failed to create variable: %s", errorMsg)
				}
			}
		}
	}

	// Update local variables.json file
	err := v.updateVariablesFile(name, description, value)
	if err != nil {
		logger.Error("Failed to update local variables file: %v", err)
		// Don't return error here as the API call was successful
	}

	return nil
}

// updateVariablesFile updates the local .processes/variables.json file
func (v *Executor) updateVariablesFile(name, description, value string) error {
	// Create .processes directory if it doesn't exist
	err := os.MkdirAll(".processes", 0755)
	if err != nil {
		return fmt.Errorf("failed to create .processes directory: %v", err)
	}

	variablesPath := ".processes/variables.json"

	// Read existing variables or create empty slice
	var variables []map[string]string

	if data, err := ioutil.ReadFile(variablesPath); err == nil {
		// File exists, parse it
		err = json.Unmarshal(data, &variables)
		if err != nil {
			return fmt.Errorf("failed to parse existing variables.json: %v", err)
		}
	}

	// Check if variable already exists and update it, otherwise add new one
	found := false
	for i, variable := range variables {
		if variable["name"] == name {
			variables[i] = map[string]string{
				"name":        name,
				"description": description,
				"value":       value,
			}
			found = true
			break
		}
	}

	if !found {
		// Add new variable
		variables = append(variables, map[string]string{
			"name":        name,
			"description": description,
			"value":       value,
		})
	}

	// Write updated variables to file
	data, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %v", err)
	}

	err = ioutil.WriteFile(variablesPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write variables.json: %v", err)
	}

	if v.Debug {
		logger.Debug("Updated local variables file: %s", variablesPath)
	}

	return nil
}

// FolderInfo holds the result of a "show folder" API call.
type FolderInfo struct {
	ObjID         int    `json:"obj_id"`
	Title         string `json:"title"`
	ObjType       int    `json:"obj_type"`
	ParentObjID   int    `json:"parent_obj_id"`
	ParentObjType string `json:"parent_obj_type"`
}

// ShowFolder calls the "show folder" API for the given folderID and returns FolderInfo.
// obj_type values: 0 — normal, 1 — root user, 2 — project, 3 — stage.
func (v *Executor) ShowFolder(folderID int) (*FolderInfo, error) {
	ops := []map[string]any{
		{
			"type":   "show",
			"obj":    "folder",
			"obj_id": folderID,
		},
	}
	response, err := v.req("json", ops)
	//fmt.Println("response", response)
	if err != nil {
		return nil, fmt.Errorf("ShowFolder request failed: %w", err)
	}
	if response["request_proc"] != "ok" {
		return nil, fmt.Errorf("ShowFolder: unexpected request_proc: %v", response["request_proc"])
	}
	opsRaw, ok := response["ops"].([]any)
	if !ok || len(opsRaw) == 0 {
		return nil, fmt.Errorf("ShowFolder: empty ops in response")
	}
	op, ok := opsRaw[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ShowFolder: unexpected ops format")
	}
	if op["proc"] != "ok" {
		return nil, fmt.Errorf("ShowFolder: proc error: %v", op["proc"])
	}
	info := &FolderInfo{}
	if val, ok := op["obj_id"].(float64); ok {
		info.ObjID = int(val)
	}
	if val, ok := op["title"].(string); ok {
		info.Title = val
	}
	if val, ok := op["obj_type"].(float64); ok {
		info.ObjType = int(val)
	}
	if val, ok := op["parent_obj_id"].(float64); ok {
		info.ParentObjID = int(val)
	}
	if val, ok := op["parent_obj_type"].(string); ok {
		info.ParentObjType = val
	}
	return info, nil
}

// CreateFolder creates a new folder under parentFolderID and returns the new folder's obj_id.
func (v *Executor) CreateFolder(parentFolderID int, title, desc string) (int, error) {
	ops := []map[string]any{
		{
			"title":       title,
			"description": desc,
			"folder_id":   parentFolderID,
			"company_id":  workspaceID,
			"obj":         "folder",
			"type":        "create",
		},
	}
	response, err := v.req("json", ops)
	if err != nil {
		return 0, fmt.Errorf("CreateFolder request failed: %w", err)
	}
	if response["request_proc"] != "ok" {
		return 0, fmt.Errorf("CreateFolder: unexpected request_proc: %v", response["request_proc"])
	}
	opsRaw, ok := response["ops"].([]any)
	if !ok || len(opsRaw) == 0 {
		return 0, fmt.Errorf("CreateFolder: empty ops in response")
	}
	op, ok := opsRaw[0].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("CreateFolder: unexpected ops format")
	}
	if op["proc"] != "ok" {
		return 0, fmt.Errorf("CreateFolder: proc error: %v", op["proc"])
	}
	objID, ok := op["obj_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("CreateFolder: obj_id not found in response")
	}
	return int(objID), nil
}

// CreateAlias creates a new alias for a process and returns the alias ID.
// shortName must match ^[a-z0-9-]*$ and be unique within the stage.
// procID is the process (conv) to link the alias to.
// stageID is taken from COREZOID_STAGE_ID.
func (v *Executor) CreateAlias(shortName string, procID, stageID int) (int, error) {
	projectID := v.GetProjectIDByStageID(stageID)

	ops := []map[string]any{
		{
			"type":        "create",
			"obj":         "alias",
			"title":       shortName,
			"short_name":  shortName,
			"company_id":  workspaceID,
			"stage_id":    stageID,
			"project_id":  projectID,
			"obj_to_id":   procID,
			"obj_to_type": "conv",
		},
	}

	if v.Debug {
		logger.Debug("Sending create alias request")
	}

	response, err := v.req("create_alias", ops)
	if err != nil {
		return 0, fmt.Errorf("create alias request failed: %w", err)
	}

	if opsArray, ok := response["ops"].([]interface{}); ok && len(opsArray) > 0 {
		if firstOp, ok := opsArray[0].(map[string]interface{}); ok {
			if proc, _ := firstOp["proc"].(string); proc != "ok" {
				desc, _ := firstOp["description"].(string)
				if desc == "" {
					desc = "unknown error"
				}
				return 0, fmt.Errorf("create alias failed: %s", desc)
			}
			if objID, ok := firstOp["obj_id"].(float64); ok {
				return int(objID), nil
			}
		}
	}

	return 0, fmt.Errorf("create alias: unexpected response format")
}

// listAliasesByStage returns a map of short_name -> obj_id for all aliases in the given stage.
func (v *Executor) listAliasesByStage(stage int) (map[string]int, error) {
	projectID := v.GetProjectIDByStageID(stage)
	ops := []map[string]any{
		{
			"type":       "list",
			"obj":        "aliases",
			"company_id": workspaceID,
			"stage_id":   stage,
			"project_id": projectID,
		},
	}
	response, err := v.req("json", ops)
	if err != nil {
		return nil, fmt.Errorf("failed to list aliases: %v", err)
	}
	opsRaw, ok := response["ops"].([]any)
	if !ok || len(opsRaw) == 0 {
		return nil, fmt.Errorf("empty response from alias list")
	}
	op, ok := opsRaw[0].(map[string]any)
	if !ok || op["proc"] != "ok" {
		return nil, fmt.Errorf("alias list failed")
	}
	result := make(map[string]int)
	list, _ := op["list"].([]any)
	for _, item := range list {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sn, _ := itemMap["short_name"].(string)
		if sn == "" {
			continue
		}
		if objID, ok := itemMap["obj_id"].(float64); ok {
			result[sn] = int(objID)
		}
	}
	return result, nil
}

// GetAliasByShortName checks if an alias with the given short_name exists.
// Returns the alias obj_id on success, or an error if not found or the request fails.
func (v *Executor) GetAliasByShortName(shortName string) (int, error) {
	aliases, err := v.listAliasesByStage(stageID)
	if err != nil {
		return 0, fmt.Errorf("alias '%s' does not exist: %v", shortName, err)
	}
	if objID, ok := aliases[shortName]; ok {
		return objID, nil
	}
	return 0, fmt.Errorf("alias '%s' does not exist", shortName)
}

// listEnvVarsByStage returns a map of short_name -> obj_id for all env vars in the given stage.
func (v *Executor) listEnvVarsByStage(stage int) (map[string]int, error) {
	projectID := v.GetProjectIDByStageID(stage)
	ops := []map[string]any{
		{
			"type":       "list",
			"obj":        "env_var",
			"company_id": workspaceID,
			"stage_id":   stage,
			"project_id": projectID,
		},
	}
	response, err := v.req("json", ops)
	if err != nil {
		return nil, fmt.Errorf("failed to list env variables: %v", err)
	}
	opsRaw, ok := response["ops"].([]any)
	if !ok || len(opsRaw) == 0 {
		return nil, fmt.Errorf("empty response from env var list")
	}
	op, ok := opsRaw[0].(map[string]any)
	if !ok || op["proc"] != "ok" {
		return nil, fmt.Errorf("env var list failed")
	}
	result := make(map[string]int)
	list, _ := op["list"].([]any)
	for _, item := range list {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sn, _ := itemMap["short_name"].(string)
		if sn == "" {
			continue
		}
		if objID, ok := itemMap["obj_id"].(float64); ok {
			result[sn] = int(objID)
		}
	}
	return result, nil
}

// GetEnvVarByShortName checks if an environment variable with the given short_name exists.
// Returns the env_var obj_id on success, or an error if not found.
func (v *Executor) GetEnvVarByShortName(shortName string) (int, error) {
	vars, err := v.listEnvVarsByStage(stageID)
	if err != nil {
		return 0, fmt.Errorf("env variable '@%s' does not exist: %v", shortName, err)
	}
	if objID, ok := vars[shortName]; ok {
		return objID, nil
	}
	return 0, fmt.Errorf("env variable '@%s' does not exist", shortName)
}

// GetProjectIDByStageID resolves the project ID (obj_type == 2) for a given stage/folder ID
// by walking up the folder hierarchy.
func (v *Executor) GetProjectIDByStageID(folderID int) int {
	const maxDepth = 20
	currentID := folderID
	for i := 0; i < maxDepth; i++ {
		info, err := v.ShowFolder(currentID)
		if err != nil {
			logger.Error("GetProjectIDByStageID: error showing folder %d: %v", currentID, err)
		}
		// obj_type 2 means this folder IS the project
		if info.ObjID == folderID {
			return info.ParentObjID
		}
		currentID = info.ParentObjID
	}
	logger.Error("GetProjectIDByStageID: cannot find folder %d", folderID)
	return 0
}
