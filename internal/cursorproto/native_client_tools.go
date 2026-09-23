package cursorproto

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
)

// DecodeNativeClientToolCall exposes a native Cursor operation only when the
// caller declared an equivalent tool. The caller executes the returned call.
func DecodeNativeClientToolCall(raw []byte, tools []ToolDefinition) (ServerEvent, bool, error) {
	if len(tools) == 0 {
		return ServerEvent{}, false, nil
	}
	server, err := newMessage("AgentServerMessage")
	if err != nil {
		return ServerEvent{}, false, err
	}
	if err := proto.Unmarshal(raw, server); err != nil {
		return ServerEvent{}, false, fmt.Errorf("decode Cursor native client tool request: %w", err)
	}
	active := server.WhichOneof(server.Descriptor().Oneofs().ByName("message"))
	if active == nil || active.Name() != "exec_server_message" {
		return ServerEvent{}, false, nil
	}
	execMessage := server.Get(active).Message()
	operation := execMessage.WhichOneof(execMessage.Descriptor().Oneofs().ByName("message"))
	if operation == nil {
		return ServerEvent{}, false, nil
	}
	args := execMessage.Get(operation).Message()
	var names []string
	var values map[string]any
	switch operation.Name() {
	case "shell_args", "shell_stream_args":
		names = []string{"Bash", "shell_command", "exec_command", "shell"}
		values = map[string]any{"command": args.Get(field(args, "command")).String(), "working_directory": args.Get(field(args, "working_directory")).String()}
	case "grep_args":
		names = []string{"Grep", "grep"}
		values = map[string]any{"pattern": args.Get(field(args, "pattern")).String(), "path": args.Get(field(args, "path")).String(), "glob": args.Get(field(args, "glob")).String()}
	case "read_args":
		names = []string{"Read", "read_file"}
		values = map[string]any{"path": args.Get(field(args, "path")).String()}
	default:
		return ServerEvent{}, false, nil
	}
	for _, name := range names {
		for _, tool := range tools {
			if !strings.EqualFold(tool.Name, name) {
				continue
			}
			arguments, ok := nativeArguments(tool, string(operation.Name()), values)
			if !ok {
				continue
			}
			callID := args.Get(field(args, "tool_call_id")).String()
			if callID == "" {
				callID = execMessage.Get(field(execMessage, "exec_id")).String()
			}
			if callID == "" {
				callID = fmt.Sprintf("exec_%d", execMessage.Get(field(execMessage, "id")).Uint())
			}
			return ServerEvent{Kind: EventToolCall, Type: "exec_server_message." + string(operation.Name()), ID: callID, Name: tool.Name, Arguments: arguments}, true, nil
		}
	}
	return ServerEvent{}, false, nil
}

func nativeArguments(tool ToolDefinition, operation string, values map[string]any) (string, bool) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Parameters, &schema); err != nil || len(schema.Properties) == 0 {
		return "", false
	}
	result := make(map[string]any)
	put := func(value any, keys ...string) bool {
		for _, key := range keys {
			if _, exists := schema.Properties[key]; exists {
				result[key] = value
				return true
			}
		}
		return false
	}
	switch operation {
	case "shell_args", "shell_stream_args":
		command := values["command"].(string)
		if command == "" {
			return "", false
		}
		if cwd := values["working_directory"].(string); cwd != "" {
			if !put(cwd, "working_directory", "workdir", "cwd") {
				if !strings.EqualFold(tool.Name, "Bash") {
					return "", false
				}
				command = "cd '" + strings.ReplaceAll(cwd, "'", "'\\''") + "' && " + command
			}
		}
		if !put(command, "command", "cmd") {
			return "", false
		}
		put("Run the shell command requested by Cursor", "description")
	case "grep_args":
		if pattern := values["pattern"].(string); pattern == "" || !put(pattern, "pattern") {
			return "", false
		}
		if path := values["path"].(string); path != "" && !put(path, "path") {
			return "", false
		}
		if glob := values["glob"].(string); glob != "" && !put(glob, "glob") {
			return "", false
		}
	case "read_args":
		if path := values["path"].(string); path == "" || !put(path, "file_path", "path") {
			return "", false
		}
	}
	for _, required := range schema.Required {
		if _, ok := result[required]; !ok {
			return "", false
		}
	}
	encoded, err := json.Marshal(result)
	return string(encoded), err == nil
}
