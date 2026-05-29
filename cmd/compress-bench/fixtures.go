package main

import (
	_ "embed"
	"fmt"
	"strings"
)

// Fixtures for Level 1, Level 2 and Level 4 are embedded from text files so the
// harness is fully reproducible and deterministic. Level 3 is assembled in code
// from a fixed template using fixed loop counts (no RNG) so repeated runs
// produce byte-identical input.
//
// IMPORTANT — the context compressor (compress.New) decides whether to fire
// using a char/4 token ESTIMATE, not a real tokenizer: it compresses only when
// total content exceeds ~compressThreshold*4 characters (8000 est-tokens =
// ~32000 chars). Levels 3 and 4 are therefore deliberately sized WELL ABOVE
// ~32000 characters so the context compressor actually runs. (The harness still
// REPORTS sizes using the real tiktoken cl100k_base count.)

//go:embed fixtures/l1_clean.txt
var l1Clean string

//go:embed fixtures/l2_toolout.txt
var l2ToolOut string

//go:embed fixtures/l4_real_session.txt
var l4RealSession string

// ---- Level 1: clean chat input, ~200 tokens, no tool output ----

func level1Messages() []map[string]any {
	return []map[string]any{
		{"role": "system", "content": "You are a senior Go engineer. Answer concisely and recommend pragmatic, well-tested solutions."},
		{"role": "user", "content": strings.TrimSpace(l1Clean)},
	}
}

// ---- Level 2: tool output (RTK territory) ----
// Production compresses each tool result message separately (CompressMessages
// scans per-message). The embedded fixture holds several distinct tool outputs
// separated by a marker; we split them into one tool message each so RTK can
// auto-detect the right filter (git/grep/tree/ls/log) per message.

func level2Messages() []map[string]any {
	msgs := []map[string]any{
		{"role": "system", "content": "You are a coding assistant with shell access."},
		{"role": "user", "content": "Check the repo state and the rate limiter package before we continue."},
	}
	for _, block := range splitToolBlocks(l2ToolOut) {
		msgs = append(msgs, map[string]any{"role": "tool", "content": block})
	}
	return msgs
}

// splitToolBlocks splits the L2 fixture into separate tool outputs on the "$ "
// command-prompt boundary, so each block is a single tool's output that RTK can
// classify cleanly.
func splitToolBlocks(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var blocks []string
	var cur []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "$ ") && len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
		cur = append(cur, ln)
	}
	if len(cur) > 0 {
		blocks = append(blocks, strings.Join(cur, "\n"))
	}
	return blocks
}

// ---- Level 3: large context (>32K chars so the compressor fires) ----
// The context compressor (compress.New) keeps system + last keepLastN(6)
// messages intact and summarizes everything in the MIDDLE. To exercise it we
// build many messages where the middle is long, repetitive, low-information
// tool output — exactly what accumulates in a long session. The ONE critical
// fact (an overflow error with a file:line + error code) is planted in BOTH the
// system prompt (protected zone) and buried in the middle (compressible zone),
// so the quality check can prove the protected copy survives.

const l3BatchTemplate = `worker[%d]: batch start offset=%d size=256 partition=%d
worker[%d]: fetched rows=256 from shard=%d latency=12ms cache=warm
worker[%d]: transform applied rules=14 dropped=0 coerced=3
worker[%d]: validate ok schema=v7 nulls=0 duplicates=0
worker[%d]: upsert ok rows=256 conflicts=0 wal=checkpointed
worker[%d]: batch complete offset=%d duration=1.84s mem=42MB`

func level3Messages() []map[string]any {
	const iterations = 160 // ~46K chars total => char/4 ~11.5K est-tokens > 8000 threshold

	msgs := []map[string]any{
		{"role": "system", "content": "You are debugging a data pipeline. Critical: the failure is an integer overflow in shard 7 at offset 1048576, error code ERR_OVERFLOW_4096 in internal/pipeline/shard.go:212."},
		{"role": "user", "content": "Here is the full worker log. Find why shard 7 stalled."},
	}

	for i := 0; i < iterations; i++ {
		worker := i%8 + 1
		shard := i % 8
		offset := i * 256
		batch := fmt.Sprintf(l3BatchTemplate,
			worker, offset, shard, worker, shard, worker, worker, worker, worker, offset)

		msgs = append(msgs,
			map[string]any{"role": "assistant", "content": fmt.Sprintf("Inspecting batch %d on shard %d.", i, shard)},
			map[string]any{"role": "tool", "content": batch})

		// Bury the one critical error in the middle of the repetition.
		if i == iterations/2 {
			msgs = append(msgs, map[string]any{"role": "tool",
				"content": "ERROR worker[7]: integer overflow at offset=1048576 shard=7 ERR_OVERFLOW_4096 internal/pipeline/shard.go:212"})
		}
	}

	// Recent turns (kept intact by keepLastN).
	msgs = append(msgs,
		map[string]any{"role": "user", "content": "What's the root cause and which file and line?"},
		map[string]any{"role": "assistant", "content": "Checking the overflow guard."})
	return msgs
}

// ---- Level 4: REAL Hermes session (redacted), ~134K chars ----
// This is a genuine captured Hermes coding/ops session exported from the
// session store (~/.hermes/state.db), with emails/tokens/phone numbers redacted.
// It is NOT synthetic. At ~134K characters it is far above the compressor's
// ~32K-char firing threshold, so both RTK (per-message) and the context
// compressor actually run on real-world content.
//
// Each message is stored as a block:  <<<MSG role=ROLE>>>\n<content>
func level4Messages() []map[string]any {
	return parseSessionBlocks(l4RealSession)
}

// parseSessionBlocks parses the embedded real-session fixture into messages.
func parseSessionBlocks(raw string) []map[string]any {
	var msgs []map[string]any
	parts := strings.Split(raw, "<<<MSG role=")
	for _, p := range parts {
		p = strings.TrimRight(p, "\n")
		if strings.TrimSpace(p) == "" {
			continue
		}
		// p looks like:  ROLE>>>\n<content>
		gt := strings.Index(p, ">>>")
		if gt < 0 {
			continue
		}
		role := strings.TrimSpace(p[:gt])
		content := strings.TrimPrefix(p[gt+3:], "\n")
		if role == "" {
			role = "user"
		}
		msgs = append(msgs, map[string]any{"role": role, "content": content})
	}
	return msgs
}
