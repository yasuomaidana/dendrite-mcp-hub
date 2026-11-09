package jetbrains

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yasuomaidana/dendrite-mcp-hub/internal/fsutil"
)

// statConfig reports whether the config file exists; a directory in its place is an error.
func statConfig(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("%s is a directory", path)
	}
	return true, nil
}

// loadTopLevel reads the config into a raw top-level map; an empty file is an empty object.
func loadTopLevel(path string) (map[string]json.RawMessage, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	topLevel := make(map[string]json.RawMessage)
	if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &topLevel); err != nil {
			return nil, fmt.Errorf("invalid json in %s: %w", path, err)
		}
	}
	if topLevel == nil {
		topLevel = make(map[string]json.RawMessage)
	}
	return topLevel, nil
}

// decodeServers extracts mcpServers from topLevel; absent, empty or null yields an empty map.
func decodeServers(topLevel map[string]json.RawMessage, path string) (map[string]json.RawMessage, error) {
	servers := make(map[string]json.RawMessage)
	raw := strings.TrimSpace(string(topLevel["mcpServers"]))
	if raw == "" || raw == "null" {
		return servers, nil
	}
	if err := json.Unmarshal([]byte(raw), &servers); err != nil {
		return nil, fmt.Errorf("invalid mcpServers in %s: %w", path, err)
	}
	return servers, nil
}

// UpsertMCPServer writes entry under mcpServers[name] in the JSON file at path, preserving every
// other top-level key and every other server entry byte-for-byte semantically (use map[string]json.RawMessage
// at both levels). Creates parent dirs and the file if missing. Empty/whitespace file = empty object.
// Invalid JSON -> error (do NOT overwrite). Atomic write via internal/fsutil.WriteFileAtomic. Output indented 2 spaces + trailing newline.
// Returns created=true when the file did not exist before.
func UpsertMCPServer(path, name string, entry MCPServerEntry) (created bool, err error) {
	if name == "" {
		return false, fmt.Errorf("server name cannot be empty")
	}

	exists, err := statConfig(path)
	if err != nil {
		return false, err
	}

	topLevel := make(map[string]json.RawMessage)
	if exists {
		if topLevel, err = loadTopLevel(path); err != nil {
			return false, err
		}
	}

	servers, err := decodeServers(topLevel, path)
	if err != nil {
		return false, err
	}
	entryBytes, _ := json.Marshal(entry) // plain strings, lists and maps of strings: cannot fail
	servers[name] = json.RawMessage(entryBytes)

	if err := writeMCPServers(path, topLevel, servers); err != nil {
		return false, err
	}

	return !exists, nil
}

// writeMCPServers stores servers under mcpServers in topLevel and writes the file atomically.
func writeMCPServers(path string, topLevel map[string]json.RawMessage, servers map[string]json.RawMessage) error {
	serversBytes, _ := json.Marshal(servers) // values are raw JSON that was decoded or marshaled above: cannot fail
	topLevel["mcpServers"] = json.RawMessage(serversBytes)

	out, _ := json.MarshalIndent(topLevel, "", "  ") // same raw JSON values: cannot fail
	out = append(out, '\n')

	return fsutil.WriteFileAtomic(path, out)
}

// readFile reads the config; tests replace it to simulate the file vanishing after the stat.
var readFile = os.ReadFile

// RemoveMCPServer deletes mcpServers[name]; missing file or name is not an error; returns whether something was removed.
func RemoveMCPServer(path, name string) (bool, error) {
	if name == "" {
		return false, nil
	}

	exists, err := statConfig(path)
	if err != nil || !exists {
		return false, err
	}

	topLevel, err := loadTopLevel(path)
	if err != nil {
		if os.IsNotExist(err) { // the file vanished after the stat
			return false, nil
		}
		return false, err
	}

	servers, err := decodeServers(topLevel, path)
	if err != nil {
		return false, err
	}
	if _, ok := servers[name]; !ok {
		return false, nil
	}

	delete(servers, name)
	if err := writeMCPServers(path, topLevel, servers); err != nil {
		return false, err
	}

	return true, nil
}
