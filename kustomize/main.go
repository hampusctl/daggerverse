// Build and edit kustomizations with kustomize.
//
// The default container is Alpine with the official kustomize release binary for
// the engine's platform (the registry.k8s.io image is amd64-only).
package main

import (
	"context"
	"dagger/kustomize/internal/dagger"
	"fmt"
	"path"
	"strings"
)

const workdir = "/work"

// Kustomize runs kustomize build and kustomize edit.
type Kustomize struct {
	Container *dagger.Container
}

// New creates a Kustomize instance. If container is not provided, kustomize
// version is downloaded into an Alpine container.
func New(
	ctx context.Context,
	// +optional
	// container is an existing container with kustomize on PATH
	container *dagger.Container,
	// +default="v5.8.3"
	// version is the kustomize release, e.g. v5.8.3
	version string,
	// +default="alpine:3.22"
	// base is the image the binary is installed into
	base string,
	// +optional
	// extraCaCerts are additional CA certificate files to trust
	extraCaCerts []*dagger.File,
) (*Kustomize, error) {
	if container == nil {
		platform, err := dag.DefaultPlatform(ctx)
		if err != nil {
			return nil, err
		}
		// Platform is os/arch[/variant], e.g. linux/arm64/v8.
		parts := strings.Split(string(platform), "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected platform %q", platform)
		}
		url := fmt.Sprintf(
			"https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%%2F%s/kustomize_%s_%s_%s.tar.gz",
			version, version, parts[0], parts[1])
		archive := dag.HTTP(url)
		container = dag.Container().From(base).
			WithMountedFile("/tmp/kustomize.tar.gz", archive).
			WithExec([]string{"tar", "-xzf", "/tmp/kustomize.tar.gz", "-C", "/usr/local/bin", "kustomize"})
		for i, cert := range extraCaCerts {
			container = container.WithFile(fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i), cert).
				WithExec([]string{"update-ca-certificates"})
		}
	}
	return &Kustomize{Container: container}, nil
}

// Build runs kustomize build on path inside source and returns the rendered YAML.
func (k *Kustomize) Build(
	// source is the directory holding the kustomization and everything it references
	source *dagger.Directory,
	// +default="."
	// path is the kustomization directory, relative to source
	path string,
	// +optional
	// extraArgs are additional kustomize build arguments, e.g. ["--enable-helm"]
	extraArgs []string,
) *dagger.File {
	args := append([]string{"sh", "-c", `kustomize build "$@" > /tmp/out.yaml`, "kustomize", path}, extraArgs...)
	return k.Container.
		WithMountedDirectory(workdir, source).
		WithWorkdir(workdir).
		WithExec(args).
		File("/tmp/out.yaml")
}

// SetImage runs "kustomize edit set image" on the kustomization in dir and returns
// the updated source directory. Each image is "name=newName[:tag|@digest]".
func (k *Kustomize) SetImage(
	// source is the directory holding the kustomization
	source *dagger.Directory,
	// +default="."
	// dir is the kustomization directory, relative to source
	dir string,
	// images are the image overrides, e.g. ["controller=ghcr.io/org/app@sha256:..."]
	images []string,
) *dagger.Directory {
	return k.Container.
		WithDirectory(workdir, source).
		WithWorkdir(path.Join(workdir, cleanRel(dir))).
		WithExec(append([]string{"kustomize", "edit", "set", "image"}, images...)).
		Directory(workdir)
}

func cleanRel(p string) string {
	c := path.Clean("/" + p)
	return strings.TrimPrefix(c, "/")
}
