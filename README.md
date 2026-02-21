# Daggerverse Modules

My personal Dagger modules I use in my day-to-day life. A small set of composable building blocks for SBOM, vulnerability scanning, license compliance, and image building—each runs in containers and can be wired into pipelines.

## Modules

| Module | Description | Status |
|--------|-------------|--------|
| **apko** | Build minimal container images from Apko YAML configs (Build, ShowConfig, ShowPackages). | ✅ |
| **melange** | Build packages for Apko (Melange). | ❌ scaffold |
| **syft** | Generate SBOM from container images (Anchore Syft). | ✅ |
| **grype** | Vulnerability scan for SBOMs; Markdown report (Anchore Grype). | ✅ |
| **grant** | License compliance check for SBOMs; `report.json` + `report.md` (Anchore Grant). | ✅ |
| **terraform** | Run Terraform in a container (plan, apply, init with optional args). | ✅ |

**Status:** ✅ has real content · ❌ scaffold only

## Usage

Use [dagger/dagger-for-github](https://github.com/dagger/dagger-for-github) in GitHub Actions with `module: github.com/hampusctl/daggerverse/<module>@main`. Examples:

**Build with apko and publish**

```yaml
- name: Build with apko image and publish
  uses: dagger/dagger-for-github@v8.2.0
  with:
    version: "latest"
    module: github.com/hampusctl/daggerverse/apko@main
    call: |-
      build \
      --config-file=apko/container.yaml \
      publish \
      --address=${{ env.IMAGE_REF }}
```

**Scan image with Syft (SBOM)**

```yaml
- name: Scan apko image
  uses: dagger/dagger-for-github@v8.2.0
  with:
    version: "latest"
    module: github.com/hampusctl/daggerverse/syft@main
    call: |-
      scan --image=${{ env.IMAGE_REF }} \
      export \
      --path=syft-output
```

**Scan SBOM with Grype (vulnerabilities)**

```yaml
- name: Scan SBOM with Grype
  uses: dagger/dagger-for-github@v8.2.0
  with:
    version: "latest"
    module: github.com/hampusctl/daggerverse/grype@main
    call: |-
      scan \
      --sbom=syft-output/sbom.json \
      --config=.grype.yaml \
      --template=.templates/grype.tmpl \
      export \
      --path=grype-output
```

**License check with Grant**

```yaml
- name: License check with Grant
  uses: dagger/dagger-for-github@v8.2.0
  with:
    version: "latest"
    module: github.com/hampusctl/daggerverse/grant@main
    call: |-
      check \
      --sbom=syft-output/sbom.json \
      --config=.grant.yaml \
      export \
      --path=grant-output
```

## Contributing

This is a monorepo; each module is a separate Go module. To contribute to a module, open a pull request. All additions are highly appreciated.
