package optimizer

import (
	"encoding/json"
	"regexp"
	"strings"
)

var whitespaceRegex = regexp.MustCompile(`[ \t]+`)
var newlineRegex = regexp.MustCompile(`\n{3,}`)
var fillerRegex = regexp.MustCompile(`(?i)\b(very|really|quite|rather|somewhat|fairly|pretty much|just|simply)\b`)

var phraseReplacements = map[*regexp.Regexp]string{
	regexp.MustCompile(`(?i)I would like you to `):   "",
	regexp.MustCompile(`(?i)I want you to `):         "",
	regexp.MustCompile(`(?i)Can you please `):        "",
	regexp.MustCompile(`(?i)Could you please `):      "",
	regexp.MustCompile(`(?i)Please make sure to `):   "",
	regexp.MustCompile(`(?i)Please ensure that `):    "Ensure ",
	regexp.MustCompile(`(?i)It is important that you `): "",
	regexp.MustCompile(`(?i)I need you to `):         "",
	regexp.MustCompile(`(?i)You should make sure to `): "",
	regexp.MustCompile(`(?i)You are required to `):   "",
	regexp.MustCompile(`(?i)In order to `):           "To ",
	regexp.MustCompile(`(?i)Due to the fact that `):  "Because ",
	regexp.MustCompile(`(?i)At this point in time `): "Now ",
	regexp.MustCompile(`(?i)In the event that `):     "If ",
	regexp.MustCompile(`(?i)For the purpose of `):    "For ",
	regexp.MustCompile(`(?i)With regard to `):        "Regarding ",
	regexp.MustCompile(`(?i)In addition to that `):   "Also ",
	regexp.MustCompile(`(?i)As a matter of fact `):   "",
	regexp.MustCompile(`(?i)It should be noted that `): "",
	regexp.MustCompile(`(?i)Please note that `):      "Note: ",
	regexp.MustCompile(`(?i)Keep in mind that `):     "",
	regexp.MustCompile(`(?i)It goes without saying that `): "",
	regexp.MustCompile(`(?i)Needless to say `):       "",
	regexp.MustCompile(`(?i)As previously mentioned `): "",
	regexp.MustCompile(`(?i)As I mentioned before `): "",
	regexp.MustCompile(`(?i)To summarize `):          "",
	regexp.MustCompile(`(?i)In summary `):            "",
	regexp.MustCompile(`(?i)Basically `):             "",
	regexp.MustCompile(`(?i)Essentially `):           "",
	regexp.MustCompile(`(?i)Actually `):              "",
	regexp.MustCompile(`(?i)Obviously `):             "",
	regexp.MustCompile(`(?i)Clearly `):               "",
}

func OptimizePrompt(text string, isSystem bool) string {
	res := text
	for pattern, replacement := range phraseReplacements {
		res = pattern.ReplaceAllString(res, replacement)
	}
	if isSystem {
		res = fillerRegex.ReplaceAllString(res, "")
	}
	res = whitespaceRegex.ReplaceAllString(res, " ")
	res = newlineRegex.ReplaceAllString(res, "\n\n")
	return strings.TrimSpace(res)
}

func OptimizeMessages(messages []any) ([]any, int) {
	optimized := []any{}
	seen := make(map[string]bool)
	savedTokens := 0
	
	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok { optimized = append(optimized, m); continue }
		
		role, _ := msg["role"].(string)
		content := ""
		if c, ok := msg["content"].(string); ok {
			content = c
		} else if b, err := json.Marshal(msg["content"]); err == nil {
			content = string(b)
		}
		
		norm := whitespaceRegex.ReplaceAllString(strings.ToLower(content), " ")
		if role != "system" && seen[norm] {
			savedTokens += len(content) / 4
			continue
		}
		seen[norm] = true
		
		if c, ok := msg["content"].(string); ok && c != "" {
			opt := OptimizePrompt(c, role == "system")
			savedTokens += (len(c) - len(opt)) / 4
			msg["content"] = opt
		}
		optimized = append(optimized, msg)
	}
	return optimized, savedTokens
}
