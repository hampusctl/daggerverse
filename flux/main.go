// Flux CLI operations that need no cluster: OCI artifacts and envsubst.
//
// Push a directory as a Flux OCI artifact, tag it, and substitute ${VAR}
// references the way Flux post-build substitution does.
package main

import (
	"context"
	"dagger/flux/internal/dagger"
	"encoding/json"
	"fmt"
	"strings"
)

// Under /tmp: flux writes its temporary layer next to the source directory, and the
// image runs as nobody.
const workdir = "/tmp/src"

// Flux runs Flux CLI commands that need no cluster.
type Flux struct {
	Container *dagger.Container
}

// New creates a Flux instance. If container is not provided, it is created from
// repository:tag.
func New(
	// +optional
	// container is an existing container with flux on PATH
	container *dagger.Container,
	// +default="ghcr.io/fluxcd/flux-cli"
	// repository is the Flux CLI image repository
	repository string,
	// +default="v2.9.6"
	// tag is the Flux CLI image tag
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to trust
	extraCaCerts []*dagger.File,
) *Flux {
	if container == nil {
		container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
		for i, cert := range extraCaCerts {
			container = container.WithFile(fmt.Sprintf("/etc/ssl/certs/extra%d.pem", i), cert)
		}
	}
	return &Flux{Container: container}
}

// WithRegistryAuth sets credentials for an OCI registry, used by PushArtifact and TagArtifact.
func (f *Flux) WithRegistryAuth(
	// address is the registry host, e.g. ghcr.io
	address string,
	// username is the registry user
	username string,
	// secret is the registry password or token
	secret *dagger.Secret,
) *Flux {
	f.Container = f.Container.WithRegistryAuth(address, username, secret).
		WithEnvVariable("FLUX_OCI_USER", username).
		WithSecretVariable("FLUX_OCI_TOKEN", secret)
	return f
}

// PushArtifact pushes source as an OCI artifact and returns its digest URL
// (oci://repo@sha256:...).
func (f *Flux) PushArtifact(
	ctx context.Context,
	// source is the directory to package
	source *dagger.Directory,
	// url is the artifact URL with tag, e.g. oci://ghcr.io/org/app:v1
	url string,
	// sourceURL is recorded as the artifact's source, e.g. the git repository URL
	sourceURL string,
	// revision is recorded as the artifact's revision, e.g. main@sha1:<sha>
	revision string,
	// +optional
	// ignorePaths are .gitignore-style patterns to exclude
	ignorePaths []string,
	// +optional
	// annotations are extra OCI annotations as key=value
	annotations []string,
	// +optional
	// insecure allows plain HTTP registries
	insecure bool,
) (string, error) {
	args := []string{
		"push", "artifact", url,
		"--path=.",
		"--source=" + sourceURL,
		"--revision=" + revision,
		"--output=json",
	}
	if len(ignorePaths) > 0 {
		args = append(args, "--ignore-paths="+strings.Join(ignorePaths, ","))
	}
	for _, a := range annotations {
		args = append(args, "--annotations="+a)
	}
	if insecure {
		args = append(args, "--insecure-registry")
	}
	out, err := f.run(source, args).Stdout(ctx)
	if err != nil {
		return "", err
	}
	return digestURL(out)
}

// TagArtifact adds tags to an existing artifact.
func (f *Flux) TagArtifact(
	ctx context.Context,
	// url is the artifact URL with tag or digest, e.g. oci://ghcr.io/org/app:sha-abc
	url string,
	// tags are the tags to add
	tags []string,
) (string, error) {
	args := []string{"tag", "artifact", url}
	for _, t := range tags {
		args = append(args, "--tag="+t)
	}
	return f.run(nil, args).Stdout(ctx)
}

// Envsubst substitutes ${VAR} references in input with vars, like Flux post-build
// substitution, and returns the result.
func (f *Flux) Envsubst(
	// input is the file to substitute
	input *dagger.File,
	// +optional
	// vars are KEY=VALUE pairs
	vars []string,
	// +default=true
	// strict fails on variables that are referenced but not set
	strict bool,
) (*dagger.File, error) {
	ctr := f.Container.WithMountedFile("/tmp/in.yaml", input)
	for _, kv := range vars {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("vars must be KEY=VALUE, got %q", kv)
		}
		ctr = ctr.WithEnvVariable(k, v)
	}
	cmd := "flux envsubst < /tmp/in.yaml > /tmp/out.yaml"
	if strict {
		cmd = "flux envsubst --strict < /tmp/in.yaml > /tmp/out.yaml"
	}
	return ctr.WithExec([]string{"sh", "-c", cmd}).File("/tmp/out.yaml"), nil
}

// run executes flux with optional registry credentials and an optional source mount.
func (f *Flux) run(source *dagger.Directory, args []string) *dagger.Container {
	ctr := f.Container
	if source != nil {
		ctr = ctr.WithMountedDirectory(workdir, source).WithWorkdir(workdir)
	}
	// Credentials, when set, are passed via --creds so no docker config is needed.
	script := `if [ -n "${FLUX_OCI_USER:-}" ]; then exec flux "$@" --creds="$FLUX_OCI_USER:$FLUX_OCI_TOKEN"; else exec flux "$@"; fi`
	return ctr.WithExec(append([]string{"sh", "-c", script, "flux"}, args...))
}

// digestURL turns "flux push artifact --output=json" output into oci://repo@digest.
func digestURL(out string) (string, error) {
	var res struct {
		Repository string `json:"repository"`
		Digest     string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return "", fmt.Errorf("parse flux push output %q: %w", out, err)
	}
	if res.Repository == "" || res.Digest == "" {
		return "", fmt.Errorf("flux push output has no repository or digest: %q", out)
	}
	return fmt.Sprintf("oci://%s@%s", res.Repository, res.Digest), nil
}
