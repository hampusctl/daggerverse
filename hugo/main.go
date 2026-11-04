// Build, link-check, search-index and package static sites made with Hugo.
//
// Build runs Hugo (extended edition, official image pinned by digest) on a site's source
// directory and returns the built site (Hugo's public/ directory); with npm it first installs
// the site's Node dependencies from its package-lock.json (themes such as Docsy take
// Bootstrap and Font Awesome from node_modules). Sass is compiled with a pinned Dart Sass
// release. Index adds a Pagefind
// search index, Linkcheck checks the built site's links with lychee, and Archive packs the
// site as a tarball. The built directory is what GitHub Pages and GitLab Pages publish.
package main

import (
	"context"
	"dagger/hugo/internal/dagger"
	"fmt"
	pathpkg "path"
	"strings"
)

const (
	// hugoImage is the official image: Hugo extended, Go (for Hugo modules), git, Dart Sass,
	// Node.js and npm (Alpine packages, so pinned by the digest too).
	hugoImage = "ghcr.io/gohugoio/hugo:v0.168.0@sha256:1fb8fe7f417abd5802d86747d297728aa48078d763159e7dff6559fe3f8bb8e7"
	// lycheeImage is lychee's official Alpine image.
	lycheeImage = "lycheeverse/lychee:0.24.2-alpine@sha256:2255c0b916cc8fc4193f59a4549358a6bae0f7d4ea16b5e1dd4c3b2733c35504"
	// pagefindVersion is the Pagefind release Index downloads; pagefindSHA256 pins its
	// linux-musl archives by architecture (the release's .sha256 files).
	pagefindVersion = "v1.5.2"

	// dartSassVersion is the Dart Sass release Build puts first on PATH (Hugo's dartsass
	// transpiler runs `sass`); dartSassSHA256 pins its linux-musl archives (the image is
	// Alpine) by architecture.
	dartSassVersion = "1.105.0"
	dartSassDir     = "/tmp/dart-sass"

	src = "/src"
	// siteRoot is where Linkcheck mounts the site (lychee's --root-dir).
	siteRoot = "/site"
)

var dartSassSHA256 = map[string]string{
	"x64":   "b71b36d88354825b57eaffed397538aa84d2e8840e8d2fe13dd57cd8aa3a3914",
	"arm64": "98fbeaea1674851cba3b6f854fa3ff00a73bf8bd6056c0e8a768c47d4f578b43",
}

var pagefindSHA256 = map[string]string{
	"x86_64":  "afb824a9e7f64905a934900481cea5be679c03975e527329e0e5e6cc70f5feda",
	"aarch64": "f50ec608bcbf431cebd84e0efa3a5b041ee63df2ad81f138023e0cfd2f509424",
}

// Hugo builds and checks Hugo sites.
type Hugo struct {
	// Container has hugo on PATH and runs as a user that owns /src, /tmp and /cache.
	Container *dagger.Container
	// Lychee has lychee on PATH.
	Lychee *dagger.Container
}

// New creates a Hugo instance. The default containers are pinned by digest.
//
// +cache="session"
func New(
	// +optional
	// container is an existing container with hugo (and go, for Hugo modules) on PATH
	container *dagger.Container,
	// +optional
	// lychee is an existing container with lychee on PATH
	lychee *dagger.Container,
) *Hugo {
	if container == nil {
		container = dag.Container().From(hugoImage).
			// Hugo modules and resources (HUGO_CACHEDIR); the image's user is hugo.
			WithMountedCache("/cache", dag.CacheVolume("hugo-cache"), dagger.ContainerWithMountedCacheOpts{Owner: "hugo"})
	}
	if lychee == nil {
		lychee = dag.Container().From(lycheeImage)
	}
	return &Hugo{Container: container, Lychee: lychee}
}

// Build runs hugo on source and returns the built site (the content of public/).
//
// +cache="session"
func (h *Hugo) Build(
	ctx context.Context,
	// source is the site's project directory (hugo.yaml or config/, content/, ...), or a
	// directory that contains it (see path)
	source *dagger.Directory,
	// +default="."
	// path is the project directory within source, for a site whose module mounts reach
	// outside it (e.g. source is the repository and path is docs/site)
	path string,
	// +default="production"
	// environment is Hugo's environment (--environment)
	environment string,
	// +optional
	// baseURL overrides the site's baseURL, e.g. https://docs.example.com/ or /group/project/
	baseURL string,
	// +default=true
	// minify minifies the HTML, CSS, JS, JSON and SVG output
	minify bool,
	// +optional
	// extraArgs are additional hugo arguments, e.g. ["--panicOnWarning"] or
	// ["--config", "hugo.yaml,more.yaml"] (relative to path)
	extraArgs []string,
	// +default=false
	// npm runs npm ci in the project directory first (it needs a package-lock.json) and puts
	// its node_modules/.bin on PATH (after the pinned Dart Sass)
	npm bool,
) (*dagger.Directory, error) {
	project := pathpkg.Join(src, pathpkg.Clean("/"+path))
	ctr, err := h.withDartSass(ctx, h.Container)
	if err != nil {
		return nil, err
	}
	ctr = ctr.
		WithDirectory(src, source, dagger.ContainerWithDirectoryOpts{Owner: "hugo"}).
		WithWorkdir(project)
	if npm {
		ctr = ctr.
			// npm's download cache, next to Hugo's (the mount is the hugo user's).
			WithEnvVariable("npm_config_cache", "/cache/npm").
			// Lifecycle scripts off: the lockfile's packages are installed as published
			// (their integrity hashes are checked), nothing runs.
			WithExec([]string{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"}).
			// After Dart Sass: an npm sass-embedded picks its native compiler by its own
			// libc detection, which npm's install does not always match on Alpine.
			WithEnvVariable("PATH", dartSassDir+":"+pathpkg.Join(project, "node_modules/.bin")+":${PATH}",
				dagger.ContainerWithEnvVariableOpts{Expand: true})
	}
	args := []string{"hugo", "--source", project, "--destination", "/tmp/public",
		"--environment", environment, "--cleanDestinationDir", "--gc"}
	if baseURL != "" {
		args = append(args, "--baseURL", baseURL)
	}
	if minify {
		args = append(args, "--minify")
	}
	args = append(args, extraArgs...)
	return ctr.WithExec(args).Directory("/tmp/public"), nil
}

// withDartSass installs the pinned Dart Sass release in ctr and puts it first on PATH.
func (h *Hugo) withDartSass(ctx context.Context, ctr *dagger.Container) (*dagger.Container, error) {
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, err
	}
	arch, err := releaseArch(string(platform), "x64", "arm64")
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("dart-sass-%s-linux-%s-musl.tar.gz", dartSassVersion, arch)
	archive := dag.HTTP(fmt.Sprintf("https://github.com/sass/dart-sass/releases/download/%s/%s", dartSassVersion, name))
	install := fmt.Sprintf(`set -e
cd /tmp
echo "%s  %s" | sha256sum -c -
tar -xzf %s -C /tmp
%s/sass --version`, dartSassSHA256[arch], name, name, dartSassDir)
	return ctr.
		// The download is mode 0600; the image's user is hugo.
		WithMountedFile("/tmp/"+name, archive, dagger.ContainerWithMountedFileOpts{Owner: "hugo"}).
		WithExec([]string{"sh", "-c", install}).
		WithEnvVariable("PATH", dartSassDir+":${PATH}", dagger.ContainerWithEnvVariableOpts{Expand: true}), nil
}

// Index adds a Pagefind search index to a built site (in pagefind/, with Pagefind's UI
// scripts) and returns the site.
//
// +cache="session"
func (h *Hugo) Index(
	ctx context.Context,
	// site is a built site (Build's output)
	site *dagger.Directory,
	// +optional
	// extraArgs are additional pagefind arguments, e.g. ["--exclude-selectors", "nav"]
	extraArgs []string,
) (*dagger.Directory, error) {
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, err
	}
	arch, err := releaseArch(string(platform), "x86_64", "aarch64")
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("pagefind-%s-%s-unknown-linux-musl.tar.gz", pagefindVersion, arch)
	archive := dag.HTTP(fmt.Sprintf("https://github.com/Pagefind/pagefind/releases/download/%s/%s", pagefindVersion, name))
	install := fmt.Sprintf(`set -e
cd /tmp
echo "%s  %s" | sha256sum -c -
mkdir -p /tmp/bin
tar -xzf %s -C /tmp/bin pagefind`, pagefindSHA256[arch], name, name)
	args := append([]string{"/tmp/bin/pagefind", "--site", "/tmp/site"}, extraArgs...)
	return h.Container.
		// The download is mode 0600; the image's user is hugo.
		WithMountedFile("/tmp/"+name, archive, dagger.ContainerWithMountedFileOpts{Owner: "hugo"}).
		WithExec([]string{"sh", "-c", install}).
		WithDirectory("/tmp/site", site, dagger.ContainerWithDirectoryOpts{Owner: "hugo"}).
		WithExec(args).
		Directory("/tmp/site"), nil
}

// Site builds source and, with index, adds the Pagefind search index: Build then Index.
//
// +cache="session"
func (h *Hugo) Site(
	ctx context.Context,
	// source is the site's project directory, or a directory that contains it (see path)
	source *dagger.Directory,
	// +default="."
	// path is the project directory within source
	path string,
	// +default="production"
	// environment is Hugo's environment (--environment)
	environment string,
	// +optional
	// baseURL overrides the site's baseURL
	baseURL string,
	// +default=true
	// index adds the Pagefind search index
	index bool,
	// +optional
	// extraArgs are additional hugo arguments
	extraArgs []string,
	// +default=false
	// npm installs the site's Node dependencies first (see Build)
	npm bool,
) (*dagger.Directory, error) {
	built, err := h.Build(ctx, source, path, environment, baseURL, true, extraArgs, npm)
	if err != nil {
		return nil, err
	}
	if !index {
		return built, nil
	}
	return h.Index(ctx, built, nil)
}

// Linkcheck checks the links in a built site's HTML with lychee and returns its report.
// Internal links (files, root-relative paths, and with fragments their #anchors) are
// always checked, offline; external links only with external. It fails on a broken link.
func (h *Hugo) Linkcheck(
	ctx context.Context,
	// site is a built site (Build's output)
	site *dagger.Directory,
	// +default=false
	// external also checks http(s) links (needs network; slower and less deterministic)
	external bool,
	// +default=true
	// fragments checks that #anchors exist in the linked pages
	fragments bool,
	// +optional
	// basePath is the URL path the site is served under when not /, e.g. /group/project/
	basePath string,
	// +optional
	// exclude are regular expressions of URLs not to check, e.g. ["^https://github\\.com/org/private"]
	exclude []string,
	// +optional
	// extraArgs are additional lychee arguments
	extraArgs []string,
) (string, error) {
	root := pathpkg.Join(siteRoot, strings.Trim(pathpkg.Clean("/"+basePath), "/"))
	args := []string{"lychee", "--no-progress", "--root-dir", siteRoot, "--index-files", "index.html"}
	if !external {
		args = append(args, "--offline")
	}
	if fragments {
		args = append(args, "--include-fragments")
	}
	for _, e := range exclude {
		args = append(args, "--exclude", e)
	}
	args = append(args, extraArgs...)
	args = append(args, "--", strings.TrimSuffix(root, "/")+"/**/*.html")
	return h.Lychee.
		WithMountedDirectory(root, site).
		WithWorkdir(siteRoot).
		WithExec(args).
		Stdout(ctx)
}

// Archive packs a built site into a gzipped tarball whose files are under prefix
// (GitLab Pages publishes public/; a GitHub Pages artifact is the site's content).
//
// +cache="session"
func (h *Hugo) Archive(
	// site is a built site
	site *dagger.Directory,
	// +default="public"
	// prefix is the directory the files are under in the archive; empty puts them at the top
	prefix string,
) *dagger.File {
	dir := pathpkg.Join("/tmp/archive", strings.Trim(prefix, "/"))
	return h.Container.
		WithDirectory(dir, site, dagger.ContainerWithDirectoryOpts{Owner: "hugo"}).
		WithWorkdir("/tmp/archive").
		WithExec([]string{"tar", "-czf", "/tmp/site.tar.gz", "."}).
		File("/tmp/site.tar.gz")
}

// releaseArch maps an OCI platform (os/arch[/variant]) to a release's architecture name:
// amd64 for linux/amd64, arm64 for linux/arm64.
func releaseArch(platform, amd64, arm64 string) (string, error) {
	parts := strings.Split(platform, "/")
	if len(parts) < 2 || parts[0] != "linux" {
		return "", fmt.Errorf("unsupported platform %q", platform)
	}
	switch parts[1] {
	case "amd64":
		return amd64, nil
	case "arm64":
		return arm64, nil
	}
	return "", fmt.Errorf("no release for %q", platform)
}
