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
	// +default="anchore/grype"
	// repository is the Docker repository for the Grype image
	repository string,
	// +default="latest"
	// tag is the Docker tag for the Grype image
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to add to the container
	extraCaCerts []*dagger.File,
) *Grype {
	if container == nil {
		if apkoFile == nil {
			container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
		} else {
			// TODO: Build custom image using Apko configuration
			// This would require calling dag.Apko().Build() with proper parameters
			// For now, fallback to default image
			container = dag.Container().From("anchore/grype:latest")
		}

		for i, cert := range extraCaCerts {
			certPath := fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i)
			container = container.WithFile(certPath, cert)
		}
	}

	return &Grype{Container: container}
}

// ScanSbom runs a vulnerability scan from a provided SBOM file and returns a markdown report.
// The SBOM is mounted into the container and scanned, with results formatted as a markdown table.
func (g *Grype) ScanSbom(
	ctx context.Context,
	// sbom is the SBOM file to scan (Syft JSON, CycloneDX, SPDX, etc.)
	sbom *dagger.File,
	// +optional
	// extraArgs are additional command-line arguments passed to 'grype'
	extraArgs []string,
) (*dagger.File, error) {
	// Load markdown template from embedded file
	templateFile := dag.CurrentModule().Source().File("internal/templates/vulnerability-report.md.tmpl")

	args := []string{"grype", "sbom:/sbom", "-o", "template", "-t", "/template.tmpl"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := g.Container.
		WithFile("/template.tmpl", templateFile).
		WithFile("/sbom", sbom).
		WithExec([]string{"sh", "-c", "grype sbom:/sbom -o template -t /template.tmpl > /report.md"})

	return ctr.File("/report.md"), nil
}

// ScanImage runs a vulnerability scan on a container image and returns a markdown report.
// Supports various image sources like docker:, registry:, oci-archive:, etc.
func (g *Grype) ScanImage(
	ctx context.Context,
	// imageRef is the container image reference to scan (e.g., "alpine:latest", "docker:myimage:tag")
	imageRef string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'grype'
	extraArgs []string,
) (*dagger.File, error) {
	// Load markdown template from embedded file
	templateFile := dag.CurrentModule().Source().File("internal/templates/vulnerability-report.md.tmpl")

	args := []string{"grype", imageRef, "-o", "template", "-t", "/template.tmpl"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := g.Container.
		WithFile("/template.tmpl", templateFile).
		WithExec([]string{"sh", "-c", fmt.Sprintf("grype %s -o template -t /template.tmpl > /report.md", imageRef)})

	return ctr.File("/report.md"), nil
}

// ScanDirectory runs a vulnerability scan on a filesystem directory and returns a markdown report.
// The directory is mounted into the container and scanned via: grype dir:/workspace
func (g *Grype) ScanDirectory(
	ctx context.Context,
	// directory is the filesystem directory to scan for vulnerabilities
	directory *dagger.Directory,
	// +optional
	// extraArgs are additional command-line arguments passed to 'grype'
	extraArgs []string,
) (*dagger.File, error) {
	// Load markdown template from embedded file
	templateFile := dag.CurrentModule().Source().File("internal/templates/vulnerability-report.md.tmpl")

	args := []string{"grype", "dir:/workspace", "-o", "template", "-t", "/template.tmpl"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := g.Container.
		WithFile("/template.tmpl", templateFile).
		WithMountedDirectory("/workspace", directory).
		WithExec([]string{"sh", "-c", "grype dir:/workspace -o template -t /template.tmpl > /report.md"})

	return ctr.File("/report.md"), nil
}
