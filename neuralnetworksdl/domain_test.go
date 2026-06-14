package neuralnetworksdl

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (mint, body, resolve), which need no network.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "neuralnetworksdl" {
		t.Errorf("Scheme = %q, want neuralnetworksdl", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "neuralnetworksdl" {
		t.Errorf("Identity.Binary = %q, want neuralnetworksdl", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"wiki/Go", "page", "wiki/Go"},
		{"/about/", "page", "about"},
		{"http://" + Host + "/chap1.html", "page", "chap1.html"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("page", "chap1.html")
	want := "http://" + Host + "/chap1.html"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	p := &Page{ID: "chap1.html", URL: "http://" + Host + "/chap1.html", Title: "Chapter 1", Body: "Neural nets."}
	u, err := h.Mint(p)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "neuralnetworksdl://page/chap1.html"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	if body, ok := h.Body(p); !ok || body == "" {
		t.Errorf("Body = (%q, %v), want non-empty", body, ok)
	}

	got, err := h.ResolveOn("neuralnetworksdl", "about")
	if err != nil || got.String() != "neuralnetworksdl://page/about" {
		t.Errorf("ResolveOn = (%q, %v), want neuralnetworksdl://page/about", got.String(), err)
	}
}
