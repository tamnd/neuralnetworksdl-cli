package neuralnetworksdl

import (
	"context"
	"net/url"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes neuralnetworksdl as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/neuralnetworksdl-cli/neuralnetworksdl"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// neuralnetworksdl:// URIs by routing to the operations Register installs.
func init() { kit.Register(Domain{}) }

// Domain is the neuralnetworksdl driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "neuralnetworksdl",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "neuralnetworksdl",
			Short:  "Browse the Neural Networks and Deep Learning book from the command line.",
			Long: `Browse the Neural Networks and Deep Learning book by Michael Nielsen
from the command line.

neuralnetworksdl reads the public neuralnetworksanddeeplearning.com site over plain
HTTP, shapes it into clean records, and prints output that pipes into the rest
of your tools. No API key, nothing to run alongside it.`,
			Site: Host,
			Repo: "https://github.com/tamnd/neuralnetworksdl-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// chapters: list the book's table of contents.
	kit.Handle(app, kit.OpMeta{Name: "chapters", Group: "read", List: true,
		Summary: "List all chapters of Neural Networks and Deep Learning"},
		listChapters)

	// Resolver op: one record per id.
	kit.Handle(app, kit.OpMeta{Name: "page", Group: "read", Single: true,
		Summary: "Fetch a page by path or URL", URIType: "page", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "page path or URL"}}}, getPage)

	// List op: members of a page.
	kit.Handle(app, kit.OpMeta{Name: "links", Group: "read", List: true,
		Summary: "List the pages a page links to", URIType: "page",
		Args: []kit.Arg{{Name: "ref", Help: "page path or URL"}}}, listLinks)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- inputs ---

type chaptersIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type pageRef struct {
	Ref    string  `kit:"arg" help:"page path or URL"`
	Client *Client `kit:"inject"`
}

type listRef struct {
	Ref    string  `kit:"arg" help:"page path or URL"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listChapters(ctx context.Context, in chaptersIn, emit func(*Chapter) error) error {
	chapters, err := in.Client.Chapters(ctx)
	if err != nil {
		return mapErr(err)
	}
	for i, ch := range chapters {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(ch); err != nil {
			return err
		}
	}
	return nil
}

func getPage(ctx context.Context, in pageRef, emit func(*Page) error) error {
	p, err := in.Client.GetPage(ctx, pagePath(in.Ref))
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

func listLinks(ctx context.Context, in listRef, emit func(*Page) error) error {
	pages, err := in.Client.PageLinks(ctx, pagePath(in.Ref), in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, p := range pages {
		if err := emit(p); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = pagePath(input)
	if id == "" {
		return "", "", errs.Usage("unrecognized neuralnetworksdl reference: %q", input)
	}
	return "page", id, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "page" {
		return "", errs.Usage("neuralnetworksdl has no resource type %q", uriType)
	}
	return BaseURL + "/" + strings.Trim(id, "/"), nil
}

// --- helpers ---

func pagePath(input string) string {
	input = strings.TrimSpace(input)
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return strings.Trim(u.Path, "/")
	}
	return strings.Trim(input, "/")
}

func mapErr(err error) error {
	return err
}
