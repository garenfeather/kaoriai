package codex

// Conversation validation helpers.
//
// NOTE: Codex exports can contain sessions with only user messages (e.g. aborted runs).
// These should be treated as invalid and skipped (not synced to DB).

// ValidateConversation returns whether the parsed conversation should be synced.
// For now we only enforce: at least one assistant message exists.
func ValidateConversation(conv *ParsedConversation) (ok bool, reason string) {
	if conv == nil {
		return false, "nil conversation"
	}

	assistantCount := 0
	for _, m := range conv.Messages {
		if m.Role == "assistant" {
			assistantCount++
		}
	}

	if assistantCount == 0 {
		return false, "no assistant messages"
	}

	return true, ""
}
