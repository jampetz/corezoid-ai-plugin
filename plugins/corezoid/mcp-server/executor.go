package main

import (
	"regexp"
	"time"
)

var reEnvVar = regexp.MustCompile(`\{\{env_var\[@([a-z0-9-]+)]}}`)

type Executor struct {
	NodeResponses []map[string]interface{}
	ProcessID     int
	APILogin      string
	Token         string
	APISecret     string
	APIUrl        string
	NodeIDMap     map[string]NodeInfo
	Debug         bool
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
