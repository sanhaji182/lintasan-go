package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// --- F2.0 canonical vocabulary -----------------------------------------------

func TestCanonicalVocabularyComplete(t *testing.T) {
	// Every constant the package defines must be in the canonical vocabulary,
	// and the vocabulary must not contain anything that isn't a real constant.
	want := []Capability{
		CapCoding, CapReasoning, CapToolCalling, CapStreaming, CapVision,
		CapEmbeddings, CapLongContext, CapJSONMode, CapCheap, CapFast, CapPremium,
	}
	if len(CanonicalVocabulary) != len(want) {
		t.Fatalf("vocabulary size mismatch: got %d want %d", len(CanonicalVocabulary), len(want))
	}
	for _, c := range want {
		if !IsCanonical(c) {
			t.Fatalf("capability %q missing from canonical vocabulary", c)
		}
	}
}

func TestCapJSONModePresent(t *testing.T) {
	// The audit found json_mode missing; F2.0 adds it. Lock it in.
	if CapJSONMode != "json_mode" {
		t.Fatalf("CapJSONMode value drift: %q", CapJSONMode)
	}
	if !IsCanonical(CapJSONMode) {
		t.Fatal("CapJSONMode must be part of the canonical vocabulary")
	}
}

func TestVocabularyReturnsCopy(t *testing.T) {
	v := Vocabulary()
	if len(v) == 0 {
		t.Fatal("vocabulary copy should be non-empty")
	}
	v[0] = Capability("mutated")
	// Mutating the copy must not corrupt the source of truth.
	if CanonicalVocabulary[0] == "mutated" {
		t.Fatal("Vocabulary() leaked a mutable reference to the source of truth")
	}
}

func TestIsCanonicalRejectsUnknown(t *testing.T) {
	if IsCanonical(Capability("not-a-real-cap")) {
		t.Fatal("unknown capability must not be reported canonical")
	}
}

// --- F2.0 pure mapping helpers -----------------------------------------------

func TestCatalogTagToCapability(t *testing.T) {
	cases := []struct {
		tag    string
		want   Capability
		wantOK bool
	}{
		{"streaming", CapStreaming, true},
		{"tools", CapToolCalling, true}, // vocabulary mismatch reconciled
		{"vision", CapVision, true},
		{"json_mode", CapJSONMode, true},
		{"chat", "", false}, // baseline, not a capability
		{"unknown-xyz", "", false},
	}
	for _, c := range cases {
		got, ok := CatalogTagToCapability(c.tag)
		if got != c.want || ok != c.wantOK {
			t.Fatalf("CatalogTagToCapability(%q) = (%q,%v) want (%q,%v)", c.tag, got, ok, c.want, c.wantOK)
		}
	}
}

func TestCatalogTagsToSet(t *testing.T) {
	// Mirror a real catalog entry: gpt-4o tags.
	set := CatalogTagsToSet([]string{"chat", "vision", "tools", "streaming", "json_mode"})
	for _, want := range []Capability{CapVision, CapToolCalling, CapStreaming, CapJSONMode} {
		if !set.Has(want) {
			t.Fatalf("expected set to contain %q", want)
		}
	}
	// "chat" is baseline and must be dropped — set has exactly 4 entries.
	if len(set) != 4 {
		t.Fatalf("expected 4 mapped caps (chat dropped), got %d: %v", len(set), set.List())
	}
}

func TestAutoModeToCapability(t *testing.T) {
	cases := []struct {
		mode   string
		want   Capability
		wantOK bool
	}{
		{"auto/coding", CapCoding, true},
		{"auto/fast", CapFast, true},
		{"auto/cheap", CapCheap, true},
		{"auto", "", false}, // balanced — no specific capability
		{"auto/nonsense", "", false},
	}
	for _, c := range cases {
		got, ok := AutoModeToCapability(c.mode)
		if got != c.want || ok != c.wantOK {
			t.Fatalf("AutoModeToCapability(%q) = (%q,%v) want (%q,%v)", c.mode, got, ok, c.want, c.wantOK)
		}
	}
}

// --- F2.0 NON-CONSUMPTION GUARD ----------------------------------------------
//
// This is the load-bearing test for the F2.0 scope contract: the canonical
// vocabulary and its mapping helpers must NOT be wired into any runtime path.
// If a future change (intentionally or accidentally) makes the live proxy/
// router/provider-selection consume these symbols, this test fails loudly and
// forces an explicit checkpoint review — exactly the F2.1/F2.4 gate.
//
// Method: scan the server package (the home of the proxy/router/handlers) for
// any reference to the F2.0 vocabulary symbols. F2.0 introduces ZERO such refs.

func TestF2_0_VocabularyNotConsumedByServer(t *testing.T) {
	serverDir := filepath.Join("..", "server")
	if _, err := os.Stat(serverDir); err != nil {
		t.Skipf("server package not found at %s (skipping): %v", serverDir, err)
	}

	// Symbols introduced/owned by the F2.0 canonical vocabulary surface.
	forbidden := regexp.MustCompile(`\b(CanonicalVocabulary|IsCanonical|Vocabulary\(|CatalogTagToCapability|CatalogTagsToSet|AutoModeToCapability|CapJSONMode)\b`)

	var offenders []string
	err := filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if forbidden.Match(data) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk server package: %v", err)
	}
	if len(offenders) != 0 {
		t.Fatalf("F2.0 scope violation: server package references F2.0 vocabulary symbols (must stay declaration-only until F2.1): %v", offenders)
	}
}
