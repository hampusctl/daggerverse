# kustomize

`kustomize build` and `kustomize edit set image`. The container is Alpine plus the official
release binary for the engine's platform (the `registry.k8s.io` image is amd64-only).

```bash
# Render a kustomization
dagger -m github.com/hampusctl/daggerverse/kustomize call build --source=. --path=deploy/overlays/prod export --path=out.yaml

# Pin an image by digest; returns the updated source directory
dagger -m github.com/hampusctl/daggerverse/kustomize call set-image --source=. --dir=deploy/base \
  --images='controller=ghcr.io/org/app@sha256:...' export --path=.
```

Version: `--version=v5.8.3` (default).
