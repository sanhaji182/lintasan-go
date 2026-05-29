package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/compress"
)

// compress-bench: a 4-level token-compression measurement harness for the
// Lintasan Go compress package. It measures, per level:
//   1. input vs output tokens -> % reduction   (REAL tiktoken cl100k_base)
//   2. compression latency overhead (ms)
//   3. quality check -> did any critical info disappear
//   4. a warning flag if compression INCREASED tokens (a bug)
//
// "Measure first, then get smart." This harness produces the numbers that
// justify (or kill) the later smart-routing work.

// keepLastN is the number of trailing messages the context compressor preserves
// verbatim. It MUST stay in sync with the value passed to compress.New below —
// the quality check uses it to define the "protected zone", so a drift here
// would silently make the harness measure the wrong messages. Single source of
// truth: change it once and both the compressor and the quality check follow.
const keepLastN = 6

type levelResult struct {
	level      int
	name       string
	mode       string // which compressor path was exercised
	inTokens   int
	outTokens  int
	reduction  float64 // 0-1, can be negative if compression grew the input
	latencyMS  float64
	quality    QualityResult
	grewTokens bool
	synthetic  bool
}

// runLevel applies a compress function to a set of messages and gathers all 4
// metrics. The compress function takes the rendered message blob (and the raw
// messages for context-level compression) and returns the compressed blob.
func runLevel(level int, name, mode string, messages []map[string]any, synthetic bool,
	compressFn func([]map[string]any) (string, []map[string]any)) levelResult {

	originalBlob := joinMessageContent(messages)
	protectedBlob := protectedZoneContent(messages, keepLastN)
	inTokens := countMessageTokens(messages)

	// Time only the compression work.
	start := time.Now()
	compressedBlob, compressedMsgs := compressFn(messages)
	latency := time.Since(start)

	var outTokens int
	if compressedMsgs != nil {
		outTokens = countMessageTokens(compressedMsgs)
	} else {
		// String-level compression (RTK/pipeline): count the compressed blob
		// plus the same per-message role overhead is not meaningful, so we
		// compare raw content token counts for both sides.
		inTokens = countTokens(originalBlob)
		outTokens = countTokens(compressedBlob)
	}

	reduction := 0.0
	if inTokens > 0 {
		reduction = 1.0 - float64(outTokens)/float64(inTokens)
	}

	res := levelResult{
		level:      level,
		name:       name,
		mode:       mode,
		inTokens:   inTokens,
		outTokens:  outTokens,
		reduction:  reduction,
		latencyMS:  float64(latency.Microseconds()) / 1000.0,
		quality:    checkQuality(protectedBlob, originalBlob, compressedBlob),
		grewTokens: outTokens > inTokens,
		synthetic:  synthetic,
	}
	return res
}

// protectedZoneContent returns the concatenated content of the messages that the
// context compressor PROMISES to preserve verbatim: every system message plus
// the last keepLastN messages. Markers found here are the ones that must
// survive; markers that live only in the compressible middle are incidental.
func protectedZoneContent(messages []map[string]any, keepLastN int) string {
	var parts []string
	n := len(messages)
	for i, msg := range messages {
		role, _ := msg["role"].(string)
		if role == "system" || i >= n-keepLastN {
			parts = append(parts, extractContent(msg))
		}
	}
	return strings.Join(parts, "\n")
}

func main() {
	jsonOut := flag.Bool("json", false, "emit results as JSON instead of a table")
	flag.Parse()

	// Sanity-check the tokenizer up front so a bad init fails loud, not silent.
	if _, err := tokenizer(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: tokenizer init failed: %v\n", err)
		os.Exit(1)
	}

	var results []levelResult

	// --- Level 1: clean input. RTK pipeline must NOT grow or damage it. ---
	results = append(results, runLevel(1, "Clean input", "pipeline(standard)",
		level1Messages(), false,
		func(msgs []map[string]any) (string, []map[string]any) {
			blob := joinMessageContent(msgs)
			out, _ := compress.CompressPipeline(blob, "standard")
			return out, nil
		}))

	// --- Level 2: tool output. RTK dedup/grouping territory. Production path
	// is CompressMessages (per-message RTK on tool/user content). We use "lite"
	// mode to isolate the input-side RTK dedup — "standard"/"aggressive" also
	// inject the caveman system prompt, whose benefit is reducing the LLM's
	// OUTPUT tokens, which this input-focused harness does not measure. ---
	results = append(results, runLevel(2, "Tool output", "CompressMessages(lite)",
		level2Messages(), false,
		func(msgs []map[string]any) (string, []map[string]any) {
			out, _ := compress.CompressMessages(msgs, "lite")
			return joinMessageContent(out), out
		}))

	// --- Level 3: large context. Context compressor collapses the middle. ---
	results = append(results, runLevel(3, "Large context", "compress.New(8000,6,8000)",
		level3Messages(), false,
		func(msgs []map[string]any) (string, []map[string]any) {
			c := compress.New(8000, keepLastN, 8000)
			out, _ := c.Compress(msgs)
			return joinMessageContent(out), out
		}))

	// --- Level 4: REAL Hermes session (redacted). RTK + context combined. ---
	results = append(results, runLevel(4, "Agent session", "RTK msgs + context",
		level4Messages(), false,
		func(msgs []map[string]any) (string, []map[string]any) {
			// Stage 1: RTK-compress individual tool/user messages (lite = pure
			// RTK, no caveman prompt injection).
			staged, _ := compress.CompressMessages(msgs, "lite")
			// Stage 2: context-compress the whole history.
			c := compress.New(8000, keepLastN, 8000)
			out, _ := c.Compress(staged)
			return joinMessageContent(out), out
		}))

	if *jsonOut {
		printJSON(results)
		return
	}
	printTable(results)
}

func printTable(results []levelResult) {
	fmt.Println()
	fmt.Println("Lintasan Go — Token Compression Measurement Harness")
	fmt.Println("Tokenizer: tiktoken cl100k_base (real BPE, offline/embedded vocab)")
	fmt.Println(strings.Repeat("=", 110))
	fmt.Printf("%-3s %-15s %-26s %9s %9s %10s %9s\n",
		"Lvl", "Scenario", "Compressor", "in_tok", "out_tok", "reduction", "lat_ms")
	fmt.Println(strings.Repeat("-", 110))

	for _, r := range results {
		name := r.name
		if r.synthetic {
			name += "*"
		}
		fmt.Printf("%-3d %-15s %-26s %9d %9d %9.1f%% %9.2f\n",
			r.level, name, r.mode, r.inTokens, r.outTokens, r.reduction*100, r.latencyMS)
	}
	fmt.Println(strings.Repeat("-", 110))

	// Quality + bug notes per level.
	fmt.Println("\nQuality check (critical-info survival) and bug flags:")
	for _, r := range results {
		flag := ""
		if r.grewTokens {
			flag = "  ⚠ BUG: compression INCREASED token count"
		}
		fmt.Printf("  L%d %-15s : %s%s\n", r.level, r.name, r.quality.Note(), flag)
		// List a few lost PROTECTED markers if any, for actionable review.
		if len(r.quality.ProtectedLost) > 0 {
			for cat, lost := range r.quality.ProtectedLost {
				show := lost
				if len(show) > 3 {
					show = show[:3]
				}
				fmt.Printf("        lost protected %s: %s%s\n", cat, strings.Join(show, ", "),
					moreSuffix(len(lost)))
			}
		}
	}

	fmt.Println("\n(no synthetic fixtures — L4 is a real redacted Hermes session)")
	fmt.Println("Note: \"protected\" = markers in system + last 6 messages (the zone the context")
	fmt.Println("      compressor promises to keep). Mid-context markers dropped from repetitive")
	fmt.Println("      tool output are EXPECTED — that is the point of compression, not a loss.")
	fmt.Println("      Level 1 target is ~0% reduction with zero quality loss.")
}

func moreSuffix(n int) string {
	if n > 3 {
		return fmt.Sprintf(" (+%d more)", n-3)
	}
	return ""
}

func printJSON(results []levelResult) {
	fmt.Println("[")
	for i, r := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		fmt.Printf(`  {"level":%d,"name":%q,"mode":%q,"in_tokens":%d,"out_tokens":%d,`+
			`"reduction_pct":%.2f,"latency_ms":%.3f,"protected_survival_pct":%.1f,`+
			`"protected_total":%d,"protected_survived":%d,"incidental_total":%d,`+
			`"incidental_survived":%d,"grew_tokens":%t,"synthetic":%t}%s`+"\n",
			r.level, r.name, r.mode, r.inTokens, r.outTokens, r.reduction*100, r.latencyMS,
			r.quality.ProtectedRate*100, r.quality.ProtectedTotal, r.quality.ProtectedSurvived,
			r.quality.IncidentalTotal, r.quality.IncidentalSurvived,
			r.grewTokens, r.synthetic, comma)
	}
	fmt.Println("]")
}
