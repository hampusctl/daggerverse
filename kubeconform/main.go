// Validate Kubernetes manifests against JSON schemas with kubeconform.
//
// Validates a directory or a single (multi-document) YAML file, optionally
// against CRD schemas (for example the datreeio CRDs-catalog).
package main

import (
	"context"
	"dagger/kubeconform/internal/dagger"
	"fmt"
)

const workdir = "/work"

// Kubeconform validates Kubernetes manifests with kubeconform.
type Kubeconform struct {
	Container *dagger.Container
}

// New creates a Kubeconform instance. If container is not provided, it is
// created from repository:tag.
func New(
	// +optional
	// container is an existing container with kubeconform on PATH
	container *dagger.Container,
	// +default="ghcr.io/yannh/kubeconform"
	// repository is the kubeconform image repository
	repository string,
	// +default="v0.8.0-alpine"
	// tag is the kubeconform image tag
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to trust
	extraCaCerts []*dagger.File,
) *Kubeconform {
	if container == nil {
		container = dag.Container().From(fmt.Sprintf("%s:%s", repository, tag))
		for i, cert := range extraCaCerts {
			container = container.WithFile(fmt.Sprintf("/etc/ssl/certs/extra%d.pem", i), cert)
		}
	}
	return &Kubeconform{Container: container}
}

// Validate validates the manifests in a directory (recursively) or in one file and
// returns kubeconform's output. It fails when any resource is invalid.
func (k *Kubeconform) Validate(
	ctx context.Context,
	// +optional
	// manifests is a directory of YAML/JSON manifests (searched recursively)
	manifests *dagger.Directory,
	// +optional
	// file is a single, possibly multi-document, manifest file
	file *dagger.File,
	// +default=true
	// strict disallows additional properties not in the schema and duplicated keys
	strict bool,
	// +default=false
	// ignoreMissingSchemas skips resources whose schema is not found
	ignoreMissingSchemas bool,
	// +optional
	// skip lists kinds to skip, e.g. ["Secret"] or ["apps/v1/Deployment"]
	skip []string,
	// +optional
	// schemaLocations are kubeconform -schema-location values; empty means "default"
	schemaLocations []string,
	// +optional
	// kubernetesVersion validates against this Kubernetes version (e.g. 1.33.0); default is master
	kubernetesVersion string,
	// +default="text"
	// format is the output format: json, junit, pretty, tap or text
	format string,
) (string, error) {
	if (manifests == nil) == (file == nil) {
		return "", fmt.Errorf("set exactly one of manifests or file")
	}

	args := []string{"/kubeconform", "-summary", "-output", format}
	if strict {
		args = append(args, "-strict")
	}
	if ignoreMissingSchemas {
		args = append(args, "-ignore-missing-schemas")
	}
	for _, kind := range skip {
		args = append(args, "-skip", kind)
	}
	if len(schemaLocations) == 0 {
		schemaLocations = []string{"default"}
	}
	for _, loc := range schemaLocations {
		args = append(args, "-schema-location", loc)
	}
	if kubernetesVersion != "" {
		args = append(args, "-kubernetes-version", kubernetesVersion)
	}

	ctr := k.Container.WithWorkdir(workdir)
	if manifests != nil {
		ctr = ctr.WithMountedDirectory(workdir, manifests)
		args = append(args, ".")
	} else {
		ctr = ctr.WithMountedFile(workdir+"/manifests.yaml", file)
		args = append(args, "manifests.yaml")
	}

	return ctr.WithExec(args).Stdout(ctx)
}
