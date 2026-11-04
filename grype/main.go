package main

import (
	"context"
	"dagger/grype/internal/dagger"
	"fmt"
)

// Grype provides functionality for scanning SBOMs using Anchore Grype.
type Grype struct {
	Container *dagger.Container
}

// New creates a new Grype instance with a configured container environment.
// If container is provided, it is used as-is. Otherwise, a container is created
// from repository:tag, or built from an Apko config if apkoFile is provided.
func New(
	// +optional
	// container is an existing container to use instead of creating a new one
	container *dagger.Container,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +default="ghcr.io/anchore/grype"
	// repository is the Docker repository for the Grype image (default: ghcr.io/anchore/grype)
	repository string,
	// +default="latest"
	// tag is the Docker tag for the Grype image (default: latest)
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to add to the container
	extraCaCerts []*dagger.File,
) *Grype {

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

	return &Grype{
		Container: container,
	}
}

// ScanSbom runs a vulnerability scan from a provided SBOM file and returns a markdown report.
// The SBOM is mounted into the container and scanned, with results formatted as a markdown table.
func (g *Grype) Scan(
	ctx context.Context,
	// sbom is the SBOM file to scan (Syft JSON, CycloneDX, SPDX, etc.)
	sbom *dagger.File,
	// +required
	// config is the Grype configuration file to use
	config *dagger.File,
	// +defaultPath="/.templates/grype.tmpl"
	// template is the Go template file to use when outputFormat=template
	template *dagger.File,
	// +default="medium"
	// failOnSeverity is the severity level to fail on
	failOnSeverity string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'grype'
	extraArgs []string,
) (*dagger.Directory, error) {
	if sbom == nil {
		return nil, fmt.Errorf("you must provide --sbom")
	}
	if template == nil {
		return nil, fmt.Errorf("you must provide --template")
	}

	ctr := g.Container.
		WithFile(".grype.yaml", config).
		WithFile("sbom.json", sbom).
		WithFile("template.tmpl", template)

	args := []string{
		"sbom.json",
		"-o", "template",
		"-t", "template.tmpl",
		"--sort-by", "severity",
		"--file", "/tmp/report.md",
	}

	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.WithExec(
		args,
		dagger.ContainerWithExecOpts{
			UseEntrypoint: true,
		},
	)
	return ctr.
		Directory("/tmp"), nil
}
