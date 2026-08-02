package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

var apiKey string
var baseURL = "https://api.themoviedb.org/3"

func InitTMDB() {

	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		log.Fatal("TMDB_API_KEY environment variable required")
	}

	apiKey = tmdbAPIKey
}

func fetchTMDB(endpoint string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", apiKey)

	fullURL := baseURL + endpoint + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func GetMovieDetailsTMDB(tmdbID int) (map[string]any, error) {
	data, err := fetchTMDB(fmt.Sprintf("/movie/%d", tmdbID), nil)
	if err != nil {
		return nil, err
	}

	var details map[string]any
	err = json.Unmarshal(data, &details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

func GetTVDetailsTMDB(tmdbID int) (map[string]any, error) {
	data, err := fetchTMDB(fmt.Sprintf("/tv/%d", tmdbID), nil)
	if err != nil {
		return nil, err
	}

	var details map[string]any
	err = json.Unmarshal(data, &details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

// TVSeasonInfo is a compact season summary from TMDB.
type TVSeasonInfo struct {
	SeasonNumber  int `json:"season_number"`
	EpisodeCount  int `json:"episode_count"`
	Name          string `json:"name"`
}

// GetTVSeasons returns non-special seasons (season_number >= 1) with episode counts.
func GetTVSeasons(tmdbID int) (name, year, poster string, seasons []TVSeasonInfo, err error) {
	details, err := GetTVDetailsTMDB(tmdbID)
	if err != nil {
		return "", "", "", nil, err
	}

	name = fmt.Sprintf("%v", details["name"])
	if air, ok := details["first_air_date"].(string); ok && len(air) >= 4 {
		year = air[:4]
	}
	if p, ok := details["poster_path"].(string); ok {
		poster = p
	}

	raw, _ := details["seasons"].([]any)
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sn := intFromAny(m["season_number"])
		if sn < 1 {
			continue // skip specials
		}
		ec := intFromAny(m["episode_count"])
		sname, _ := m["name"].(string)
		seasons = append(seasons, TVSeasonInfo{
			SeasonNumber: sn,
			EpisodeCount: ec,
			Name:         sname,
		})
	}
	return name, year, poster, seasons, nil
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

