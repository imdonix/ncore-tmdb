package service

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// S01E02, s1e2, S01.E02
	reSeasonEpisode = regexp.MustCompile(`(?i)S(\d{1,2})\s*[. _-]*E(\d{1,3})`)
	// 1x02
	reSeasonX = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{1,3})\b`)
	// Full season markers
	reSeasonOnly = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])S(\d{1,2})(?:[^A-Za-z0-9]|$)`)
	reSeasonWord = regexp.MustCompile(`(?i)Season\s*(\d{1,2})`)
)

// ParsedRelease describes what a torrent title covers.
// Episode == 0 means full season pack.
type ParsedRelease struct {
	Season  int
	Episode int // 0 = season pack
	Raw     string
}

// ParseEpisodeFromTitle extracts season/episode from common nCore naming.
func ParseEpisodeFromTitle(title string) (ParsedRelease, bool) {
	// Episode patterns first so S01E01 is not treated as a season pack
	if m := reSeasonEpisode.FindStringSubmatch(title); len(m) == 3 {
		s, _ := strconv.Atoi(m[1])
		e, _ := strconv.Atoi(m[2])
		if s > 0 && e > 0 {
			return ParsedRelease{Season: s, Episode: e, Raw: m[0]}, true
		}
	}
	if m := reSeasonX.FindStringSubmatch(title); len(m) == 3 {
		s, _ := strconv.Atoi(m[1])
		e, _ := strconv.Atoi(m[2])
		if s > 0 && e > 0 {
			return ParsedRelease{Season: s, Episode: e, Raw: m[0]}, true
		}
	}

	// Season packs: only if no episode pattern matched
	if m := reSeasonOnly.FindStringSubmatch(title); len(m) == 2 {
		s, _ := strconv.Atoi(m[1])
		if s > 0 {
			// If the title still has SxxEyy elsewhere, prefer episode (already handled)
			return ParsedRelease{Season: s, Episode: 0, Raw: m[0]}, true
		}
	}
	if m := reSeasonWord.FindStringSubmatch(title); len(m) == 2 {
		// Avoid "Season 1 Episode 2" style if E pattern failed — still treat as pack if no Eyy
		if reSeasonEpisode.MatchString(title) {
			// shouldn't reach here
		} else if !strings.Contains(strings.ToLower(title), "episode") {
			s, _ := strconv.Atoi(m[1])
			if s > 0 {
				return ParsedRelease{Season: s, Episode: 0, Raw: m[0]}, true
			}
		}
	}
	return ParsedRelease{}, false
}

// MatchesQuality filters by requested resolution in the title.
// Requires an explicit resolution tag — bare WEBRip/x264 without 720p/1080p is rejected
// (e.g. House.of.the.Dragon.S01.CRAV.WEBRip.x264 must not count as 1080p).
func MatchesQuality(title, quality string) bool {
	t := strings.ToLower(title)
	q := strings.ToLower(strings.TrimSpace(quality))

	has2160 := strings.Contains(t, "2160p") || strings.Contains(t, "4k") || strings.Contains(t, "uhd")
	has1080 := strings.Contains(t, "1080p") || strings.Contains(t, "1080i") ||
		strings.Contains(t, "fullhd") || strings.Contains(t, "full.hd")
	has720 := strings.Contains(t, "720p")

	switch q {
	case "1080p":
		// Must explicitly claim 1080p — no guessing from bare WEBRip/x264
		return has1080
	case "720p":
		if has1080 || has2160 {
			return false
		}
		// Must explicitly claim 720p
		return has720
	default:
		return true
	}
}

// TitleLooksLikeSeries does a light check the release is for the show name.
func TitleLooksLikeSeries(title, seriesName string) bool {
	t := normalizeLoose(title)
	n := normalizeLoose(seriesName)
	if n == "" {
		return true
	}
	for _, tok := range strings.Fields(n) {
		if len(tok) < 2 {
			continue
		}
		if !strings.Contains(t, tok) {
			return false
		}
	}
	return true
}

func normalizeLoose(s string) string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(".", " ", "_", " ", "-", " ", "'", "", ":", " ", "!", " ", ",", " ")
	s = repl.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
