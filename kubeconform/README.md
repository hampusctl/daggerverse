# kubeconform

Validate Kubernetes manifests (a directory or one multi-document file) with
[kubeconform](https://github.com/yannh/kubeconform). Fails when a resource is invalid.

```bash
# Directory, default schemas, strict
dagger -m github.com/hampusctl/daggerverse/kubeconform call validate --manifests=./manifests

# Rendered file, CRDs from the datreeio catalog, skip Secrets
dagger -m github.com/hampusctl/daggerverse/kubeconform call validate \
  --file=dist/cluster.yaml --skip=Secret --ignore-missing-schemas \
  --schema-locations='default,https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json'
```

Image: `ghcr.io/yannh/kubeconform:v0.8.0-alpine` (override with `--repository`/`--tag`).
