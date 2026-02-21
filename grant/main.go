// A generated module for Grant functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/grant/internal/dagger"
	"dagger/grant/internal/report"
	"fmt"
)

type Grant struct {
	Container *dagger.Container
}

// New creates a new Grant instance with base container.
func New(
	// +optional
	// container is an existing container to use instead of creating a new one
	container *dagger.Container,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +default="ghcr.io/anchore/grant"
	// repository is the Docker repository for the Grant image (default: ghcr.io/anchore/grant)
	repository string,
	// +default="latest"
	// tag is the Docker tag for the Grant image (default: latest)
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to add to the container
	extraCaCerts []*dagger.File,
) *Grant {

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

	return &Grant{
		Container: container,
	}
}

// Grant runs a grant scan and returns a markdown report from a provided SBOM.
func (g *Grant) Check(
	ctx context.Context,
	// sbom is the SBOM file to scan (Syft JSON, CycloneDX, SPDX, etc.)
	sbom *dagger.File,
	// +required
	// config is the Grant configuration file to use
	config *dagger.File,
	// +optional
	// extraArgs are additional command-line arguments passed to 'grant'
	extraArgs []string,
) (*dagger.Directory, error) {
	if sbom == nil {
		return nil, fmt.Errorf("you must provide --sbom")
	}

	ctr := g.Container.
		WithFile(".grant.yaml", config).
		WithFile("sbom.json", sbom)

	args := []string{
		"check",
		"sbom.json",
		"--output", "json",
		"--output-file", "/tmp/report.json",
	}

	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.WithExec(
		args,
		dagger.ContainerWithExecOpts{
			UseEntrypoint: true,
			Expect:        dagger.ReturnTypeAny,
		},
	)

	jsonBytes, err := ctr.File("/tmp/report.json").Contents(ctx)
	if err != nil {
		return nil, fmt.Errorf("read report.json: %w", err)
	}
	md, err := report.ToMarkdown([]byte(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("report to markdown: %w", err)
	}

	return ctr.Directory("/tmp").WithNewFile("report.md", md), nil
}
