package claude

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestTreatsZeroMaxTokensAsUnset(t *testing.T) {
	zero := uint(0)
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-5",
		MaxTokens: &zero,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-sonnet-4-5",
		},
	}

	out, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, req)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.NotNil(t, converted.MaxTokens)
	assert.Equal(t, uint(model_setting.GetClaudeSettings().GetDefaultMaxTokens(req.Model)), *converted.MaxTokens)
}

func TestConvertClaudeRequestPreservesNativeClaudeCodeThinking(t *testing.T) {
	budget := 10000
	maxTokens := uint(20000)
	temperature := 0.7
	topP := 0.9
	req := &dto.ClaudeRequest{
		Model:        "claude-opus-4-8",
		MaxTokens:    &maxTokens,
		Temperature:  &temperature,
		TopP:         &topP,
		Thinking:     &dto.Thinking{Type: "enabled", BudgetTokens: &budget},
		OutputConfig: []byte(`{"effort":"high"}`),
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: req.Model,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: req.Model,
		},
	}

	out, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, req)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.NotNil(t, converted.Thinking)
	assert.Equal(t, "enabled", converted.Thinking.Type)
	require.NotNil(t, converted.Thinking.BudgetTokens)
	assert.Equal(t, budget, *converted.Thinking.BudgetTokens)
	assert.Equal(t, temperature, *converted.Temperature)
	assert.Equal(t, topP, *converted.TopP)
	assert.JSONEq(t, `{"effort":"high"}`, string(converted.OutputConfig))
	assert.Empty(t, info.ConversionDiagnostics())
}

func TestConvertClaudeRequestZeroMaxTokensStillRaisesThinkingBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	zero := uint(0)
	original := &dto.ClaudeRequest{
		Model:     "claude-3-7-sonnet-thinking",
		MaxTokens: &zero,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "claude-3-7-sonnet-thinking",
		Request:         original,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-7-sonnet-thinking",
		},
	}
	outbound, err := common.DeepCopy(original)
	require.NoError(t, err)
	require.NoError(t, helper.ModelMappedHelper(c, info, outbound))
	err = helper.ApplyReasoningModelSuffix(nil, info, outbound)
	require.NoError(t, err)

	out, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, outbound)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)
	assert.Equal(t, "claude-3-7-sonnet", converted.Model)
	require.NotNil(t, converted.Thinking)
	require.NotNil(t, converted.MaxTokens)
	assert.Greater(t, *converted.MaxTokens, uint(1024))
}

func TestConvertClaudeRequestDoesNotOverwriteTrimmedUpstreamModelName(t *testing.T) {
	req := &dto.ClaudeRequest{
		Model: "claude-3-7-sonnet-thinking",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-7-sonnet",
		},
	}

	_, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, req)
	require.NoError(t, err)
	assert.Equal(t, "claude-3-7-sonnet", info.UpstreamModelName)
}

func geminiToClaudeInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: "claude-3-7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-7-sonnet",
		},
	}
}

func TestConvertGeminiRequestMapsSystemInstructionToolsAndMultimodal(t *testing.T) {
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "What is in this image?"},
					{InlineData: &dto.GeminiInlineData{MimeType: "image/png", Data: "aGVsbG8="}},
				},
			},
		},
		SystemInstructions: &dto.GeminiChatContent{
			Parts: []dto.GeminiPart{{Text: "You are a helpful assistant."}},
		},
	}
	req.SetTools([]dto.GeminiChatTool{
		{
			FunctionDeclarations: []dto.FunctionRequest{
				{
					Name:        "lookup",
					Description: "Lookup data",
					Parameters: map[string]any{
						"type":       "object",
						"properties": map[string]any{"q": map[string]any{"type": "string"}},
					},
				},
			},
		},
	})

	out, err := (&Adaptor{}).ConvertGeminiRequest(nil, geminiToClaudeInfo(), req)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)

	system := converted.ParseSystem()
	require.NotEmpty(t, system)
	assert.Contains(t, system[0].GetText(), "You are a helpful assistant.")
	require.NotEmpty(t, converted.Messages)
	assert.Equal(t, "user", converted.Messages[0].Role)

	blocks, parseErr := converted.Messages[0].ParseContent()
	require.NoError(t, parseErr)
	var foundImage bool
	for _, block := range blocks {
		if block.Type == "image" || (block.Source != nil && block.Source.Type == "base64") {
			foundImage = true
			break
		}
	}
	assert.True(t, foundImage)

	require.NotNil(t, converted.Tools)
	tools, err := common.Marshal(converted.Tools)
	require.NoError(t, err)
	assert.Contains(t, string(tools), `"lookup"`)
	require.NotNil(t, converted.MaxTokens)
	assert.Greater(t, *converted.MaxTokens, uint(0))
}

func TestConvertGeminiRequestThinkingConfigUsesReasoningIntent(t *testing.T) {
	budget := 1024
	maxTokens := uint(4096)
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{Text: "think"}}},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			MaxOutputTokens: &maxTokens,
			ThinkingConfig:  &dto.GeminiThinkingConfig{ThinkingBudget: &budget},
		},
	}

	out, err := (&Adaptor{}).ConvertGeminiRequest(nil, geminiToClaudeInfo(), req)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.NotNil(t, converted.Thinking)
	assert.Equal(t, "enabled", converted.Thinking.Type)
	require.NotNil(t, converted.Thinking.BudgetTokens)
	assert.Equal(t, 1024, *converted.Thinking.BudgetTokens)
}

func TestConvertGeminiRequestNilRequest(t *testing.T) {
	_, err := (&Adaptor{}).ConvertGeminiRequest(nil, geminiToClaudeInfo(), nil)
	require.Error(t, err)
}

// enableClaudeAutoCache sets AutoCacheEnabled for the duration of a test and
// restores the previous value on cleanup, since it lives on the shared
// package-level Claude settings instance.
func enableClaudeAutoCache(t *testing.T, enabled bool) {
	t.Helper()
	settings := model_setting.GetClaudeSettings()
	orig := settings.AutoCacheEnabled
	settings.AutoCacheEnabled = enabled
	t.Cleanup(func() { settings.AutoCacheEnabled = orig })
}

func TestApplyAutoPromptCachingSkipsShortContent(t *testing.T) {
	enableClaudeAutoCache(t, true)

	req := &dto.ClaudeRequest{
		Model:  "claude-sonnet-4-5",
		System: "You are a helpful assistant.",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
			{Role: "user", Content: "how are you"},
		},
	}

	applyAutoPromptCaching(req)

	assert.True(t, req.IsStringSystem(), "short system prompt should be left as a plain string")
	for i, message := range req.Messages {
		assert.Truef(t, message.IsStringContent(), "message %d content should be untouched", i)
	}
}

func TestApplyAutoPromptCachingMarksLongSystemAndHistory(t *testing.T) {
	enableClaudeAutoCache(t, true)

	longSystem := strings.Repeat("You are a meticulous coding assistant. ", 200) // well over the 4000-byte floor
	longHistoryTurn := strings.Repeat("Here is the file content under review. ", 200)
	req := &dto.ClaudeRequest{
		Model:  "claude-opus-5",
		System: longSystem,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: longHistoryTurn},
			{Role: "assistant", Content: "Understood, I've reviewed it."},
			{Role: "user", Content: "what's next"},
		},
	}

	applyAutoPromptCaching(req)

	// System prompt: converted to block form with the marker set.
	require.False(t, req.IsStringSystem())
	systemBlocks := req.ParseSystem()
	require.Len(t, systemBlocks, 1)
	assert.Equal(t, longSystem, systemBlocks[0].GetText())
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(systemBlocks[0].CacheControl))

	// History: the marker lands on the message just before the newest turn
	// (index 1, "Understood..."), and the newest message (index 2) is left
	// exactly as the client sent it.
	assert.True(t, req.Messages[2].IsStringContent(), "newest turn must stay untouched")
	require.False(t, req.Messages[1].IsStringContent())
	blocks, err := req.Messages[1].ParseContent()
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "Understood, I've reviewed it.", blocks[0].GetText())
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(blocks[0].CacheControl))
}

func TestApplyAutoPromptCachingRespectsExistingCacheControl(t *testing.T) {
	enableClaudeAutoCache(t, true)

	longSystem := strings.Repeat("You are a meticulous coding assistant. ", 200)
	clientText := strings.Repeat("Here is the file content under review. ", 200)
	req := &dto.ClaudeRequest{
		Model:  "claude-opus-5",
		System: longSystem,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: []dto.ClaudeMediaMessage{{
				Type:         "text",
				Text:         &clientText,
				CacheControl: json.RawMessage(`{"type":"ephemeral","ttl":"1h"}`),
			}}},
			{Role: "assistant", Content: "Understood, I've reviewed it."},
			{Role: "user", Content: "what's next"},
		},
	}

	applyAutoPromptCaching(req)

	assert.True(t, req.IsStringSystem(), "a request that already manages its own caching must be left untouched")
	assert.True(t, req.Messages[1].IsStringContent(), "a request that already manages its own caching must be left untouched")
}

func TestApplyAutoPromptCachingDisabledSetting(t *testing.T) {
	enableClaudeAutoCache(t, false)

	longSystem := strings.Repeat("You are a meticulous coding assistant. ", 200)
	req := &dto.ClaudeRequest{
		Model:  "claude-opus-5",
		System: longSystem,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: strings.Repeat("x", 5000)},
			{Role: "assistant", Content: "ok"},
			{Role: "user", Content: "next"},
		},
	}

	applyAutoPromptCaching(req)

	assert.True(t, req.IsStringSystem(), "auto-caching must not act while disabled in settings")
	for i, message := range req.Messages {
		assert.Truef(t, message.IsStringContent(), "message %d should be untouched when the feature is disabled", i)
	}
}

func TestConvertClaudeRequestAppliesAutoPromptCaching(t *testing.T) {
	enableClaudeAutoCache(t, true)

	longSystem := strings.Repeat("You are a meticulous coding assistant. ", 200)
	req := &dto.ClaudeRequest{
		Model:  "claude-sonnet-4-5",
		System: longSystem,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-sonnet-4-5",
		},
	}

	out, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, req)
	require.NoError(t, err)
	converted, ok := out.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.False(t, converted.IsStringSystem())
	systemBlocks := converted.ParseSystem()
	require.Len(t, systemBlocks, 1)
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(systemBlocks[0].CacheControl))
}
