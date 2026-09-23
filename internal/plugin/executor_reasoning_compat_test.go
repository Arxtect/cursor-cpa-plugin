package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"cursorplugin/internal/cursorapi"
	"cursorplugin/internal/cursorproto"

	"github.com/stretchr/testify/require"
)

func TestToolResponseOmitsUnsignedCursorThinking(t *testing.T) {
	for _, withTools := range []bool{true, false} {
		name := "without tools"
		if withTools {
			name = "with tools"
		}
		t.Run(name, func(t *testing.T) {
			client := &recordingCursorClient{steps: []cursorRunStep{{
				events: []cursorproto.ServerEvent{
					{Kind: cursorproto.EventThinking, Text: "private reasoning"},
					{Kind: cursorproto.EventToolCall, ID: "call_1", Name: "Bash", Arguments: `{"command":"pwd"}`},
				},
				result: cursorapi.RunResult{ToolExposed: true},
			}}}
			handler := NewHandler(Dependencies{Cursor: client})
			raw := executorFixture(t, "session", "account", "auth", "auto", "", []map[string]any{textMessage("user", "Use Bash")})
			var request executorRequest
			require.NoError(t, json.Unmarshal(raw, &request))
			if withTools {
				var payload map[string]any
				require.NoError(t, json.Unmarshal(request.Payload, &payload))
				payload["tools"] = []map[string]any{{
					"type": "function",
					"function": map[string]any{
						"name": "Bash", "parameters": map[string]any{"type": "object"},
					},
				}}
				encodedPayload, err := json.Marshal(payload)
				require.NoError(t, err)
				request.Payload = encodedPayload
			}
			raw, err := json.Marshal(request)
			require.NoError(t, err)

			response, err := handler.execute(context.Background(), raw)
			require.NoError(t, err)
			payload := response.(executorResponse).Payload
			var decoded struct {
				Choices []struct {
					Message struct {
						ReasoningContent string `json:"reasoning_content"`
						ToolCalls        []any  `json:"tool_calls"`
					} `json:"message"`
				} `json:"choices"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			require.Len(t, decoded.Choices, 1)
			require.Len(t, decoded.Choices[0].Message.ToolCalls, 1)
			if withTools {
				require.Empty(t, decoded.Choices[0].Message.ReasoningContent)
			} else {
				require.Equal(t, "private reasoning", decoded.Choices[0].Message.ReasoningContent)
			}
		})
	}
}
