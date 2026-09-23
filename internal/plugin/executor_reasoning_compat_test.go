package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"cursorplugin/internal/cursorapi"
	"cursorplugin/internal/cursorproto"

	"github.com/stretchr/testify/require"
)

func TestClaudeToolResponseOmitsUnsignedCursorThinking(t *testing.T) {
	for _, source := range []string{"claude", "openai"} {
		t.Run(source, func(t *testing.T) {
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
			request.SourceFormat = source
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
			if source == "claude" {
				require.Empty(t, decoded.Choices[0].Message.ReasoningContent)
			} else {
				require.Equal(t, "private reasoning", decoded.Choices[0].Message.ReasoningContent)
			}
		})
	}
}
