package utils

import (
	"regexp"
	"strings"
)

var (
	hashtagRegex = regexp.MustCompile(`(?i)#([a-zA-Z0-9_]+)`) // Case-insensitive
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]{4,30})`) // Usernames 4-30 chars, alphanumeric + underscore
)

func ExtractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	var tags []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			tag := strings.ToLower(match[1])
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}
	return tags
}

func ExtractMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	var usernames []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			username := match[1]
			if !seen[username] {
				usernames = append(usernames, username)
				seen[username] = true
			}
		}
	}
	return usernames
}