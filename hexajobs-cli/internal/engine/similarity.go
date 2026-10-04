package engine

import (
	"math"
	"strings"
	"unicode"
)

// NormalizeSkill handles common aliases without collapsing C, C++, or C#.
func NormalizeSkill(skill string) string {
	s := strings.ToLower(strings.Join(strings.Fields(skill), " "))
	switch s {
	case "golang", "go language":
		return "go"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "postgres", "postgresql":
		return "postgresql"
	case "k8s":
		return "kubernetes"
	case "node", "nodejs", "node.js":
		return "node.js"
	case "reactjs", "react.js":
		return "react"
	}
	return s
}

func skillSet(skills []string) map[string]struct{} {
	out := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		if s := NormalizeSkill(skill); s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}

func Jaccard(a, b []string) float64 {
	left, right := skillSet(a), skillSet(b)
	intersection := 0
	for skill := range left {
		if _, ok := right[skill]; ok {
			intersection++
		}
	}
	union := len(left) + len(right) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// Tokens implements a small deterministic lexical model, not embedding semantics.
func Tokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+' && r != '#' && r != '.'
	})
}

func CosineSimilarity(a, b string) float64 {
	counts := func(s string) map[string]float64 {
		out := make(map[string]float64)
		for _, token := range Tokens(s) {
			if token = NormalizeSkill(strings.Trim(token, ".")); token != "" {
				out[token]++
			}
		}
		return out
	}
	left, right := counts(a), counts(b)
	var dot, aa, bb float64
	for word, value := range left {
		dot += value * right[word]
		aa += value * value
	}
	for _, value := range right {
		bb += value * value
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return math.Min(1, dot/math.Sqrt(aa*bb))
}
