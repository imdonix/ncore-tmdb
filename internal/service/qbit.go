package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	qbitHost   string
	qbitUser   string
	qbitPass   string
	qbitClient *http.Client
)

func InitQbit() {
	qbitHost = strings.TrimRight(os.Getenv("QBIT_HOST"), "/")
	qbitUser = os.Getenv("QBIT_USER")
	qbitPass = os.Getenv("QBIT_PASS")

	if qbitHost == "" {
		panic("QBIT_HOST must be set")
	}

	jar, _ := cookiejar.New(nil)
	qbitClient = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

// qbitOK treats 2xx (including 204 No Content) as success.
// Newer qBittorrent builds return 204 on login instead of 200 "Ok.".
func qbitOK(code int) bool {
	return code >= 200 && code < 300
}

func qbitLogin() error {
	loginURL := fmt.Sprintf("%s/api/v2/auth/login", qbitHost)
	data := url.Values{}

	if qbitUser != "" {
		data.Set("username", qbitUser)
	}
	if qbitPass != "" {
		data.Set("password", qbitPass)
	}

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Recent qBittorrent versions require a Referer for CSRF checks
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))

	if !qbitOK(resp.StatusCode) {
		return fmt.Errorf("qbit login failed with status: %s (%s)", resp.Status, bodyStr)
	}
	if strings.EqualFold(bodyStr, "Fails.") || strings.Contains(bodyStr, "Fails") {
		return fmt.Errorf("qbit login failed: %s", bodyStr)
	}

	return nil
}

func AddTorrent(torrentData []byte, filename string) error {
	if err := qbitLogin(); err != nil {
		return fmt.Errorf("failed to login to qbit: %w", err)
	}

	addURL := fmt.Sprintf("%s/api/v2/torrents/add", qbitHost)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("torrents", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(torrentData)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, addURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if !qbitOK(resp.StatusCode) {
		return fmt.Errorf("failed to add torrent: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

type QbitTorrent struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	Progress float64 `json:"progress"`
	Status   string  `json:"state"`
}

func GetTorrentsStatus() ([]QbitTorrent, error) {
	if err := qbitLogin(); err != nil {
		return nil, err
	}

	infoURL := fmt.Sprintf("%s/api/v2/torrents/info", qbitHost)
	req, err := http.NewRequest(http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !qbitOK(resp.StatusCode) {
		return nil, fmt.Errorf("failed to get torrents info: %s", resp.Status)
	}

	var torrents []QbitTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, err
	}

	return torrents, nil
}
