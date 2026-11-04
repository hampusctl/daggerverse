# hugo

Build a [Hugo](https://gohugo.io) site, check its links, add a search index and package it
for static hosting (GitHub Pages, GitLab Pages, any web server).

| Function | What it does |
|---|---|
| `build` | `hugo` on a source directory; returns the built site (what Hugo writes to `public/`). `--environment` (default `production`), `--base-url`, `--minify` (default on), `--extra-args`. Hugo modules work (the image has Go and git; downloads are cached). |
| `index` | Adds a [Pagefind](https://pagefind.app) search index (`pagefind/`, with Pagefind's UI scripts) to a built site. |
| `site` | `build`, then `index` (`--index=false` skips it). |
| `linkcheck` | Checks the built site's links with [lychee](https://lychee.cli.rs): internal links and `#anchors` always (offline), external links with `--external`. `--base-path` for a site served under a sub-path, `--exclude` for URL patterns. Fails on a broken link. |
| `archive` | The site as a `.tar.gz`, files under `public/` (`--prefix`; empty for none). |

```bash
# Build and index the example site, then export it
dagger -m github.com/hampusctl/daggerverse/hugo call site --source=hugo/testdata/site export --path=public

# Check its links (internal only, no network)
dagger -m github.com/hampusctl/daggerverse/hugo call linkcheck --site=./public

# GitLab Pages: export public/ in the job; GitHub Pages: export the directory and pass it to
# actions/upload-pages-artifact.
dagger -m github.com/hampusctl/daggerverse/hugo call site --source=docs export --path=public
dagger -m github.com/hampusctl/daggerverse/hugo call archive --site=./public export --path=site.tar.gz
```

From another module (`dagger install github.com/hampusctl/daggerverse/hugo@<commit>`):

```go
h := dag.Hugo()
site := h.Build(src, dagger.HugoBuildOpts{BaseURL: "/docs/"})
report, err := h.Linkcheck(ctx, site, dagger.HugoLinkcheckOpts{BasePath: "/docs/"})
```

Pinned versions (override with `--container` / `--lychee`):

- Hugo extended `v0.168.0`: `ghcr.io/gohugoio/hugo`, by digest. It runs as user `hugo`; a
  replacement container needs a `hugo` user and writable `/tmp`.
- lychee `0.24.2`: `lycheeverse/lychee:0.24.2-alpine`, by digest.
- Pagefind `v1.5.2`: the linux-musl release archive for the engine's platform (amd64,
  arm64), checked against its SHA-256.

`testdata/site` is a minimal site for trying the functions (no theme).
