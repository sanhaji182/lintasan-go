package guard

import (
	"regexp"
	"strings"
)

// ViolationType categorizes what kind of safety issue was detected.
type ViolationType string

const (
	ViolationPII       ViolationType = "pii"
	ViolationInjection ViolationType = "injection"
	ViolationContent   ViolationType = "content"
)

// ViolationSeverity indicates how serious the violation is.
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "low"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityHigh     ViolationSeverity = "high"
	SeverityCritical ViolationSeverity = "critical"
)

// Violation represents a single guard check result.
type Violation struct {
	Type        ViolationType     `json:"type"`
	Severity    ViolationSeverity `json:"severity"`
	Description string            `json:"description"`
	Match       string            `json:"match"`
}

// GuardConfig controls which guard filters are active.
type GuardConfig struct {
	PIIEnabled       bool     `json:"pii_enabled"`
	InjectionEnabled bool     `json:"injection_enabled"`
	ContentEnabled   bool     `json:"content_enabled"`
	BlockList        []string `json:"block_list,omitempty"`
	AllowList        []string `json:"allow_list,omitempty"`
}

// DefaultGuardConfig returns a config with all filters enabled.
func DefaultGuardConfig() GuardConfig {
	return GuardConfig{
		PIIEnabled:       true,
		InjectionEnabled: true,
		ContentEnabled:   true,
	}
}

// ───────────────────────────────────────────────
// PII detection
// ───────────────────────────────────────────────

type piiPattern struct {
	regex       *regexp.Regexp
	label       string
	severity    ViolationSeverity
	description string
}

var piiPatterns = []piiPattern{
	{
		regex:       regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		label:       "email",
		severity:    SeverityHigh,
		description: "Email address detected",
	},
	{
		regex:       regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`),
		label:       "phone",
		severity:    SeverityMedium,
		description: "Phone number detected",
	},
	{
		regex:       regexp.MustCompile(`\b\d{3}[-]\d{2}[-]\d{4}\b`),
		label:       "ssn",
		severity:    SeverityCritical,
		description: "Social Security Number detected",
	},
	{
		regex:       regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})\b`),
		label:       "credit_card",
		severity:    SeverityCritical,
		description: "Credit card number detected",
	},
}

func detectPII(content string) []Violation {
	var violations []Violation
	for _, p := range piiPatterns {
		matches := p.regex.FindAllString(content, -1)
		for _, m := range matches {
			violations = append(violations, Violation{
				Type:        ViolationPII,
				Severity:    p.severity,
				Description: p.description,
				Match:       maskPII(p.label, m),
			})
		}
	}
	return violations
}

// maskPII partially hides the matched value for safe logging.
func maskPII(kind, value string) string {
	switch kind {
	case "email":
		parts := strings.SplitN(value, "@", 2)
		if len(parts) == 2 && len(parts[0]) > 2 {
			return parts[0][:2] + "***@" + parts[1]
		}
	case "phone":
		if len(value) > 4 {
			return "***" + value[len(value)-4:]
		}
	case "ssn":
		return "***-**-" + value[len(value)-4:]
	case "credit_card":
		if len(value) > 4 {
			return "****-****-****-" + value[len(value)-4:]
		}
	}
	return "***"
}

// ───────────────────────────────────────────────
// Injection detection
// ───────────────────────────────────────────────

type injectionPattern struct {
	regex       *regexp.Regexp
	severity    ViolationSeverity
	description string
}

var injectionPatterns = []injectionPattern{
	{
		regex:       regexp.MustCompile(`(?i)ignore\s+(all\s+)?previous\s+(instructions|prompts|rules)`),
		severity:    SeverityCritical,
		description: "Instruction override attempt",
	},
	{
		regex:       regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an)\s+`),
		severity:    SeverityHigh,
		description: "Role reassignment attempt",
	},
	{
		regex:       regexp.MustCompile(`(?i)disregard\s+(all\s+)?prior`),
		severity:    SeverityCritical,
		description: "Instruction disregard attempt",
	},
	{
		regex:       regexp.MustCompile(`(?i)new\s+instructions\s*:`),
		severity:    SeverityHigh,
		description: "New instruction injection",
	},
	{
		regex:       regexp.MustCompile(`(?i)system\s*:\s*you\s+are`),
		severity:    SeverityCritical,
		description: "System prompt injection",
	},
	{
		regex:       regexp.MustCompile(`(?i)\[INST\]|\[/INST\]|<<SYS>>|<</SYS>>`),
		severity:    SeverityHigh,
		description: "Llama format injection attempt",
	},
	{
		regex:       regexp.MustCompile(`(?i)pretend\s+(you\s+are|to\s+be|you're)`),
		severity:    SeverityMedium,
		description: "Persona override attempt",
	},
	{
		regex:       regexp.MustCompile(`(?i)DAN\s+mode|jailbreak|do\s+anything\s+now`),
		severity:    SeverityCritical,
		description: "Jailbreak attempt detected",
	},
}

func detectInjection(content string) []Violation {
	var violations []Violation
	for _, p := range injectionPatterns {
		match := p.regex.FindString(content)
		if match != "" {
			violations = append(violations, Violation{
				Type:        ViolationInjection,
				Severity:    p.severity,
				Description: p.description,
				Match:       match,
			})
		}
	}
	return violations
}

// ───────────────────────────────────────────────
// Content filter (block/allow lists)
// ───────────────────────────────────────────────

func detectContent(content string, cfg GuardConfig) []Violation {
	var violations []Violation
	lower := strings.ToLower(content)

	// Block-list: if content contains any blocked term, flag it
	for _, term := range cfg.BlockList {
		if strings.Contains(lower, strings.ToLower(term)) {
			violations = append(violations, Violation{
				Type:        ViolationContent,
				Severity:    SeverityHigh,
				Description: "Blocked content term: " + term,
				Match:       term,
			})
		}
	}

	// Allow-list: if an allow list is configured and content matches nothing in it, flag it
	if len(cfg.AllowList) > 0 {
		allowed := false
		for _, term := range cfg.AllowList {
			if strings.Contains(lower, strings.ToLower(term)) {
				allowed = true
				break
			}
		}
		if !allowed {
			violations = append(violations, Violation{
				Type:        ViolationContent,
				Severity:    SeverityMedium,
				Description: "Content does not match any allowed terms",
				Match:       "",
			})
		}
	}

	return violations
}

// ───────────────────────────────────────────────
// Public API
// ───────────────────────────────────────────────

// Check runs all enabled guard filters against the content and returns any violations found.
func Check(content string, config GuardConfig) []Violation {
	var violations []Violation

	if config.PIIEnabled {
		violations = append(violations, detectPII(content)...)
	}
	if config.InjectionEnabled {
		violations = append(violations, detectInjection(content)...)
	}
	if config.ContentEnabled {
		violations = append(violations, detectContent(content, config)...)
	}

	if violations == nil {
		violations = []Violation{}
	}
	return violations
}
