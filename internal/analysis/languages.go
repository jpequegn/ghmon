package analysis

import (
	"sort"

	"github.com/julienpequegnot/ghmon/internal/activity"
)

type LanguageStats struct {
	Language   string
	Count      int
	Percentage float64
}

// AnalyzeLanguages aggregates language usage from repos and stars
func AnalyzeLanguages(repos []activity.Repo, stars []activity.Star) []LanguageStats {
	counts := make(map[string]int)
	total := 0

	for _, repo := range repos {
		if repo.Language != "" {
			counts[repo.Language]++
			total++
		}
	}

	for _, star := range stars {
		if star.RepoLanguage != "" {
			counts[star.RepoLanguage]++
			total++
		}
	}

	if total == 0 {
		return nil
	}

	var stats []LanguageStats
	for lang, count := range counts {
		stats = append(stats, LanguageStats{
			Language:   lang,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// Return top 5
	if len(stats) > 5 {
		stats = stats[:5]
	}

	return stats
}

// GetTopLanguageNames returns just the language names
func GetTopLanguageNames(stats []LanguageStats) []string {
	names := make([]string, len(stats))
	for i, s := range stats {
		names[i] = s.Language
	}
	return names
}
