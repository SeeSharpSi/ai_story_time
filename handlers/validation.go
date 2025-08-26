package handlers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// validateAIResponse validates AI response content for security and appropriateness
func validateAIResponse(response string) error {
	// Check for script injection attempts
	dangerousPatterns := []string{
		`<script`, `javascript:`, `on\w+\s*=`, `<iframe`, `<object`, `<embed`,
		`eval\s*\(`, `document\.`, `window\.`, `alert\s*\(`, `prompt\s*\(`,
		`confirm\s*\(`, `setTimeout\s*\(`, `setInterval\s*\(`,
		`<img[^>]*src\s*=\s*["'][^"']*javascript:`,
	}

	responseLower := strings.ToLower(response)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(responseLower, pattern) {
			return fmt.Errorf("AI response contains potentially harmful content")
		}
	}

	// Check for excessive special characters (might indicate encoding issues)
	specialChars := 0
	for _, char := range response {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char) && !unicode.IsPunct(char) {
			specialChars++
		}
	}
	if specialChars > len(response)/5 { // More than 20% special characters
		return fmt.Errorf("AI response contains too many special characters")
	}

	// Check response length
	if len(response) > 10000 { // 10KB limit
		return fmt.Errorf("AI response is too long")
	}
	if len(response) < 10 { // Minimum length
		return fmt.Errorf("AI response is too short")
	}

	// Check for proper JSON structure if it contains JSON
	jsonPattern := regexp.MustCompile(`\{.*\}`)
	if jsonPattern.MatchString(response) {
		// Basic JSON validation - check for balanced braces
		openBraces := strings.Count(response, "{")
		closeBraces := strings.Count(response, "}")
		if openBraces != closeBraces {
			return fmt.Errorf("AI response contains malformed JSON structure")
		}
	}

	// Check for excessive repetition (might indicate AI hallucination)
	words := strings.Fields(response)
	if len(words) > 10 {
		wordCount := make(map[string]int)
		for _, word := range words {
			word = strings.ToLower(strings.Trim(word, ".,!?"))
			if len(word) > 3 { // Only count meaningful words
				wordCount[word]++
			}
		}

		maxRepetition := 0
		for _, count := range wordCount {
			if count > maxRepetition {
				maxRepetition = count
			}
		}

		// If any word is repeated more than 10% of the time, flag it
		if maxRepetition > len(words)/10 && maxRepetition > 3 {
			return fmt.Errorf("AI response contains excessive word repetition")
		}
	}

	return nil
}

// sanitizeHTMLContent sanitizes HTML content in AI responses
func sanitizeHTMLContent(content string) string {
	// Remove any script tags that might have slipped through
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove javascript: URLs
	jsURLRegex := regexp.MustCompile(`(?i)javascript:[^\s"']*`)
	content = jsURLRegex.ReplaceAllString(content, "#")

	// Remove event handlers
	eventRegex := regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`)
	content = eventRegex.ReplaceAllString(content, "")

	return content
}
