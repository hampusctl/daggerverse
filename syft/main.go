package main

import (
	"context"
	"dagger/syft/internal/dagger"
	"fmt"
	"strings"
)

// Syft provides functionality for generating Software Bill of Materials (SBOM) using Anchore Syft.
type Syft struct {
	Container *dagger.Container
}

// New creates a new Syft instance with base container.
func New(
	// +optional
	// container is an existing container to use instead of creating a new one
	container *dagger.Container,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +default="ghcr.io/anchore/syft"
	// repository is the Docker repository for the Syft image (default: ghcr.io/anchore/syft)
	repository string,
	// +default="latest"
	// tag is the Docker tag for the Syft image (default: latest)
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to add to the container
	extraCaCerts []*dagger.File,
) *Syft {

	if container == nil {
		if apkoFile == nil {
			if repository != "" && tag != "" {
				container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
			}
		} else {
			container = dag.Apko().Build(apkoFile)
		}

		// Add extra CA certificates if provided.
		for i, cert := range extraCaCerts {
			certPath := fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i)
			container = container.WithFile(certPath, cert)
		}
	}
	container = container.WithWorkdir("/workspace")

	return &Syft{
		Container: container,
	}
}

// ScanImage generates an SBOM from a container image and returns it as a file.
// Supports various image sources like docker:, registry:, oci-archive:, etc.
func (s *Syft) Scan(
	ctx context.Context,
	// +optional
	// image is a container image to scan (e.g., "alpine:latest" or "docker:myimage:tag")
	image *dagger.Container,
	// +optional
	// directory is a directory to scan for SBOM (can be "." for current directory)
	directory *dagger.Directory,
	// +optional
	// file is a single file to scan for SBOM
	file *dagger.File,
	// +default="spdx-json"
	// outputFormat specifies the SBOM output format.
	// Supported formats: cyclonedx-json, cyclonedx-xml, github-json, purls, spdx-json, spdx-tag-value, syft-json, syft-table, syft-text, template
	outputFormat string,
	// +defaultPath="/.templates/syft.tmpl"
	// template is the Go template file to use when outputFormat=template
	template *dagger.File,
	// +optional
	// scheme prefixes the source path (e.g., "docker:SOURCE")
	scheme string,
	// +optional
	// extraArgs are additional command-line arguments passed directly to `syft scan`
	extraArgs []string,
) (*dagger.Directory, error) {
	if image == nil && directory == nil && file == nil {
		return nil, fmt.Errorf("You must provide either --image, --directory or --file")
	}

	ctr := s.Container

	if image != nil {
		tarball := image.AsTarball()
		ctr = ctr.
			WithFile("SOURCE", tarball)
	} else if directory != nil {
		ctr = ctr.
			WithDirectory("SOURCE", directory)
	} else if file != nil {
		ctr = ctr.
			WithFile("SOURCE", file)
	} else {
		return nil, fmt.Errorf("You can only provide either a image, directory or file")
	}

	if scheme != "" {
		scheme = fmt.Sprintf("%s:", scheme)
	}

	args := []string{
		"scan",
		fmt.Sprintf("%sSOURCE", scheme),
		"-o", fmt.Sprintf("%s=/tmp/sbom.json", outputFormat),
		"-o", "template=/tmp/report.md",
		"-t", "syft.tpml",
	}

	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.
		WithFile("syft.tpml", template).
		WithExec(args, dagger.ContainerWithExecOpts{UseEntrypoint: true})
	return detach(ctx, ctr.Directory("/tmp"))
}

// detach copies the files at the top of dir into a new directory built only from their
// contents. A directory returned straight from a container keeps, in its ID, the inputs that
// made it, including caller files loaded from the caller's module context by host path. Dagger
// caches function results across sessions, so another runner (another checkout path) that
// gets this result from the cache fails to load it with "module not found: <first caller's
// path>". A directory of new files refers to nothing of the caller's.
func detach(ctx context.Context, dir *dagger.Directory) (*dagger.Directory, error) {
	names, err := dir.Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list outputs: %w", err)
	}
	out := dag.Directory()
	for _, name := range names {
		if strings.HasSuffix(name, "/") {
			continue
		}
		contents, err := dir.File(name).Contents(ctx)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		out = out.WithNewFile(name, contents)
	}
	return out, nil
}
