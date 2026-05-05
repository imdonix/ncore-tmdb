package service

import (
	"testing"
)

func TestExtractTMDBIDFromPath(t *testing.T) {
	tests := []struct {
		path            string
		expectedID      string
		expectedType    string
	}{
		{"/movie/12345-somename", "12345", "movie"},
		{"/movie/12345", "12345", "movie"},
		{"/movie/12345/", "12345", "movie"},
		{"/tv/67890-anothername", "67890", "tv"},
		{"/tv/67890", "67890", "tv"},
		{"/tv/67890/", "67890", "tv"},
		{"/person/111", "", ""},
		{"/movie/top-rated", "", ""},
	}

	for _, tt := range tests {
		id, contentType := extractTMDBIDFromPath(tt.path)
		if id != tt.expectedID || contentType != tt.expectedType {
			t.Errorf("extractTMDBIDFromPath(%q) = (%q, %q), want (%q, %q)", tt.path, id, contentType, tt.expectedID, tt.expectedType)
		}
	}
}
