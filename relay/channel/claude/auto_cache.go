package claude

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

// claudeEphemeralCacheControl is Anthropic's default (5-minute) cache
// marker. It is a fixed literal, not the output of a marshal call, so it is
// exempt from the project's common.Marshal wrapper rule.
var claudeEphemeralCacheControl = json.RawMessage(`{"type":"ephemeral"}`)

const (
	// claudeAutoCacheMinBytes approximates Anthropic's minimum cacheable
	// prompt length (1024 tokens for most Claude models; Haiku needs 2048)
	// at roughly 4 bytes per token for English text. It is intentionally a
	// byte-length proxy rather than a real token count: this runs on every
	// outbound request and has no tokenizer handy, and a conservative floor
	// only costs a missed cache opportunity on borderline content, never a
	// wrong bill. Marking content shorter than this would still work, but
	// would frequently pay the ~1.25x cache-write premium on a prompt that
	// changes again before any cache-read ever lands, guaranteeing a net
	// loss instead of savings.
	claudeAutoCacheMinBytes = 4000
)

// applyAutoPromptCaching marks the system prompt and the conversation
// history preceding the newest turn as cacheable, so repeat requests that
// share the same system prompt and a growing message history (coding
// agents such as Claude Code/ZCode, chat UIs) hit Anthropic's prompt cache
// instead of paying full price for tokens the model already saw moments
// earlier.
//
// It is a no-op when the feature is disabled in Claude settings
// (model_setting.ClaudeSettings.AutoCacheEnabled), or when the incoming
// request already carries any cache_control marker of its own — a client
// managing its own caching is left completely untouched, both to respect
// its choices and to stay under Anthropic's four-breakpoint-per-request
// cap (this function uses at most two: one for the system prompt, one for
// the message history).
func applyAutoPromptCaching(request *dto.ClaudeRequest) {
	if request == nil {
		return
	}
	if !model_setting.GetClaudeSettings().AutoCacheEnabled {
		return
	}
	if claudeRequestHasCacheControl(request) {
		return
	}

	if request.System != nil {
		markClaudeSystemCacheable(request)
	}
	markClaudeHistoryCacheable(request)
}

// claudeRequestHasCacheControl reports whether the request (top-level
// field, any system block, or any message block) already sets its own
// cache_control, in which case auto-caching stands aside entirely rather
// than risk exceeding Anthropic's four-breakpoint cap or fighting the
// client's own placement.
func claudeRequestHasCacheControl(request *dto.ClaudeRequest) bool {
	if len(request.CacheControl) > 0 {
		return true
	}
	if request.System != nil && !request.IsStringSystem() {
		for _, block := range request.ParseSystem() {
			if len(block.CacheControl) > 0 {
				return true
			}
		}
	}
	for _, message := range request.Messages {
		if message.IsStringContent() {
			continue
		}
		blocks, err := message.ParseContent()
		if err != nil {
			continue
		}
		for _, block := range blocks {
			if len(block.CacheControl) > 0 {
				return true
			}
		}
	}
	return false
}

// markClaudeSystemCacheable tags the system prompt as cacheable when it is
// long enough to be worth the cache-write premium. A string system prompt
// is converted into the single-block array form Anthropic's Messages API
// also accepts for system (a functionally identical representation); an
// array system prompt keeps its existing blocks and the marker lands on
// the last one, which is where Anthropic requires it for the whole section
// to be covered.
func markClaudeSystemCacheable(request *dto.ClaudeRequest) {
	if request.IsStringSystem() {
		text := request.GetStringSystem()
		if len(text) < claudeAutoCacheMinBytes {
			return
		}
		request.System = []dto.ClaudeMediaMessage{{
			Type:         "text",
			Text:         &text,
			CacheControl: claudeEphemeralCacheControl,
		}}
		return
	}

	blocks := request.ParseSystem()
	if len(blocks) == 0 {
		return
	}
	raw, err := common.Marshal(blocks)
	if err != nil || len(raw) < claudeAutoCacheMinBytes {
		return
	}
	blocks[len(blocks)-1].CacheControl = claudeEphemeralCacheControl
	request.System = blocks
}

// markClaudeHistoryCacheable tags the end of the conversation history
// preceding the newest message as cacheable. Coding agents and chat
// clients resend the entire prior conversation on every request; caching
// through the previous turn means only the newest message is billed at
// full price on the next call, while everything before it is a cache read.
func markClaudeHistoryCacheable(request *dto.ClaudeRequest) {
	// A single-message request (or none) has no reusable history yet — the
	// one message present is always the "newest" turn.
	if len(request.Messages) < 2 {
		return
	}

	historyEnd := len(request.Messages) - 1 // exclusive of the newest message
	raw, err := common.Marshal(request.Messages[:historyEnd])
	if err != nil || len(raw) < claudeAutoCacheMinBytes {
		return
	}

	// Walk backward from the message just before the newest turn so a
	// trailing message with no cacheable content (e.g. an empty tool
	// result) doesn't silently skip caching the rest of the history.
	for i := historyEnd - 1; i >= 0; i-- {
		if markClaudeMessageCacheable(&request.Messages[i]) {
			return
		}
	}
}

// markClaudeMessageCacheable tags the last content block of a single
// message as cacheable, converting string content into the equivalent
// single-block array form when needed. It returns false for a message with
// no markable content, so the caller can fall back to an earlier message.
func markClaudeMessageCacheable(message *dto.ClaudeMessage) bool {
	if message.IsStringContent() {
		text := message.GetStringContent()
		if text == "" {
			return false
		}
		message.SetContent([]dto.ClaudeMediaMessage{{
			Type:         "text",
			Text:         &text,
			CacheControl: claudeEphemeralCacheControl,
		}})
		return true
	}

	blocks, err := message.ParseContent()
	if err != nil || len(blocks) == 0 {
		return false
	}
	blocks[len(blocks)-1].CacheControl = claudeEphemeralCacheControl
	message.SetContent(blocks)
	return true
}
