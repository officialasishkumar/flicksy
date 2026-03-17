package letterboxd

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var normalizeNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
var runtimePattern = regexp.MustCompile(`([0-9]+)(?:\s|&nbsp;|&#160;)+mins`)
var trailingYearPattern = regexp.MustCompile(`\s*\([0-9]{4}\)$`)

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = normalizeNonAlnum.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func sameTitle(left, right string) bool {
	return normalizeText(left) == normalizeText(right)
}

func numberFromString(value string) int {
	value = strings.ReplaceAll(value, ",", "")
	value = strings.TrimSpace(value)
	number, _ := strconv.Atoi(value)
	return number
}

func slugFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "film" {
			return parts[i+1]
		}
	}

	return parts[len(parts)-1]
}

func pathFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Path
}

func titleWithoutYear(value string) string {
	return strings.TrimSpace(trailingYearPattern.ReplaceAllString(strings.TrimSpace(value), ""))
}
