package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDamerauLevenshteinDistance(t *testing.T) {
	testCases := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{"empty strings", "", "", 0},
		{"s1 empty", "", "abc", 3},
		{"s2 empty", "abc", "", 3},
		{"identical", "test", "test", 0},
		{"identical case diff", "Test", "test", 0},
		{"substitution", "kitten", "sitten", 1},
		{"insertion", "sit", "sits", 1},
		{"deletion", "saturday", "sturday", 1},
		{"transposition", "ca", "ac", 1},
		{"complex transposition", "abcdef", "abcfed", 2},
		{"known transposition 1", "apple", "apply", 1},
		{"known transposition 2", "banana", "banaan", 1},
		{"transposition test", "act", "cat", 1},
		{"longer transposition", "damerau", "admerau", 1},
		{"multiple edits", "analyze", "analyse", 1},
		{"unicode simple", "gö", "go", 1},
		{"unicode transposition", "ümlaut", "umlaüt", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := DamerauLevenshteinDistance(tc.s1, tc.s2)
			assert.Equal(t, tc.expected, actual, "Damerau-Levenshtein distance mismatch for '%s' and '%s'", tc.s1, tc.s2)
		})
	}
}

func TestCalculateSimilarityNormalized(t *testing.T) {
	assert.InDelta(t, 1.0, CalculateSimilarityNormalized("test", "test"), 0.001)
	assert.InDelta(t, 0.0, CalculateSimilarityNormalized("", "testtest"), 0.001)
	assert.InDelta(t, 1.0, CalculateSimilarityNormalized("", ""), 0.001)
	assert.InDelta(t, 0.8, CalculateSimilarityNormalized("apple", "apply"), 0.001)
	assert.InDelta(t, 0.5, CalculateSimilarityNormalized("ca", "ac"), 0.001)
}
