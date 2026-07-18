// Package subjects holds the shared slug→CDN PDF id catalog for project subjects.
package subjects

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

// Index maps project slug to the numeric CDN PDF id
// (cdn.intra.42.fr/pdf/pdf/{id}/{lang}.subject.pdf).
type Index map[string]int

var embedded Index

func init() {
	embedded = mustParse(catalogJSON)
}

func mustParse(data []byte) Index {
	idx, err := Parse(data)
	if err != nil {
		panic("subjects: invalid embedded catalog.json: " + err.Error())
	}
	return idx
}

// Parse decodes a JSON object of slug→pdf-id.
func Parse(data []byte) (Index, error) {
	var raw map[string]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse subject catalog: %w", err)
	}
	out := make(Index, len(raw))
	for slug, id := range raw {
		slug = strings.TrimSpace(slug)
		if slug == "" || id <= 0 {
			continue
		}
		out[slug] = id
	}
	return out, nil
}

// Lookup returns the embedded catalog PDF id for slug, or 0.
func Lookup(slug string) int {
	return embedded[strings.TrimSpace(slug)]
}

// Embedded returns a copy of the shipped catalog.
func Embedded() Index {
	out := make(Index, len(embedded))
	for k, v := range embedded {
		out[k] = v
	}
	return out
}

// Merge copies entries from src into dst. Returns how many keys were new
// and how many existing keys changed value.
func Merge(dst, src Index) (added, updated int) {
	if dst == nil {
		return 0, 0
	}
	for slug, id := range src {
		slug = strings.TrimSpace(slug)
		if slug == "" || id <= 0 {
			continue
		}
		prev, ok := dst[slug]
		if !ok {
			dst[slug] = id
			added++
			continue
		}
		if prev != id {
			dst[slug] = id
			updated++
		}
	}
	return added, updated
}

// MergeAbsent copies only keys missing from dst (never overwrites).
func MergeAbsent(dst, src Index) (added int) {
	if dst == nil {
		return 0
	}
	for slug, id := range src {
		slug = strings.TrimSpace(slug)
		if slug == "" || id <= 0 {
			continue
		}
		if _, ok := dst[slug]; ok {
			continue
		}
		dst[slug] = id
		added++
	}
	return added
}

// MatchSlug finds a catalog slug by exact or substring match on query.
func MatchSlug(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if _, ok := embedded[query]; ok {
		return query
	}
	lower := strings.ToLower(query)
	for slug := range embedded {
		if strings.EqualFold(slug, query) {
			return slug
		}
	}
	var hit string
	for slug := range embedded {
		if strings.Contains(strings.ToLower(slug), lower) {
			if hit != "" && hit != slug {
				return "" // ambiguous
			}
			hit = slug
		}
	}
	return hit
}
