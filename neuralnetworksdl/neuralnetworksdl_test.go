package neuralnetworksdl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0 // no pacing in the test

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

const fakeHTML = `<html><body>
<p class='toc_mainchapter'><a href="chap1.html">Using neural nets to recognize handwritten digits</a></p>
<p class='toc_mainchapter'><a href="chap2.html">How the backpropagation algorithm works</a></p>
<p class="toc_not_mainchapter"><a href="sai.html">Appendix: Is there a <em>simple</em> algorithm for intelligence?</a></p>
</body></html>`

// proxyTransport redirects all requests to target, preserving path and query.
type proxyTransport struct {
	target *url.URL
}

func (pt *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	r2.URL.Scheme = pt.target.Scheme
	r2.URL.Host = pt.target.Host
	return http.DefaultTransport.RoundTrip(r2)
}

func TestChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fakeHTML)
	}))
	defer srv.Close()

	target, _ := url.Parse(srv.URL)

	c := NewClient()
	c.Rate = 0
	c.HTTP = &http.Client{
		Timeout:   5 * time.Second,
		Transport: &proxyTransport{target: target},
	}

	chapters, err := c.Chapters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 3 {
		t.Fatalf("want 3, got %d", len(chapters))
	}
	if chapters[0].Number != "1" {
		t.Errorf("Number[0] = %q, want 1", chapters[0].Number)
	}
	if chapters[0].Title != "Using neural nets to recognize handwritten digits" {
		t.Errorf("Title[0] = %q", chapters[0].Title)
	}
	if chapters[2].Number != "A" {
		t.Errorf("Number[2] = %q, want A", chapters[2].Number)
	}
	if chapters[0].Rank != 1 {
		t.Errorf("Rank = %d, want 1", chapters[0].Rank)
	}
}
