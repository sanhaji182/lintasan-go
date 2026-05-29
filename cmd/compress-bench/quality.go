package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Quality check: a real "did we lose anything critical?" test, not a token
// count. Two principles, both learned from measuring the real compressors:
//
//  1. PROTECTED ZONE. The context compressor (compress.New) only PROMISES to
//     preserve the system message(s) plus the last keepLastN messages verbatim;
//     everything in the middle is fair game to summarize away. So a "critical
//     marker" is one that lives in that protected zone. A file:line that only
//     ever appeared in a 40-line grep dump in the middle is NOT a quality loss
//     when it's collapsed — that is the whole point of compression. We therefore
//     score survival of PROTECTED markers (target ~100%) and report incidental
//     mid-context markers separately as informational only.
//
//  2. COMPONENT MATCH. RTK reformats grep output: "bucket.go:42:\t// TODO"
//     becomes "bucket.go:" on one line and "42: // TODO" on the next. The path
//     and the line number both survive, just not as the contiguous substring
//     "bucket.go:42". A naive strings.Contains would falsely report it lost. So
//     a file:line marker counts as "survived" when BOTH its file path AND its
//     line number survive in the compressed text, contiguous or not.

var (
	// Go-style file:line locations: internal/proxy/proxy.go:418
	reFileLoc = regexp.MustCompile(`\b[a-zA-Z0-9_./-]+\.(?:go|js|ts|py|rs|java|c|cpp|h):\d+\b`)
	// ALLCAPS status/error codes: ERR_OVERFLOW_4096, ECONNREFUSED.
	reCode = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b`)
	// Error-ish lines.
	reErrorLine = regexp.MustCompile(`(?im)^.*(?:\berror\b|\bpanic\b|\bfatal\b|\bfailed\b|\bexception\b|\bcannot\b|\bFAIL\b|traceback)[^\n]*$`)
)

// markerSet is the deduplicated set of critical markers found in some text,
// grouped by category.
type markerSet struct {
	fileLocs []string // file.go:line
	paths    []string // bare dir/file paths
	codes    []string // ERR_* style codes
	errors   []string // error-line signatures
}

func extractMarkers(text string) markerSet {
	return markerSet{
		fileLocs: uniqueSorted(reFileLoc.FindAllString(text, -1)),
		paths:    nil, // bare paths intentionally excluded: too noisy (URLs, \n escape
		// artifacts in JSON tool output, route strings). file:line refs in fileLocs
		// are the actionable debugging markers; codes + error lines cover the rest.
		codes:  uniqueSorted(reCode.FindAllString(text, -1)),
		errors: uniqueSorted(extractErrorSignatures(text)),
	}
}

func (m markerSet) total() int {
	return len(m.fileLocs) + len(m.paths) + len(m.codes) + len(m.errors)
}

// filterPaths removed: bare-path markers were too noisy (URLs, \n escape
// artifacts, route strings). file:line refs are tracked via fileLocs instead.

func extractErrorSignatures(text string) []string {
	var sigs []string
	for _, line := range reErrorLine.FindAllString(text, -1) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 60 {
			line = line[:60]
		}
		sigs = append(sigs, line)
	}
	return sigs
}

// fileLocSurvives reports whether a "path.go:line" marker survives in the
// compressed text. It counts as survived when BOTH the file path AND the line
// number survive (contiguous or reformatted onto separate lines), because RTK's
// grep filter legitimately splits "file.go:42" into a "file.go:" header line
// followed by "  42: ..." lines.
func fileLocSurvives(loc, compressed string) bool {
	if strings.Contains(compressed, loc) {
		return true // contiguous, easy case
	}
	idx := strings.LastIndex(loc, ":")
	if idx < 0 {
		return strings.Contains(compressed, loc)
	}
	path := loc[:idx]
	line := loc[idx+1:]
	if !strings.Contains(compressed, path) {
		return false
	}
	// Grep-reformat case: the path appears once as a header ("file.go:") and the
	// line number shows up later as "  <line>: ...". Accept if the path survives
	// AND the line number survives as its own token anywhere after the path's
	// first occurrence. The line is specific enough (e.g. ":418") that a stray
	// collision is unlikely within the same file's block.
	pathPos := strings.Index(compressed, path)
	rest := compressed[pathPos+len(path):]
	lineToken := regexp.MustCompile(`(?m)(^|\s|:)` + regexp.QuoteMeta(line) + `(\b|:)`)
	return lineToken.MatchString(rest)
}

// QualityResult is the outcome of one quality check.
type QualityResult struct {
	// Protected zone = system + last keepLastN messages (what compress.New
	// promises to keep). This is the real quality score.
	ProtectedTotal    int
	ProtectedSurvived int
	ProtectedRate     float64
	ProtectedLost     map[string][]string

	// Incidental = markers that ONLY appeared in the compressible middle.
	// Losing these is expected/correct, reported for transparency only.
	IncidentalTotal    int
	IncidentalSurvived int
}

func (q QualityResult) Note() string {
	if q.ProtectedTotal == 0 {
		return "no critical markers in protected zone (clean input)"
	}
	base := fmt.Sprintf("protected %d/%d survived %.0f%%",
		q.ProtectedSurvived, q.ProtectedTotal, q.ProtectedRate*100)
	if q.IncidentalTotal > 0 {
		dropped := q.IncidentalTotal - q.IncidentalSurvived
		base += fmt.Sprintf("; %d/%d mid-context markers dropped (expected)",
			dropped, q.IncidentalTotal)
	}
	if len(q.ProtectedLost) > 0 {
		var cats []string
		for cat, lost := range q.ProtectedLost {
			cats = append(cats, fmt.Sprintf("%d %s", len(lost), cat))
		}
		sort.Strings(cats)
		base = "⚠ LOST PROTECTED " + strings.Join(cats, ", ") + " — " + base
	}
	return base
}

// markerSurvives checks one marker of a given category against the compressed text.
func markerSurvives(cat, marker, compressed string) bool {
	if cat == "fileLocs" {
		return fileLocSurvives(marker, compressed)
	}
	return strings.Contains(compressed, marker)
}

// checkQuality compares critical markers against the compressed text, splitting
// them into protected-zone markers (must survive) and incidental mid-context
// markers (OK to drop). protectedText is the concatenation of the system +
// last keepLastN message contents from the ORIGINAL input.
func checkQuality(protectedText, fullOriginal, compressed string) QualityResult {
	res := QualityResult{ProtectedLost: map[string][]string{}}

	protSet := extractMarkers(protectedText)
	fullSet := extractMarkers(fullOriginal)

	// Build a lookup of protected markers per category.
	protected := map[string]map[string]bool{
		"fileLocs": toSet(protSet.fileLocs),
		"paths":    toSet(protSet.paths),
		"codes":    toSet(protSet.codes),
		"errors":   toSet(protSet.errors),
	}

	cats := map[string][]string{
		"fileLocs": fullSet.fileLocs,
		"paths":    fullSet.paths,
		"codes":    fullSet.codes,
		"errors":   fullSet.errors,
	}

	for cat, list := range cats {
		for _, m := range list {
			survived := markerSurvives(cat, m, compressed)
			if protected[cat][m] {
				res.ProtectedTotal++
				if survived {
					res.ProtectedSurvived++
				} else {
					res.ProtectedLost[cat] = append(res.ProtectedLost[cat], m)
				}
			} else {
				res.IncidentalTotal++
				if survived {
					res.IncidentalSurvived++
				}
			}
		}
	}

	if res.ProtectedTotal > 0 {
		res.ProtectedRate = float64(res.ProtectedSurvived) / float64(res.ProtectedTotal)
	} else {
		res.ProtectedRate = 1.0
	}
	return res
}

func toSet(in []string) map[string]bool {
	s := map[string]bool{}
	for _, v := range in {
		s[v] = true
	}
	return s
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
