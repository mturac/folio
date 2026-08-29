package store

import "strings"

// Snippet returns a window of body around the first query term hit.
func Snippet(body, query string, radius int) string {
	if radius <= 0 {
		radius = 80
	}
	body = strings.ReplaceAll(body, "\n", " ")
	fields := strings.Fields(strings.ToLower(query))
	lower := strings.ToLower(body)
	idx := -1
	termLen := 0
	for _, f := range fields {
		if f == "" {
			continue
		}
		if i := strings.Index(lower, f); i >= 0 {
			idx = i
			termLen = len(f)
			break
		}
	}
	if idx < 0 {
		if len(body) <= radius*2 {
			return body
		}
		return body[:radius*2] + "…"
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + termLen + radius
	if end > len(body) {
		end = len(body)
	}
	out := body[start:end]
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out = out + "…"
	}
	return out
}
