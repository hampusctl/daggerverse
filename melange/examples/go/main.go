// A generated module for Go functions
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
	"dagger/go/internal/dagger"
	"fmt"
)

type Go struct {
	Container *dagger.Container
}

// New creates a new Go instance. Uses repository:tag by default, or an apko-built image if apkoFile is set for building the APK packages.
func New(
	// +optional
	// container to use for Apko operations. If not provided, creates a new one from repository:tag
	container *dagger.Container,
	// +optional
	// melangeConfig is the Melange configuration file to use for building the APK packages
	melangeConfig *dagger.File,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +default="docker.io/library/golang"
	// repository is the container registry and image name for the Go tool
	repository string,
	// +default="latest"
	// tag specifies the version of the Go tool to use
	tag string,
) *Go {
	if container == nil {
		if apkoFile != nil {
			if melangeConfig != nil {
				packages := dag.Melange().WithGeneratedSignKey(dagger.MelangeWithGeneratedSignKeyOpts{
					Bits: 4096,
				}).Build(melangeConfig)
				container = dag.
					Apko().
					WithPackages(packages).
					Build(apkoFile)

			} else {
				container = dag.Apko().Build(apkoFile)
			}
		} else {
			container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
		}
	}
	container = container.WithWorkdir("/workspace")

	return &Go{
		Container: container,
	}
}
