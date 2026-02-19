package main

import (
	"context"
	"dagger/syft/internal/dagger"
	"fmt"
)

// Syft provides functionality for generating Software Bill of Materials (SBOM) using Anchore Syft.
type Syft struct {
	Container *dagger.Container
}

// New creates a new Syft instance with a configured container environment.
// If no container is provided, it creates one from the specified repository and tag.
func New(
	// +optional
	// container to use for Syft operations. If not provided, creates a new one from repository:tag
	container *dagger.Container,
	// +default="anchore/syft"
	// repository is the container registry and image name for the Syft tool
	repository string,
	// +default="latest"
	// tag specifies the version of the Syft tool to use
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to trust in the container environment
	extraCaCerts []*dagger.File,
) *Syft {
	if container == nil {
		// Create base container from the specified Syft image
		container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))

		// Install additional CA certificates for secure connections
		for i, cert := range extraCaCerts {
			certPath := fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i)
			container = container.WithFile(certPath, cert)
		}
	}

	return &Syft{Container: container}
}

// ScanImage generates an SBOM from a container image and returns it as a file.
// Supports various image sources like docker:, registry:, oci-archive:, etc.
func (s *Syft) ScanImage(
	ctx context.Context,
	// imageRef is the container image reference to scan (e.g., "alpine:latest", "docker:myimage:tag")
	imageRef string,
	// +default="syft-json"
	// outputFormat is the SBOM output format (syft-json, spdx-json, cyclonedx-json, etc.)
	outputFormat string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'syft scan'
	extraArgs []string,
) (*dagger.File, error) {
	args := []string{"syft", "scan", imageRef, "-o", outputFormat, "--file", "/sbom.json"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := s.Container.WithExec(args)
	return ctr.File("/sbom.json"), nil
}

// ScanDirectory generates an SBOM from a filesystem directory and returns it as a file.
// The directory is mounted into the container and scanned via: syft scan dir:/workspace
func (s *Syft) ScanDirectory(
	ctx context.Context,
	// directory is the filesystem directory to scan for packages
	directory *dagger.Directory,
	// +default="syft-json"
	// outputFormat is the SBOM output format (syft-json, spdx-json, cyclonedx-json, etc.)
	outputFormat string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'syft scan'
	extraArgs []string,
) (*dagger.File, error) {
	args := []string{"syft", "scan", "dir:/workspace", "-o", outputFormat, "--file", "/sbom.json"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := s.Container.
		WithMountedDirectory("/workspace", directory).
		WithExec(args)

	return ctr.File("/sbom.json"), nil
}

// ScanFile generates an SBOM from a single file (archive, binary, etc.) and returns it as a file.
// Useful for scanning tarballs, zip files, or individual binaries.
func (s *Syft) ScanFile(
	ctx context.Context,
	// file is the file to scan (tarball, binary, etc.)
	file *dagger.File,
	// +default="syft-json"
	// outputFormat is the SBOM output format (syft-json, spdx-json, cyclonedx-json, etc.)
	outputFormat string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'syft scan'
	extraArgs []string,
) (*dagger.File, error) {
	args := []string{"syft", "scan", "file:/input", "-o", outputFormat, "--file", "/sbom.json"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := s.Container.
		WithFile("/input", file).
		WithExec(args)

	return ctr.File("/sbom.json"), nil
}

// Convert converts an SBOM from one format to another.
// Useful for converting between SPDX, CycloneDX, and Syft JSON formats.
func (s *Syft) Convert(
	ctx context.Context,
	// sbom is the SBOM file to convert
	sbom *dagger.File,
	// +default="spdx-json"
	// outputFormat is the target SBOM format (spdx-json, cyclonedx-json, syft-json, etc.)
	outputFormat string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'syft convert'
	extraArgs []string,
) (*dagger.File, error) {
	args := []string{"syft", "convert", "/input.json", "-o", outputFormat, "--file", "/output.json"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr := s.Container.
		WithFile("/input.json", sbom).
		WithExec(args)

	return ctr.File("/output.json"), nil
}
