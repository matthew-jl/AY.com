package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHashtags(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{"no hashtags", "Hello world", []string{}},
		{"simple hashtag", "Hello #Svelte world", []string{"svelte"}},
		{"multiple hashtags", "I love #Go and #Svelte_Rocks!", []string{"go", "svelte_rocks"}},
		{"case insensitive", "Check out #TeStInG", []string{"testing"}},
		{"duplicate hashtags", "#test #Test #TEST", []string{"test"}},
		{"alphanumeric", "#Tag123 and #Another_Tag", []string{"tag123", "another_tag"}},
		{"no tag after hash", "Hello # world", []string{}},
		{"empty content", "", []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ExtractHashtags(tc.content)
			if len(tc.expected) == 0 && len(actual) == 0 {
				return
			}
			assert.ElementsMatch(t, tc.expected, actual, "Extracted hashtags do not match expected")
		})
	}
}

func TestExtractMentions(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{"no mentions", "Hello world", []string{}},
		{"simple mention", "Hi @tester", []string{"tester"}},
		{"multiple mentions", "@user_one and @another_user123", []string{"user_one", "another_user123"}},
		{"mention too short", "Hey @usr", []string{}},
		{"mention at start", "@startUser hello", []string{"startUser"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ExtractMentions(tc.content)
            if len(tc.expected) == 0 && len(actual) == 0 {
				return
			}
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}