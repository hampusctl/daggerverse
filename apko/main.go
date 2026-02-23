package main

import (
	"context"
	"dagger/apko/internal/dagger"
	"fmt"
)

const (
	apkoTemplateName  = "apko.yaml"
	apkoImageFileName = "image.tar"
)

// Apko provides functionality for building minimal container images using Apko
type Apko struct {
	Container *dagger.Container
}

// New creates a new Apko instance with a configured container environment.
// If no container is provided, it creates one from the specified repository and tag.
func New(
	// +optional
	// container to use for Apko operations. If not provided, creates a new one from repository:tag
	container *dagger.Container,
	// +default="cgr.dev/chainguard/apko"
	// repository is the container registry and image name for the Apko tool
	repository string,
	// +default="latest"
	// tag specifies the version of the Apko tool to use
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to trust in the container environment
	extraCaCerts []*dagger.File,
) *Apko {

	if container == nil {
		// Create base container from the specified Apko image
		container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))

		// Install additional CA certificates for secure connections
		for i, cert := range extraCaCerts {
			certPath := fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i)
			container = container.WithFile(certPath, cert)
		}
	}
	return &Apko{
		Container: container,
	}
}

// WithPackages mounts a directory of packages to the container before building.
func (a *Apko) WithPackages(
	// +required
	// packages is the directory of packages to mount
	packages *dagger.Directory,
) *Apko {
	a.Container = a.Container.WithMountedDirectory("/workspace/packages", packages)
	return a
}

// Build creates a container image from an Apko configuration file.
// The configuration is processed and built into a container image.
// Returns the built container ready for use or publishing.
func (a *Apko) Build(
	ctx context.Context,
	// +required
	// configFile is the Apko YAML configuration defining the image contents and metadata
	configFile *dagger.File,
	// +default="latest"
	// tag is the image tag to use when building (default: latest)
	tag string,
	// +optional
	// extraArgs are additional command-line arguments passed to 'apko build'
	extraArgs []string,
) (*dagger.Container, error) {
	// Prepare build command: apko build <config.yaml> <tag> <output.tar>
	args := []string{"apko", "build", apkoTemplateName, tag, apkoImageFileName}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	// Execute the build process
	ctr := a.Container.
		WithFile(apkoTemplateName, configFile).
		WithExec(args)

	// Import the generated tar archive as a Dagger container
	imageFile := ctr.File(apkoImageFileName)
	return dag.Container().Import(imageFile), nil
}

// ShowConfig displays the resolved configuration from an Apko YAML file.
// This is useful for debugging and understanding what the final configuration looks like.
func (a *Apko) ShowConfig(
	ctx context.Context,
	// +required
	// configFile is the Apko YAML configuration file to analyze
	configFile *dagger.File,
) (string, error) {
	ctr := a.Container.
		WithFile(apkoTemplateName, configFile).
		WithExec([]string{"apko", "show-config", apkoTemplateName})
	return ctr.Stdout(ctx)
}

// ShowPackages displays the packages and versions that would be installed by the configuration.
// This is useful for understanding what will be included in the final image.
func (a *Apko) ShowPackages(
	ctx context.Context,
	// +required
	// configFile is the Apko YAML configuration file to analyze
	configFile *dagger.File,
) (string, error) {
	ctr := a.Container.
		WithFile(apkoTemplateName, configFile).
		WithExec([]string{"apko", "show-packages", apkoTemplateName})
	return ctr.Stdout(ctx)
}
