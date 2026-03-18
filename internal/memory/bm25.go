package memory

import (
	"math"
	"strings"
	"unicode"
)

// BM25Scorer scores documents against a query using the BM25 ranking function
// (k1=1.5, b=0.75 are the standard Okapi BM25 defaults).
type BM25Scorer struct{}

// Score returns a BM25 relevance score for each document in corpus relative to
// query. corpus and query must be pre-tokenised (use tokenize). Returns a zero
// slice when corpus is empty or query is empty.
func (BM25Scorer) Score(corpus [][]string, query []string) []float64 {
	n := len(corpus)
	scores := make([]float64, n)
	if n == 0 || len(query) == 0 {
		return scores
	}

	const k1 = 1.5
	const b = 0.75

	// Average document length.
	total := 0
	for _, doc := range corpus {
		total += len(doc)
	}
	avgdl := float64(total) / float64(n)

	// Document frequencies per query term.
	df := make(map[string]int, len(query))
	for _, doc := range corpus {
		seen := make(map[string]bool)
		for _, term := range doc {
			if !seen[term] {
				df[term]++
				seen[term] = true
			}
		}
	}

	for i, doc := range corpus {
		// Term frequencies in this document.
		tf := make(map[string]int)
		for _, term := range doc {
			tf[term]++
		}
		docLen := float64(len(doc))

		for _, term := range query {
			if df[term] == 0 {
				continue
			}
			idf := math.Log((float64(n)-float64(df[term])+0.5)/(float64(df[term])+0.5) + 1)
			tfNorm := float64(tf[term]) * (k1 + 1) / (float64(tf[term]) + k1*(1-b+b*docLen/avgdl))
			scores[i] += idf * tfNorm
		}
	}
	return scores
}

// bm25Stopwords is a minimal English stopword list used during tokenisation.
var bm25Stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "is": true, "it": true, "be": true, "are": true, "was": true,
	"were": true, "as": true, "this": true, "that": true, "by": true, "from": true,
	"do": true, "have": true, "not": true, "they": true, "we": true, "he": true,
	"she": true, "you": true, "i": true, "my": true, "your": true, "his": true,
}

// tokenize lowercases text, splits on non-alphanumeric characters, and removes
// short tokens and common English stopwords.
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) > 1 && !bm25Stopwords[w] {
			out = append(out, w)
		}
	}
	return out
}
