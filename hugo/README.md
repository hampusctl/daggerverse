# hugo

Build a [Hugo](https://gohugo.io) site, check its links, add a search index and package it
for static hosting (GitHub Pages, GitLab Pages, any web server).

| Function | What it does |
|---|---|
| `build` | `hugo` on a source directory; returns the built site (what Hugo writes to `public/`). `--path` when the project is a subdirectory of `--source` (its mounts reach outside it), `--environment` (default `production`), `--base-url`, `--minify` (default on), `--extra-args`, `--npm`. Hugo modules work (the image has Go and git; downloads are cached). |
| `index` | Adds a [Pagefind](https://pagefind.app) search index (`pagefind/`, with Pagefind's UI scripts) to a built site. |
| `site` | `build`, then `index` (`--index=false` skips it); `--npm` as for `build`. |
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

### Node dependencies (`--npm`)

Themes such as [Docsy](https://www.docsy.dev) take Bootstrap, Font Awesome or a search
library from the site's `node_modules`. With `--npm`, `build` (and `site`) first run
`npm ci --ignore-scripts --no-audit --no-fund` in the project directory (`--path`; it needs a
`package-lock.json`), then put its `node_modules/.bin` on `PATH`, after the pinned Dart Sass:
a `sass-embedded` in the lockfile is not the `sass` Hugo runs (its native compiler is chosen
by its own libc detection, which npm's install does not always match on Alpine).

- Node.js and npm are the Hugo image's (Alpine packages), pinned by its digest.
- Packages are checked against the lockfile's `integrity` hashes; lifecycle scripts do not
  run. npm's download cache is kept in the `hugo-cache` volume.
- Hugo resolves a theme module's `node_modules/...` mounts from the project's `node_modules`.

```go
site := h.Build(src, dagger.HugoBuildOpts{Path: "docs/site", Npm: true})
```

Pinned versions (override with `--container` / `--lychee`):

- Hugo extended `v0.168.0`: `ghcr.io/gohugoio/hugo`, by digest, with its Go, git, Node.js
  and npm. It runs as user `hugo`; a replacement container needs a `hugo` user and
  writable `/tmp` (and `npm` on `PATH` for `--npm`).
- Dart Sass `1.105.0`: the linux-musl release archive for the engine's platform (amd64,
  arm64), checked against its SHA-256, first on `PATH` for every build (the image's own Dart
  Sass is older). Hugo's `dartsass` transpiler runs it.
- lychee `0.24.2`: `lycheeverse/lychee:0.24.2-alpine`, by digest.
- Pagefind `v1.5.2`: the linux-musl release archive for the engine's platform (amd64,
  arm64), checked against its SHA-256.

`testdata/site` is a minimal site for trying the functions (no theme).
