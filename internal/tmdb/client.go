package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

var apiKey string
var baseURL = "https://api.themoviedb.org/3"

func Init(key string) {
	apiKey = key
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

	os.WriteFile("a.json", data, 0644)

	return data, nil
}

func GetMovieDetails(tmdbID int) (map[string]any, error) {
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

func GetTVDetails(tmdbID int) (map[string]any, error) {
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
