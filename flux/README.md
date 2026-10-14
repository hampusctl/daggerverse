# flux

Flux CLI operations that need no cluster: push and tag OCI artifacts, and `flux envsubst`.

```bash
# Push a directory as an OCI artifact; prints oci://repo@sha256:...
dagger -m github.com/hampusctl/daggerverse/flux call \
  with-registry-auth --address=ghcr.io --username=$USER --secret=env://GITHUB_TOKEN \
  push-artifact --source=. --url=oci://ghcr.io/org/app-manifests:sha-$SHA \
  --source-url=https://github.com/org/app --revision=main@sha1:$SHA \
  --ignore-paths='/*,!/deploy/'

# Add tags
dagger -m github.com/hampusctl/daggerverse/flux call \
  with-registry-auth --address=ghcr.io --username=$USER --secret=env://GITHUB_TOKEN \
  tag-artifact --url=oci://ghcr.io/org/app-manifests:sha-$SHA --tags=prod

# Substitute ${VAR}s like Flux post-build substitution (strict by default)
dagger -m github.com/hampusctl/daggerverse/flux call envsubst --input=out.yaml --vars=SITE=fra1,CLUSTER=c1 export --path=out.yaml
```

Image: `ghcr.io/fluxcd/flux-cli:v2.9.6` (override with `--repository`/`--tag`).
