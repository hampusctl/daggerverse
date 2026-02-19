package main

import (
	"context"
	"fmt"
	"terraform/internal/dagger"
)

type Terraform struct {
	Container *dagger.Container
}

// New creates a new Terraform instance with base container
func New(
	// +defaultPath="."
	// source is the directory containing Terraform configuration files
	source *dagger.Directory,
	// +optional
	// container is an existing container to use instead of creating a new one
	container *dagger.Container,
	// +optional
	// apkoFile is a custom Apko image file to import instead of using repository:tag
	apkoFile *dagger.File,
	// +optional
	// repository is the Docker repository for the Terraform image (default: hashicorp/terraform)
	repository string,
	// +optional
	// tag is the Docker tag for the Terraform image (default: latest)
	tag string,
	// +optional
	// extraCaCerts are additional CA certificate files to add to the container
	extraCaCerts []*dagger.File,
) *Terraform {

	if container == nil {
		if apkoFile == nil {
			// Build image reference
			imageRef := "hashicorp/terraform:latest"
			if repository != "" && tag != "" {
				imageRef = fmt.Sprintf("%s:%s", repository, tag)
			}
			container = dag.Container().From(imageRef)
		} else {
			container = dag.Apko().Build(apkoFile)
		}

		// Add extra CA certificates if provided.
		for i, cert := range extraCaCerts {
			certPath := fmt.Sprintf("/usr/local/share/ca-certificates/extra%d.crt", i)
			container = container.WithFile(certPath, cert)
		}
	}

	// Configure workspace with mounted directory and nonroot user
	container = container.
		WithMountedDirectory("/workspace", source). //dagger.ContainerWithMountedDirectoryOpts{
		//Owner: "nonroot:nonroot",
		//}

		WithWorkdir("/workspace") //.WithUser("nonroot:nonroot")

	return &Terraform{
		Container: container,
	}
}

// WithCloudsYaml adds OpenStack clouds.yaml configuration to the container
func (t *Terraform) WithCloudsYaml(
	ctx context.Context,
	// +optional
	// cloudsYaml is the OpenStack clouds.yaml configuration file
	cloudsYaml *dagger.File,
) *Terraform {
	t.Container = t.Container.
		WithExec([]string{"mkdir", "-p", "/etc/openstack"}).
		WithFile("/etc/openstack/clouds.yaml", cloudsYaml).
		WithEnvVariable("OS_CLIENT_CONFIG_FILE", "/etc/openstack/clouds.yaml")
	return t
}

// Init initializes a Terraform working directory
func (t *Terraform) Init(
	ctx context.Context,
	// +optional
	// extraArgs are additional arguments to pass to terraform init command
	extraArgs []string,
) (*dagger.Container, error) {
	args := []string{"terraform", "init"}

	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	return t.Container.WithExec(args), nil
}

// Plan creates an execution plan and saves it to output.tfplan with markdown report (automatically runs Init first)
func (t *Terraform) Plan(
	ctx context.Context,
	// +optional
	// extraArgs are additional arguments to pass to terraform plan command
	extraArgs []string,
) (*dagger.Directory, error) {
	ctr, err := t.Init(ctx, nil)
	if err != nil {
		return nil, err
	}
	args := []string{"terraform", "plan", "-out=output.tfplan"}

	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.
		WithExec(args).
		// Create markdown report from the plan
		WithExec([]string{"sh", "-c", "echo '# Terraform Plan Report' > /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '## Plan Summary' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '```hcl' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "terraform show -no-color /workspace/output.tfplan >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '```' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '---' >> /workspace/output.md"}).
		WithExec([]string{"sh", "-c", "echo '*Report generated on '$(date)'*' >> /workspace/output.md"})

	return ctr.Directory("/workspace"), nil
}

// Apply executes a specific Terraform plan file (automatically runs Init first)
func (t *Terraform) Apply(
	ctx context.Context,
	// planFile is the Terraform plan file to apply (usually output.tfplan)
	planFile *dagger.File,
	// +optional
	// extraArgs are additional arguments to pass to terraform apply command
	extraArgs []string,
) (*dagger.Container, error) {
	ctr, err := t.Init(ctx, nil)
	if err != nil {
		return nil, err
	}

	args := []string{"terraform", "apply", "plan.tfplan", "--auto-approve"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}

	ctr = ctr.
		WithFile("/workspace/plan.tfplan", planFile).
		WithExec(args)

	return ctr, nil
}

// Validate checks whether the configuration is valid (automatically runs Init first)
func (t *Terraform) Validate(
	ctx context.Context,
) (string, error) {
	ctr, err := t.Init(ctx, nil)
	if err != nil {
		return "", err
	}

	ctr = ctr.WithExec([]string{"terraform", "validate"})
	return ctr.Stdout(ctx)
}
