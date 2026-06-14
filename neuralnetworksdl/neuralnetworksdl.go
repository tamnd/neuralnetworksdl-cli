// Package neuralnetworksdl is the library behind the neuralnetworksdl command line:
// the HTTP client, request shaping, and the typed data models for
// neuralnetworksanddeeplearning.com.
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public site throws under load.
package neuralnetworksdl

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DefaultUserAgent identifies the client to neuralnetworksanddeeplearning.com.
const DefaultUserAgent = "neuralnetworksdl/dev (+https://github.com/tamnd/neuralnetworksdl-cli)"

// Host is the site this client talks to.
const Host = "neuralnetworksanddeeplearning.com"

// BaseURL is the root every request is built from.
const BaseURL = "http://" + Host

// Client talks to neuralnetworksanddeeplearning.com over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	last time.Time
}

// NewClient returns a Client with sensible defaults: a 30s timeout, a 200ms
// minimum gap between requests, and five retries on transient errors.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   5,
	}
}

// Get fetches url and returns the response body. It paces and retries according
// to the client's settings. The caller owns nothing extra; the body is read
// fully and closed here.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}

// Page is the scaffold's one example record: a single page, addressed by the
// path that names it on neuralnetworksanddeeplearning.com.
type Page struct {
	ID    string `json:"id" kit:"id"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty" kit:"body"`
}

// GetPage fetches one page by its path and returns it as a record.
func (c *Client) GetPage(ctx context.Context, path string) (*Page, error) {
	path = strings.Trim(path, "/")
	url := BaseURL + "/" + path
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Page{ID: path, URL: url, Title: path, Body: pageText(body)}, nil
}

// PageLinks fetches a page and returns the same-host pages it links to, as
// page stubs.
func (c *Client) PageLinks(ctx context.Context, path string, limit int) ([]*Page, error) {
	path = strings.Trim(path, "/")
	body, err := c.Get(ctx, BaseURL+"/"+path)
	if err != nil {
		return nil, err
	}
	var out []*Page
	seen := map[string]bool{}
	for _, p := range linkPaths(body) {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, &Page{ID: p, URL: BaseURL + "/" + p})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Chapter is one entry in the Neural Networks and Deep Learning table of
// contents.
type Chapter struct {
	Rank   int    `json:"rank"   kit:"rank"`
	Number string `json:"number" kit:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

var (
	mainChapRe = regexp.MustCompile(`(?s)class=['"]toc_mainchapter['"][^>]*>.*?<a href="(chap\d+\.html)"[^>]*>([^<]+)</a>`)
	appendixRe = regexp.MustCompile(`<a href="(sai\.html)"[^>]*>((?:[^<]|<em>[^<]*</em>)+)</a>`)
	tagRE      = regexp.MustCompile(`<[^>]+>`)
)

// Chapters fetches the homepage and returns the full table of contents.
func (c *Client) Chapters(ctx context.Context) ([]*Chapter, error) {
	body, err := c.Get(ctx, BaseURL+"/")
	if err != nil {
		return nil, err
	}
	pageHTML := string(body)
	var chapters []*Chapter
	rank := 1

	for _, m := range mainChapRe.FindAllStringSubmatch(pageHTML, -1) {
		href := m[1]
		title := html.UnescapeString(strings.TrimSpace(m[2]))
		chapters = append(chapters, &Chapter{
			Rank:   rank,
			Number: chapNum(href),
			Title:  title,
			URL:    BaseURL + "/" + href,
		})
		rank++
	}

	if m := appendixRe.FindStringSubmatch(pageHTML); m != nil {
		href := m[1]
		title := html.UnescapeString(strings.TrimSpace(tagRE.ReplaceAllString(m[2], "")))
		chapters = append(chapters, &Chapter{
			Rank:   rank,
			Number: "A",
			Title:  title,
			URL:    BaseURL + "/" + href,
		})
	}

	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found")
	}
	return chapters, nil
}

// chapNum converts a filename like "chap1.html" to "1", and "sai.html" to "A".
func chapNum(href string) string {
	href = strings.TrimSuffix(href, ".html")
	href = strings.TrimPrefix(href, "chap")
	if href == "sai" {
		return "A"
	}
	return href
}

var (
	hrefRE = regexp.MustCompile(`href="(/[^":#?]+)"`)
)

// linkPaths pulls the relative link targets out of an HTML response.
func linkPaths(body []byte) []string {
	var out []string
	for _, m := range hrefRE.FindAllSubmatch(body, -1) {
		if p := strings.Trim(string(m[1]), "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// pageText reduces an HTML response to a short plain-text preview.
func pageText(body []byte) string {
	s := strings.Join(strings.Fields(tagRE.ReplaceAllString(string(body), " ")), " ")
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}
