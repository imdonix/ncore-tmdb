package service

import "testing"

func TestDetectResolution(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Fight.Club.1999.2160p.UHD.BluRay", "2160p"},
		{"Fight.Club.1999.4K.HDR", "2160p"},
		{"Fight.Club.1999.1080p.BluRay", "1080p"},
		{"Fight.Club.1999.FullHD.HUN", "1080p"},
		{"Fight.Club.1999.720p.WEB", "720p"},
		{"Fight.Club.1999.576p.DVDRip", "480p"},
		{"Fight.Club.1999.DVDRip.XviD", "SD"},
		{"Fight.Club.1999.WEBRip.x264", ""},
	}
	for _, tc := range cases {
		if got := DetectResolution(tc.title); got != tc.want {
			t.Errorf("DetectResolution(%q) = %q, want %q", tc.title, got, tc.want)
		}
	}
}

func TestBuildMovieResolutions(t *testing.T) {
	releases := []Torrent{
		{ID: "1", Title: "Movie.2020.720p.WEB", Seeders: 4},
		{ID: "2", Title: "Movie.2020.1080p.WEB", Seeders: 10},
		{ID: "3", Title: "Movie.2020.1080p.BluRay", Seeders: 20},
		{ID: "4", Title: "Movie.2020.WEBRip.x264", Seeders: 99},
	}

	got := buildMovieResolutions(releases)
	if len(got) != 2 {
		t.Fatalf("got %d resolutions, want 2: %+v", len(got), got)
	}
	if got[0].Quality != "1080p" || got[0].Releases != 2 || got[0].Seeders != 20 {
		t.Fatalf("unexpected 1080p entry: %+v", got[0])
	}
	if got[1].Quality != "720p" || got[1].Releases != 1 || got[1].Seeders != 4 {
		t.Fatalf("unexpected 720p entry: %+v", got[1])
	}
}

func TestBuildMovieResolutionsAutoFallback(t *testing.T) {
	releases := []Torrent{
		{ID: "1", Title: "Movie.2020.WEBRip.x264", Seeders: 7},
	}

	got := buildMovieResolutions(releases)
	if len(got) != 1 || got[0].Quality != "Auto" || got[0].Releases != 1 || got[0].Seeders != 7 {
		t.Fatalf("unexpected Auto fallback: %+v", got)
	}

	best, ok := bestMovieRelease(releases, "Auto")
	if !ok || best.ID != "1" {
		t.Fatalf("Auto best release = %+v (ok=%v)", best, ok)
	}
}

func TestBestMovieRelease(t *testing.T) {
	releases := []Torrent{
		{ID: "cam", Title: "Movie.2020.1080p.HDCAM", Seeders: 500},
		{ID: "web", Title: "Movie.2020.1080p.WEB-DL", Seeders: 12},
		{ID: "blu", Title: "Movie.2020.1080p.BluRay", Seeders: 25},
	}

	best, ok := bestMovieRelease(releases, "1080p")
	if !ok || best.ID != "blu" {
		t.Fatalf("bestMovieRelease = %+v (ok=%v), want blu", best, ok)
	}

	// If only a cam exists, it is still usable.
	onlyCam := []Torrent{{ID: "cam", Title: "Movie.2020.1080p.HDCAM", Seeders: 1}}
	best, ok = bestMovieRelease(onlyCam, "1080p")
	if !ok || best.ID != "cam" {
		t.Fatalf("cam fallback = %+v (ok=%v), want cam", best, ok)
	}
}

func TestMovieYearMatches(t *testing.T) {
	cases := []struct {
		title string
		name  string
		year  string
		want  bool
	}{
		{"Dune.2021.1080p.WEB", "Dune", "2021", true},
		{"Dune.1984.1080p.BluRay", "Dune", "2021", false},
		{"Blade.Runner.2049.2017.1080p", "Blade Runner 2049", "2017", true},
		{"1917.2019.1080p", "1917", "2019", true},
		{"Movie.1080p.WEB", "Movie", "2021", true}, // no year in title
	}
	for _, tc := range cases {
		if got := movieYearMatches(tc.title, tc.name, tc.year); got != tc.want {
			t.Errorf("movieYearMatches(%q, %q, %q) = %v, want %v", tc.title, tc.name, tc.year, got, tc.want)
		}
	}
}

func TestMovieFolderName(t *testing.T) {
	if got := MovieFolderName("Fight Club", "1999"); got != "Fight.Club.1999" {
		t.Fatalf("MovieFolderName = %q", got)
	}
}
