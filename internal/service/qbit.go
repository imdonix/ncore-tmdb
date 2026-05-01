package service

import (
	"bytes"
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
	qbitHost string
	qbitUser string
	qbitPass string
	qbitClient *http.Client
)

func InitQbit() {
	qbitHost = os.Getenv("QBIT_HOST")
	qbitUser = os.Getenv("QBIT_USER")
	qbitPass = os.Getenv("QBIT_PASS")

	if qbitHost == "" || qbitUser == "" || qbitPass == "" {
		panic("QBIT_HOST, QBIT_USER, and QBIT_PASS must be set")
	}

	jar, _ := cookiejar.New(nil)
	qbitClient = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

func qbitLogin() error {
	loginURL := fmt.Sprintf("%s/api/v2/auth/login", qbitHost)
	data := url.Values{}
	data.Set("username", qbitUser)
	data.Set("password", qbitPass)

	resp, err := qbitClient.PostForm(loginURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit login failed with status: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "Fails") {
		return fmt.Errorf("qbit login failed: %s", string(body))
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
	_, err = io.Copy(part, bytes.NewReader(torrentData))
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", addURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add torrent: %s - %s", resp.Status, string(respBody))
	}

	return nil
}
