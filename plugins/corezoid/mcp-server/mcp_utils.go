package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// confineToWorkdir is a lexical guard against path-traversal in LLM-supplied
// path arguments. It rejects absolute paths and any relative path whose
// cleaned form starts with ".." (i.e. escapes the working directory).
//
// Threat model: a prompt-injected LLM might call lint-process with
// process_path="../../etc/passwd" or pull-folder writing to ~/.ssh/. Handlers
// run with the user's full file permissions, so anything the path argument
// says, the OS will do. This check is defence-in-depth: it blocks the
// path-traversal escape without restricting legitimate relative paths inside
// the workspace.
//
// Design note: absolute paths are rejected outright rather than allowed-if-
// under-cwd. The "under-cwd" form invites subtle symlink edge cases (on
// macOS in particular, /var/folders/... and /private/var/folders/... are the
// same directory via symlink, breaking lexical Rel comparison) and the MCP
// server always runs with cwd set to the project root via COREZOID_WORK_DIR,
// so legitimate paths are naturally relative.
func confineToWorkdir(p string) (string, error) {
	if p == "" {
		return p, nil
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute paths are not allowed (received %q); pass a path relative to the project root", p)
	}
	clean := filepath.Clean(p)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes working directory", p)
	}
	return p, nil
}

// resolveFolderIDFromDir looks for a file matching <id>_<name>.folder.json or
// <id>_<name>.stage.json in the given directory and returns the numeric id.
func resolveFolderIDFromDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory '%s': %v", dir, err)
	}
	reFolderFile := regexp.MustCompile(`^(\d+)_.*\.(folder|stage)\.json$`)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := reFolderFile.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		id, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		return id, nil
	}
	return 0, fmt.Errorf("no <id>_<name>.folder.json file found in '%s'; cannot determine folder ID", dir)
}

// intArg extracts an integer argument from args map.
func intArg(args map[string]interface{}, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing required argument: %s", key)
	}
	switch val := v.(type) {
	case float64:
		return int(val), nil
	case int:
		return val, nil
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("argument %s must be an integer, got: %s", key, val)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("argument %s has unexpected type %T", key, v)
	}
}

// strArg extracts a string argument from args map.
func strArg(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string, got %T", key, v)
	}
	return s, nil
}

// optStrArg returns the string value of an optional argument, or "" if absent.
func optStrArg(args map[string]interface{}, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// resolveProcessPath resolves an optional process_path argument.
// If the argument is empty, it searches the current directory for a single
// .conv.json file and returns its path. The supplied path is confined to the
// workspace tree (see confineToWorkdir) so a prompt-injected tool call can't
// reach files outside the project root.
func resolveProcessPath(args map[string]interface{}, key string) (string, error) {
	p := optStrArg(args, key)
	if p != "" {
		safe, err := confineToWorkdir(p)
		if err != nil {
			return "", err
		}
		return safe, nil
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("cannot read current directory: %v", err)
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".conv.json") {
			matches = append(matches, e.Name())
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple .conv.json files found in current directory — pass process_path explicitly: %v", matches)
	}
	return "", fmt.Errorf("no .conv.json file found in current directory and process_path was not provided")
}

// resolveDirPath returns the path argument confined to the workspace tree,
// or "." if the argument is absent. Returns "." and logs a warning if the
// supplied path escapes the workspace — directory handlers (create-process,
// create-folder) can't surface a typed error through resolveDirPath's
// signature, so the caller's subsequent resolveFolderIDFromDir(".") will
// fail naturally if cwd has no folder marker, which is the same error
// the user would see for any unrecognised directory.
func resolveDirPath(args map[string]interface{}, key string) string {
	p := optStrArg(args, key)
	if p == "" {
		return "."
	}
	safe, err := confineToWorkdir(p)
	if err != nil {
		logger.Warn("resolveDirPath: rejected path %q: %v — falling back to cwd", p, err)
		return "."
	}
	return safe
}
