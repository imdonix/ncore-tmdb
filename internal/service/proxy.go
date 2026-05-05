package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

var tmdbURL *url.URL
var widgetContent string

var (
	movieRe = regexp.MustCompile(`/movie/(\d+)(?:-|/|$)`)
	tvRe    = regexp.MustCompile(`/tv/(\d+)(?:-|/|$)`)
)

func InitProxy() {
	var err error
	tmdbURL, err = url.Parse("https://www.themoviedb.org")
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile("widget/widget.html")
	if err != nil {
		panic(fmt.Sprintf("failed to load widget: %v", err))
	}
	widgetContent = string(content)
}

func SetupProxy(r *gin.Engine) {
	proxy := httputil.NewSingleHostReverseProxy(tmdbURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = tmdbURL.Host
		req.Header.Set("Host", tmdbURL.Host)
    	req.Header.Set("X-Forwarded-Host", req.Host)
	}

	proxy.ModifyResponse = modifyResponse

	r.NoRoute(func(c *gin.Context) {
		proxyPath := c.Request.URL.Path
		if proxyPath == "" {
			proxyPath = "/"
		}

		c.Request.URL.Path = proxyPath
		c.Request.Host = tmdbURL.Host

		proxy.ServeHTTP(c.Writer, c.Request)
	})
}

func modifyResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil
	}

	var reader io.Reader = resp.Body

	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
		resp.Header.Del("Content-Encoding")
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	resp.Body.Close()

	tmdbID, contentType := extractTMDBIDFromPath(resp.Request.URL.Path)
	modifiedBody := modifyContent(body, tmdbID, contentType)

	resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	resp.ContentLength = int64(len(modifiedBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))

	return nil
}

func modifyContent(body []byte, tmdbID string, contentType string) []byte {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return body
	}

	doc.Find("div.flex").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("header").Length() > 0 {
			s.Remove()
		}
	})

	doc.Find("footer").Remove()
	doc.Find("section.inner_content.bg_image.community").Remove()

	if tmdbID != "" && contentType != "" && contentType != "tv" {
		w := widgetContent
		w = strings.ReplaceAll(w, "#CONTENT_TMDBID#", tmdbID)
		w = strings.ReplaceAll(w, "#CONTENT_TYPE#", contentType)
		doc.Find("div#media_v4").Find("div.white_column").PrependHtml(w)
	}

	html, err := doc.Html()
	if err != nil {
		return body
	}

	return []byte(html)
}

func extractTMDBIDFromPath(path string) (string, string) {
	movieMatches := movieRe.FindStringSubmatch(path)
	if len(movieMatches) > 1 {
		return movieMatches[1], "movie"
	}

	tvMatches := tvRe.FindStringSubmatch(path)
	if len(tvMatches) > 1 {
		return tvMatches[1], "tv"
	}

	return "", ""
}
