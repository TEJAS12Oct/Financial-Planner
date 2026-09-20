package utils

import (
	"regexp"
	"strings"
)

var invalidFileChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

// SanitizeFilename converts a string into a safe filename.
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)

	if name == "" {
		return "financial_planner_report"
	}

	name = strings.ReplaceAll(name, " ", "_")
	name = invalidFileChars.ReplaceAllString(name, "")

	return name
}
