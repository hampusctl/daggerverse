// Melange builds APK packages from a YAML config (Chainguard Melange).
// Use WithGeneratedSignKey or WithProvidedSignKey before Build().
package main

import (
	"dagger/melange/internal/dagger"
	"fmt"
)

type Melange struct {
	Container *dagger.Container
}

// New creates a new Melange instance. Uses repository:tag by default, or an apko-built image if apkoFile is set.
func New(
	// +optional
	// container to use for Apko operations. If not provided, creates a new one from repository:tag
	container *dagger.Container,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +default="cgr.dev/chainguard/melange"
	// repository is the container registry and image name for the Melange tool
	repository string,
	// +default="latest"
	// tag specifies the version of the Melange tool to use
	tag string,
) *Melange {
	if container == nil {
		if apkoFile != nil {
			container = dag.Apko().Build(apkoFile)
		} else {
			container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
		}
	}
	container = container.WithWorkdir("/workspace")

	return &Melange{
		Container: container,
	}
}

// WithGeneratedSignKey generates a new signing key for the Melange container.
func (m *Melange) WithGeneratedSignKey(
	// +default="4096"
	// bits is the number of bits to use for the key
	bits int,
) *Melange {
	m.Container = m.Container.
		WithEnvVariable("MELANGE_SIGN_KEY_PATH", "/workspace/melange.rsa").
		WithEnvVariable("MELANGE_SIGN_KEY_PUB_PATH", "/workspace/melange.rsa.pub").
		WithExec([]string{
			"keygen",
			"--key-size", fmt.Sprintf("%d", bits),
		}, dagger.ContainerWithExecOpts{
			UseEntrypoint: true,
		})
	return m
}

// WithProvidedSignKey sets the signing key for the Melange container.
func (m *Melange) WithProvidedSignKey(
	// +required
	// privateKey is the private key to use for signing
	privateKey *dagger.Secret,
	// +required
	// publicKey is the public key file to use for signing
	publicKey *dagger.File,
) *Melange {
	m.Container = m.Container.
		WithMountedSecret("/workspace/melange.rsa", privateKey).
		WithFile("/workspace/melange.rsa.pub", publicKey).
		WithEnvVariable("MELANGE_SIGN_KEY_PATH", "/workspace/melange.rsa").
		WithEnvVariable("MELANGE_SIGN_KEY_PUB_PATH", "/workspace/melange.rsa.pub")
	return m
}

// Build builds APK packages from a Melange config. Call WithGeneratedSignKey or WithProvidedSignKey first.
func (m *Melange) Build(
	// +required
	// config is the Melange configuration file to use
	config *dagger.File,
	// +optional
	// extraArgs are additional command-line arguments passed to 'melange'
	extraArgs []string,
) (*dagger.Directory, error) {

	ctr := m.Container.
		WithFile("melange.yaml", config).
		WithDirectory("/workspace/packages", dag.Directory())

	args := []string{
		"build",
		"melange.yaml",
		"--signing-key", "/workspace/melange.rsa",
		"--out-dir", "/workspace/packages",
	}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.WithExec(
		args,
		dagger.ContainerWithExecOpts{
			UseEntrypoint:            true,
			InsecureRootCapabilities: true, // required for melange's bwrap/sandbox inside the container
		},
	)

	ctr = ctr.
		WithFile("/workspace/packages/melange.rsa.pub",
			ctr.File("/workspace/melange.rsa.pub"))

	return ctr.Directory("/workspace/packages"), nil
}
