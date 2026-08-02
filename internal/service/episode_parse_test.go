package service

import "testing"

func TestParseEpisodeFromTitle(t *testing.T) {
	cases := []struct {
		title   string
		season  int
		episode int
		ok      bool
	}{
		{"Show.Name.S01E05.1080p.WEB", 1, 5, true},
		{"Show Name S02E10 720p", 2, 10, true},
		{"Show.S01.1080p.COMPLETE", 1, 0, true},
		{"Show Season 3 1080p", 3, 0, true},
		{"Show.1x07.HDTV", 1, 7, true},
		{"Random.Movie.2020.1080p", 0, 0, false},
	}
	for _, tc := range cases {
		p, ok := ParseEpisodeFromTitle(tc.title)
		if ok != tc.ok {
			t.Fatalf("%q ok=%v want %v", tc.title, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if p.Season != tc.season || p.Episode != tc.episode {
			t.Fatalf("%q => S%02dE%02d want S%02dE%02d", tc.title, p.Season, p.Episode, tc.season, tc.episode)
		}
	}
}

func TestMatchesQuality(t *testing.T) {
	if !MatchesQuality("Show.S01E01.1080p.WEB", "1080p") {
		t.Fatal("expected 1080p match")
	}
	if MatchesQuality("Show.S01E01.720p.WEB", "1080p") {
		t.Fatal("720p should not match 1080p filter")
	}
	if !MatchesQuality("Show.S01E01.720p.HDTV", "720p") {
		t.Fatal("expected 720p match")
	}
	if MatchesQuality("Show.S01E01.1080p.WEB", "720p") {
		t.Fatal("1080p should not match 720p filter")
	}
	// Real bug: SD/unknown WEBRip without resolution must not pass as 1080p
	bad := "House.of.the.Dragon.S01.CRAV.WEBRip.x264.HUN.ENG-FULCRUM"
	if MatchesQuality(bad, "1080p") {
		t.Fatalf("%q must not match 1080p", bad)
	}
	if MatchesQuality(bad, "720p") {
		t.Fatalf("%q must not match 720p", bad)
	}
	good := "House.of.the.Dragon.S01.1080p.BluRay.x264-GROUP"
	if !MatchesQuality(good, "1080p") {
		t.Fatalf("%q should match 1080p", good)
	}
}
