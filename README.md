# neuralnetworksdl

Browse the [Neural Networks and Deep Learning](http://neuralnetworksanddeeplearning.com/) book
by Michael Nielsen from the command line.

`neuralnetworksdl` is a single pure-Go binary. It reads public data from
neuralnetworksanddeeplearning.com over plain HTTP, shapes it into clean records,
and prints output that pipes into the rest of your tools. No API key, nothing to
run alongside it.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
neuralnetworksdl as `neuralnetworksdl://` URIs.

## Install

```bash
go install github.com/tamnd/neuralnetworksdl-cli/cmd/neuralnetworksdl@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/neuralnetworksdl-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/neuralnetworksdl:latest --help
```

## Usage

```bash
neuralnetworksdl chapters                       # list all 6 chapters + appendix
neuralnetworksdl chapters -o json               # as JSON, ready for jq
neuralnetworksdl chapters -o csv                # CSV with header row
neuralnetworksdl chapters -n 3                  # first 3 chapters only
neuralnetworksdl chapters --fields number,title # pick columns
neuralnetworksdl page <path>                    # fetch one page as a record
neuralnetworksdl links <path>                   # the pages it links to
neuralnetworksdl --help                         # the whole command tree
```

Every command shares one output contract: `-o table|json|jsonl|csv|tsv|url|raw`,
`--fields` to pick columns, `--template` for a custom line, and `-n` to limit.
The default adapts to where output goes (a table on a terminal, JSONL in a
pipe), so the same command reads well by hand and parses cleanly downstream.

## Example

```
$ neuralnetworksdl chapters -o table
RANK  NUMBER  TITLE                                                     URL
1     1       Using neural nets to recognize handwritten digits         http://neuralnetworksanddeeplearning.com/chap1.html
2     2       How the backpropagation algorithm works                   http://neuralnetworksanddeeplearning.com/chap2.html
3     3       Improving the way neural networks learn                   http://neuralnetworksanddeeplearning.com/chap3.html
4     4       A visual proof that neural nets can compute any function  http://neuralnetworksanddeeplearning.com/chap4.html
5     5       Why are deep neural networks hard to train?               http://neuralnetworksanddeeplearning.com/chap5.html
6     6       Deep learning                                             http://neuralnetworksanddeeplearning.com/chap6.html
7     A       Appendix: Is there a simple algorithm for intelligence?   http://neuralnetworksanddeeplearning.com/sai.html
```

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
neuralnetworksdl serve --addr :7777    # GET /v1/chapters  returns NDJSON
neuralnetworksdl mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`neuralnetworksdl` registers a `neuralnetworksdl` domain the way a program registers a
database driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/neuralnetworksdl-cli/neuralnetworksdl"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `neuralnetworksdl://` URIs without knowing anything about the site:

```bash
ant get neuralnetworksdl://page/<path>   # fetch the record
ant cat neuralnetworksdl://page/<path>   # just the body text
ant ls  neuralnetworksdl://page/<path>   # the pages it links to, each addressable
ant url neuralnetworksdl://page/<path>   # the live http URL
```

## Development

```
cmd/neuralnetworksdl/   thin main: hands cli.NewApp to kit.Run
cli/                 assembles the kit App from the neuralnetworksdl domain
neuralnetworksdl/    the library: HTTP client, data models, and domain.go (the driver)
docs/                tago documentation site
```

```bash
make build      # ./bin/neuralnetworksdl
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
