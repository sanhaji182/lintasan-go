package memory

import (
	"hash/fnv"
	"math"
	"regexp"
	"strings"
)

const EmbeddingDim = 384

var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "shall": true, "can": true,
	"in": true, "for": true, "on": true, "with": true, "at": true, "by": true,
	"from": true, "as": true, "to": true, "of": true, "and": true, "or": true,
	"if": true, "what": true, "which": true, "who": true, "this": true, "that": true,
	"it": true, "my": true, "your": true, "his": true, "her": true, "their": true,
	"its": true, "he": true, "she": true, "they": true, "we": true, "you": true,
	"me": true, "him": true, "them": true, "us": true, "not": true, "no": true,
	"but": true, "so": true, "then": true, "than": true, "just": true, "also": true,
	"very": true, "too": true, "only": true, "now": true, "up": true, "out": true,
	"about": true, "into": true, "over": true, "after": true, "all": true,
	"how": true, "when": true, "where": true, "why": true, "get": true, "got": true,
	"put": true, "set": true, "let": true, "go": true, "see": true, "use": true,
	"make": true, "made": true, "take": true, "say": true, "said": true,
}

var nonAlnum = regexp.MustCompile(`[^\w\s]`)

func stem(w string) string {
	if len(w) <= 4 {
		return w
	}
	switch {
	case strings.HasSuffix(w, "ation"):
		return w[:len(w)-5]
	case strings.HasSuffix(w, "ment"), strings.HasSuffix(w, "ness"),
		strings.HasSuffix(w, "tion"), strings.HasSuffix(w, "sion"),
		strings.HasSuffix(w, "able"), strings.HasSuffix(w, "ible"),
		strings.HasSuffix(w, "less"), strings.HasSuffix(w, "ally"):
		return w[:len(w)-4]
	case strings.HasSuffix(w, "ing"), strings.HasSuffix(w, "ful"),
		strings.HasSuffix(w, "ous"), strings.HasSuffix(w, "ive"):
		return w[:len(w)-3]
	case strings.HasSuffix(w, "ies"):
		return w[:len(w)-3] + "y"
	case strings.HasSuffix(w, "ly"), strings.HasSuffix(w, "er"),
		strings.HasSuffix(w, "ed"), strings.HasSuffix(w, "es"):
		return w[:len(w)-2]
	case strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return w[:len(w)-1]
	}
	return w
}

func tokenize(text string) []string {
	clean := nonAlnum.ReplaceAllString(strings.ToLower(text), " ")
	words := strings.Fields(clean)
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		w = stem(w)
		if len(w) > 2 && !stopwords[w] {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func buildTF(tokens []string) map[string]float64 {
	tf := make(map[string]float64, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}
	for t, v := range tf {
		tf[t] = math.Log(1 + v)
	}
	return tf
}

func hashToDim(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() % EmbeddingDim)
}

// Embed turns text into a 384-dimensional normalized vector.
func Embed(text string) []float64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return make([]float64, EmbeddingDim)
	}
	tf := buildTF(tokens)

	vec := make([]float64, EmbeddingDim)
	for term, weight := range tf {
		dim := hashToDim(term)
		vec[dim] += weight
	}

	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

// CosineSimilarity computes cosine similarity between two equal-length vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (magA * magB)
}
