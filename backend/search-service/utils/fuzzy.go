package utils

import (
	"strings"
	"unicode/utf8"
)

// returns the minimum of a list of integers
func min(nums ...int) int {
	if len(nums) == 0 {
		panic("min: no arguments")
	}
	m := nums[0]
	for _, num := range nums[1:] {
		if num < m {
			m = num
		}
	}
	return m
}

// considers insertions, deletions, substitutions, and transpositions of adjacent characters
// works with runes to correctly handle multi-byte characters
func DamerauLevenshteinDistance(s1, s2 string) int {
	r1 := []rune(strings.ToLower(s1))
	r2 := []rune(strings.ToLower(s2))
	lenS1 := len(r1)
	lenS2 := len(r2)

	if lenS1 == 0 {
		return lenS2
	}
	if lenS2 == 0 {
		return lenS1
	}

	// d[i][j] will be the D-L distance between the first i characters of s1 and the first j characters of s2
	d := make([][]int, lenS1+1)
	for i := range d {
		d[i] = make([]int, lenS2+1)
	}

	// Initialize the first row and column of the matrix
	// Cost of deleting all characters from s1 to get an empty string
	for i := 0; i <= lenS1; i++ {
		d[i][0] = i
	}
	// Cost of inserting all characters from s2 into an empty string
	for j := 0; j <= lenS2; j++ {
		d[0][j] = j
	}

	// Fill the rest of the matrix
	for i := 1; i <= lenS1; i++ {
		for j := 1; j <= lenS2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			// Standard Levenshtein operations
			deletion := d[i-1][j] + 1
			insertion := d[i][j-1] + 1
			substitution := d[i-1][j-1] + cost
			d[i][j] = min(deletion, insertion, substitution)

			// Damerau-Levenshtein: check for transposition of adjacent characters
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				// The cost of the swap itself is 1, so we add it to the state before the two chars.
				// d[i-2][j-2] + 1 (cost of transposition)
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[lenS1][lenS2]
}

// returns a similarity score between 0 and 1
// 1 means identical, 0 means completely different
func CalculateSimilarityNormalized(s1, s2 string) float64 {
	if s1 == "" && s2 == "" {
		return 1.0
	}
	maxLen := utf8.RuneCountInString(s1)
	if utf8.RuneCountInString(s2) > maxLen {
		maxLen = utf8.RuneCountInString(s2)
	}
	if maxLen == 0 {
		return 0.0
	}
	distance := DamerauLevenshteinDistance(s1, s2)
	return 1.0 - (float64(distance) / float64(maxLen))
}

func CalculateSimilarityPerWord(target, query string) float64 {
	if target == "" || query == "" {
		return 0.0
	}

	words := strings.Fields(strings.ToLower(target))
	if len(words) == 0 {
		return 0.0
	}

	normalizedQuery := strings.ToLower(query)
	maxSimilarity := 0.0

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:'\"()[]{}")
		if word == "" {
			continue
		}
		similarity := CalculateSimilarityNormalized(word, normalizedQuery)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}
	}

	return maxSimilarity
}