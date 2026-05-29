package compress

// Caveman mode: inject terse reply instructions to save output tokens.
// Inspired by Caveman (https://github.com/JuliusBrussee/caveman) — 52K stars.

const cavemanSystemPrompt = `Reply terse. Short sentences. No filler words. Technical substance only. Use code blocks. Skip pleasantries. No restating what user said. Direct answers only. If fixing code, show the fix — don't explain what was wrong unless asked.`

const cavemanLitePrompt = `Be concise. Prefer short sentences. Skip pleasantries and filler.`

const cavemanAggressivePrompt = `Caveman mode. Ultra terse. Single words when possible. Code > prose. No explanations unless asked. No pleasantries. No restating. Bullet points only. Max 2 sentences per point.`

// InjectCaveman modifies messages to get terse LLM responses.
// Adds caveman instruction to the system message.
// mode: "lite", "standard", "aggressive"
func InjectCaveman(messages []map[string]any, mode string) []map[string]any {
	if mode == "" || mode == "off" {
		return messages
	}

	var prompt string
	switch mode {
	case "lite":
		prompt = cavemanLitePrompt
	case "aggressive":
		prompt = cavemanAggressivePrompt
	default: // "standard"
		prompt = cavemanSystemPrompt
	}

	result := make([]map[string]any, len(messages))
	copy(result, messages)

	// Find and update system message, or prepend one
	found := false
	for i, msg := range result {
		if role, ok := msg["role"].(string); ok && role == "system" {
			if content, ok := msg["content"].(string); ok {
				result[i] = map[string]any{
					"role":    "system",
					"content": content + "\n\n" + prompt,
				}
				found = true
				break
			}
		}
	}

	if !found {
		// Prepend system message
		sysMsg := map[string]any{
			"role":    "system",
			"content": prompt,
		}
		result = append([]map[string]any{sysMsg}, result...)
	}

	return result
}

// InjectCavemanPrompt is a convenience function that returns just the prompt string.
func InjectCavemanPrompt(mode string) string {
	switch mode {
	case "lite":
		return cavemanLitePrompt
	case "aggressive":
		return cavemanAggressivePrompt
	default:
		return cavemanSystemPrompt
	}
}
