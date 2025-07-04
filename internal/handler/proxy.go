package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/steamcmd"
	"github.com/gin-gonic/gin"
)

var (
	infoRegex = regexp.MustCompile(`(?i)(?:SubscribeItem|SubscribeCollection|SubscribeCollectionItem)\(\s*'(\d+)',\s*'(\d+)'\s*\);`)
)

type SteamDownloaderAPI struct {
	steamcmd      *steamcmd.SteamCMD
	saveDirectory string
}

func New(s *steamcmd.SteamCMD) *SteamDownloaderAPI {
	temp, err := os.MkdirTemp("", "steam-downloader-")
	if err != nil {
		panic(err)
	}
	return &SteamDownloaderAPI{steamcmd: s, saveDirectory: temp}
}

func (h *SteamDownloaderAPI) UnsupportedPageHandler(c *gin.Context) {
	c.Data(http.StatusNotImplemented, "text/html; charset=utf-8", []byte("We don't support this page. <a href='/'>Back</a>"))
}

func (h *SteamDownloaderAPI) SteamProxyHandler(c *gin.Context) {
	remote, _ := url.Parse("https://steamcommunity.com")
	proxy := httputil.NewSingleHostReverseProxy(remote)

	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host

		req.URL = c.Request.URL
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host

		req.Header.Del("Accept-Encoding")
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		defer res.Body.Close()

		if !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
			body = bytes.ReplaceAll(body,
				[]byte("https://steamcommunity.com/workshop/ajaxfindworkshops"),
				[]byte("/workshop/ajaxfindworkshops"),
			)

			res.Body = io.NopCloser(bytes.NewReader(body))
			return nil
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		titleElm := doc.Find("title")
		title := titleElm.Text()
		title = strings.Replace(title, "Steam Community", "SteamDownloaderAPI", 1)
		titleElm.SetText(title)

		replacedCollection := true

		doc.Find("a[onclick*='SubscribeItem']").Each(func(i int, s *goquery.Selection) {
			onclick, _ := s.Attr("onclick")
			if matches := infoRegex.FindStringSubmatch(onclick); len(matches) == 3 {
				workshopID, appID := matches[1], matches[2]
				dlBtn := fmt.Sprintf(`<div><a href="/api/workshop/%s/%s" class="btn_darkred_white_innerfade btn_border_2px btn_medium" style="position: relative"> <div class="followIcon"></div> <span class="subscribeText"> <div>Download</div> </span> </a> </div>`, appID, workshopID)
				s.Parent().Parent().AppendHtml(dlBtn)
			}
		})

		doc.Find(`span[class="valve_links"]`).Each(func(i int, s *goquery.Selection) {
			s.AppendHtml(` |  <a style="color: #1497cb;font-weight: bold;font-size: medium;" href="https://github.com/SyNdicateFoundation/SteamDownloaderAPI" target="_blank">SteamDownloaderAPI GitHub</a>`)
		})

		doc.Find(".subscribe[onclick*='SubscribeCollection']").Each(func(i int, s *goquery.Selection) {
			onclick, _ := s.Attr("onclick")
			if matches := infoRegex.FindStringSubmatch(onclick); len(matches) == 3 {
				collectionID, appID := matches[1], matches[2]

				var dlBtn string

				if replacedCollection {
					dlBtn = fmt.Sprintf(`<a class="general_btn subscribe" style="background: #640000; color: white;display:table" href="/api/collection/%s/%s"><div class="followIcon"></div> <span class="subscribeText">Download Collection</span> </a>`, appID, collectionID)
				} else {
					dlBtn = fmt.Sprintf(`<div><a href="/api/workshop/%s/%s" class="btn_darkred_white_innerfade btn_medium" style="position: relative"> <div class="followIcon"></div> <span class="subscribeText"> <div>Download</div> </span> </a> </div>`, appID, collectionID)
				}

				replacedCollection = false

				s.Parent().AppendHtml(dlBtn)
			}
		})

		rewriteURL := func(rawURL string) string {
			if rawURL == "" || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(rawURL, "mailto:") || strings.HasPrefix(rawURL, "javascript:") {
				return rawURL
			}

			if !strings.Contains(rawURL, "steamcommunity.com") && !strings.Contains(rawURL, "akamai.steamstatic.com") {
				return rawURL
			}

			hasTopLocationHref := false

			if strings.HasPrefix(rawURL, "top.location.href='") {
				rawURL = strings.TrimPrefix(rawURL, "top.location.href='")
				rawURL = strings.TrimSuffix(rawURL, "'")
				hasTopLocationHref = true
			}

			parsedURL, err := url.Parse(rawURL)
			if err != nil {
				return rawURL
			}

			if parsedURL.Host == "" {
				if hasTopLocationHref {
					return "top.location.href='" + rawURL + "'"
				}
				return rawURL
			}

			if parsedURL.Host == "steamcommunity.com" || parsedURL.Host == "community.akamai.steamstatic.com" {
				if hasTopLocationHref {
					return "top.location.href='" + parsedURL.RequestURI() + "'"
				}

				return parsedURL.RequestURI()
			}

			if hasTopLocationHref {
				return "top.location.href='" + rawURL + "'"
			}

			return rawURL
		}

		doc.Find("[href]").Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr("href"); exists {
				s.SetAttr("href", rewriteURL(val))
			}
		})

		doc.Find("[onclick]").Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr("onclick"); exists {
				s.SetAttr("onclick", rewriteURL(val))
			}
		})

		doc.Find("[src]").Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr("src"); exists {
				s.SetAttr("src", rewriteURL(val))
			}
		})

		doc.Find("[srcset]").Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr("srcset"); exists {
				var newSrcset []string
				for _, part := range strings.Split(val, ",") {
					trimmed := strings.TrimSpace(part)
					urlAndDescriptor := strings.Fields(trimmed)
					if len(urlAndDescriptor) > 0 {
						urlAndDescriptor[0] = rewriteURL(urlAndDescriptor[0])
						newSrcset = append(newSrcset, strings.Join(urlAndDescriptor, " "))
					}
				}
				s.SetAttr("srcset", strings.Join(newSrcset, ", "))
			}
		})

		html, err := doc.Html()
		if err != nil {
			return fmt.Errorf("failed to render modified HTML: %w", err)
		}

		if loc, err := res.Location(); err == nil {
			res.Header.Set("Location", rewriteURL(loc.String()))
		}

		res.Body = io.NopCloser(strings.NewReader(html))
		res.Header["Content-Length"] = []string{fmt.Sprint(len(html))}
		res.Header.Del("Content-Security-Policy")
		res.Header.Del("X-Frame-Options")
		res.Header.Del("Content-Encoding")
		res.Header.Del("Access-Control-Allow-Origin")

		return nil
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func (h *SteamDownloaderAPI) Cleanup() {
	_ = os.RemoveAll(h.saveDirectory)
}
