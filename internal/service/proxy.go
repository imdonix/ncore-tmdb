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
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/static"
)

var tmdbURL *url.URL
var widgetSnippet string

var (
	movieRe = regexp.MustCompile(`/movie/(\d+)(?:-|/|$)`)
	tvRe    = regexp.MustCompile(`/tv/(\d+)(?:-|/|$)`)
)

// Header link to the embedded NCore SPA (site chrome only).
const ncoreHeaderBtn = `<a href="/ncore" id="ncore-header-btn" style="display:inline-flex;align-items:center;margin-left:12px;padding:6px 12px;border-radius:8px;background:#01d277;color:#032541;font:600 13px/1 system-ui,sans-serif;text-decoration:none;white-space:nowrap;vertical-align:middle;z-index:10000;position:relative;">NCore Dashboard</a>`

// Kill TMDB (or any) service workers that hijack same-origin routes like /ncore.
const swKillScript = `<script id="ncore-sw-kill">
(function () {
  try {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.getRegistrations().then(function (regs) {
        regs.forEach(function (r) { r.unregister(); });
      });
    }
    if (window.caches && caches.keys) {
      caches.keys().then(function (keys) {
        keys.forEach(function (k) { caches.delete(k); });
      });
    }
  } catch (e) {}
})();
</script>`

// Service worker that immediately unregisters itself (replaces TMDB's /sw.js).
const killSwitchSW = `/* ncore-tmdb: neutralize site service workers */
self.addEventListener('install', function (e) { self.skipWaiting(); });
self.addEventListener('activate', function (e) {
  e.waitUntil((async function () {
    try {
      var keys = await caches.keys();
      await Promise.all(keys.map(function (k) { return caches.delete(k); }));
    } catch (err) {}
    try { await self.registration.unregister(); } catch (err) {}
    try {
      var clients = await self.clients.matchAll({ type: 'window' });
      clients.forEach(function (c) { c.navigate(c.url); });
    } catch (err) {}
  })());
});
self.addEventListener('fetch', function (e) {
  // Never intercept — pass through to network
  return;
});
`

func InitProxy() {
	var err error
	tmdbURL, err = url.Parse("https://www.themoviedb.org")
	if err != nil {
		log.Fatal(err)
	}

	content, err := static.WidgetSnippet()
	if err != nil {
		log.Printf("Warning: widget snippet not found (run make): %v", err)
		widgetSnippet = ""
		return
	}
	widgetSnippet = string(content)
}

// SetupProxy reverse-proxies unmatched routes to TMDB.
func SetupProxy(r *gin.Engine) {
	// Own /sw.js so TMDB's service worker cannot control this origin
	r.GET("/sw.js", serveKillSwitchSW)
	r.GET("/service-worker.js", serveKillSwitchSW)
	r.GET("/serviceworker.js", serveKillSwitchSW)

	proxy := httputil.NewSingleHostReverseProxy(tmdbURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = tmdbURL.Host
		req.Header.Set("Host", tmdbURL.Host)
		req.Header.Set("X-Forwarded-Host", req.Host)
		// Prefer gzip so HTML injection can decode reliably
		req.Header.Set("Accept-Encoding", "gzip")
	}

	proxy.ModifyResponse = modifyResponse
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("TMDB proxy error %s: %v", req.URL.RequestURI(), err)
		http.Error(w, "upstream error", http.StatusBadGateway)
	}

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

func serveKillSwitchSW(c *gin.Context) {
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Service-Worker-Allowed", "/")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.String(http.StatusOK, killSwitchSW)
}

// ServeSPA writes the embedded NCore SPA index.html (with SW kill script).
func ServeSPA(c *gin.Context) {
	index, err := static.WebappIndex()
	if err != nil {
		c.String(http.StatusServiceUnavailable, "Webapp not built. Run: make")
		return
	}
	html := string(index)
	if !strings.Contains(html, "ncore-sw-kill") {
		html = strings.Replace(html, "<head>", "<head>"+swKillScript, 1)
	}
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func modifyResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	var reader io.Reader = resp.Body
	enc := resp.Header.Get("Content-Encoding")

	if strings.Contains(enc, "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
		resp.Header.Del("Content-Encoding")
	} else if enc != "" && enc != "identity" {
		return nil
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	path := ""
	if resp.Request != nil && resp.Request.URL != nil {
		path = resp.Request.URL.Path
	}
	tmdbID, pageType := extractTMDBIDFromPath(path)
	modifiedBody := modifyContent(body, path, tmdbID, pageType)

	resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	resp.ContentLength = int64(len(modifiedBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))
	resp.Header.Del("Content-Encoding")
	// Discourage caching of rewritten HTML (and SW control)
	resp.Header.Set("Cache-Control", "no-cache")

	return nil
}

func modifyContent(body []byte, path, tmdbID, pageType string) []byte {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return body
	}

	// Original stripping: drop auth chrome in header, footer, community block
	doc.Find("div.flex").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("header, #header").Length() > 0 {
			s.Remove()
		}
	})
	doc.Find("footer").Remove()
	doc.Find("section.inner_content.bg_image.community").Remove()

	// Cookie / consent popups (OneTrust / CookieLaw used by TMDB)
	stripCookieConsent(doc)

	// Prevent TMDB from re-registering a service worker on this origin
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		txt := s.Text()
		if strings.Contains(txt, "serviceWorker") && strings.Contains(txt, "register") {
			s.SetHtml("/* service worker registration disabled by ncore-tmdb */")
		}
	})

	// Kill any existing SW on page load
	if doc.Find("#ncore-sw-kill").Length() == 0 {
		doc.Find("head").PrependHtml(swKillScript)
	}

	// Header button only in the real site chrome — never in page content /
	// popular-movie lists (body prepend used to dump it into the main column).
	injectDownloadDashboardBtn(doc, path)

	// Torrent widget on movie detail pages only
	if tmdbID != "" && pageType == "movie" && widgetSnippet != "" && doc.Find("#ncore-widget-root").Length() == 0 {
		w := widgetSnippet
		w = strings.ReplaceAll(w, "#CONTENT_TMDBID#", tmdbID)
		w = strings.ReplaceAll(w, "#CONTENT_TYPE#", pageType)
		col := doc.Find("div#media_v4 div.white_column").First()
		if col.Length() > 0 {
			col.PrependHtml(w)
		}
	}

	html, err := doc.Html()
	if err != nil {
		return body
	}

	// Extra safety: neutralize register calls that survived as external bundles won't,
	// but inline ones we rewrote. Also block common patterns in attributes.
	out := html
	out = strings.ReplaceAll(out, "navigator.serviceWorker.register", "Promise.resolve.bind(Promise)/*sw-disabled*/")
	out = strings.ReplaceAll(out, "serviceWorker.register", "/*sw-disabled*/")

	return []byte(out)
}

// listingPage is a TMDB browse/list URL (e.g. /movie popular list), not a detail page.
func listingPage(path string) bool {
	p := strings.TrimSuffix(path, "/")
	if p == "" {
		return false
	}
	// Exact browse roots
	switch p {
	case "/movie", "/tv", "/person":
		return true
	}
	// /movie/now-playing, /tv/top-rated, etc. — no numeric id
	if movieRe.MatchString(path) || tvRe.MatchString(path) {
		return false // detail page with id
	}
	if strings.HasPrefix(p, "/movie/") || strings.HasPrefix(p, "/tv/") || strings.HasPrefix(p, "/person/") {
		return true
	}
	return false
}

func injectDownloadDashboardBtn(doc *goquery.Document, path string) {
	// Remove any previous misplaced instances first
	doc.Find("#ncore-header-btn").Each(func(i int, s *goquery.Selection) {
		// Keep only if already inside primary site header; drop content clones
		if s.ParentsFiltered("#header").Length() == 0 && s.ParentsFiltered("header").Length() == 0 {
			s.Remove()
		} else if s.ParentsFiltered("main, #main, #media_v4, .white_column, .page_wrapper .content").Length() > 0 {
			// Button nested under content wrappers incorrectly
			if s.ParentsFiltered("#header").Length() == 0 {
				s.Remove()
			}
		}
	})

	// Do not inject on browse/list pages (popular movies, etc.)
	if listingPage(path) {
		doc.Find("#ncore-header-btn").Remove()
		return
	}

	if doc.Find("#header #ncore-header-btn, header #ncore-header-btn").Length() > 0 {
		return
	}

	// Primary TMDB chrome only
	siteHeader := doc.Find("#header").First()
	if siteHeader.Length() == 0 {
		// Fallback: top-level <header>, not ones inside article/main
		doc.Find("header").Each(func(i int, s *goquery.Selection) {
			if siteHeader.Length() > 0 {
				return
			}
			if s.ParentsFiltered("main, article, #media_v4, .white_column").Length() == 0 {
				siteHeader = s
			}
		})
	}
	if siteHeader.Length() == 0 {
		return
	}

	nav := siteHeader.Find("ul.dropdown_menu").First()
	if nav.Length() > 0 {
		nav.AppendHtml(`<li class="ncore-nav-item" style="display:flex;align-items:center;list-style:none;">` + ncoreHeaderBtn + `</li>`)
		return
	}
	// Sub-nav / media row inside chrome
	if sub := siteHeader.Find(".sub_media, .nav_wrapper, .content").First(); sub.Length() > 0 {
		sub.AppendHtml(ncoreHeaderBtn)
		return
	}
	siteHeader.AppendHtml(ncoreHeaderBtn)
}

func extractTMDBIDFromPath(path string) (string, string) {
	if m := movieRe.FindStringSubmatch(path); len(m) > 1 {
		return m[1], "movie"
	}
	if m := tvRe.FindStringSubmatch(path); len(m) > 1 {
		return m[1], "tv"
	}
	return "", ""
}

// stripCookieConsent removes OneTrust/CookieLaw banners and loaders so the
// "Accept cookies" popup never appears behind the proxy.
func stripCookieConsent(doc *goquery.Document) {
	// Banner / modal containers
	selectors := []string{
		"#onetrust-banner-sdk",
		"#onetrust-consent-sdk",
		"#onetrust-pc-sdk",
		"#ot-sdk-btn-floating",
		".onetrust-pc-dark-filter",
		".ot-sdk-container",
		"#ot-sdk-btn",
		"[id*='onetrust']",
		"[class*='onetrust']",
		"[id*='cookie-banner']",
		"[class*='cookie-banner']",
		"[id*='cookie_banner']",
		"[class*='cookie_consent']",
		"[id*='cookie-consent']",
	}
	for _, sel := range selectors {
		doc.Find(sel).Remove()
	}

	// External consent scripts (cookielaw / onetrust / optanon)
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		txt := s.Text()
		blob := strings.ToLower(src + " " + txt)
		if strings.Contains(blob, "cookielaw") ||
			strings.Contains(blob, "onetrust") ||
			strings.Contains(blob, "optanon") ||
			strings.Contains(blob, "otSDKStub") ||
			strings.Contains(blob, "ot-sdk") {
			s.Remove()
		}
	})

	// Linked consent stylesheets
	doc.Find("link[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		h := strings.ToLower(href)
		if strings.Contains(h, "cookielaw") || strings.Contains(h, "onetrust") {
			s.Remove()
		}
	})

	// Hide anything that still gets injected client-side
	if doc.Find("#ncore-cookie-hide").Length() == 0 {
		doc.Find("head").AppendHtml(`<style id="ncore-cookie-hide">
#onetrust-banner-sdk,
#onetrust-consent-sdk,
#onetrust-pc-sdk,
#ot-sdk-btn-floating,
.onetrust-pc-dark-filter,
.ot-sdk-container,
#ot-sdk-btn,
[id*="onetrust"],
[class*="onetrust"] {
  display: none !important;
  visibility: hidden !important;
  pointer-events: none !important;
  opacity: 0 !important;
  max-height: 0 !important;
  overflow: hidden !important;
}
</style>`)
	}
}
