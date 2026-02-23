# Melange + Apko Example

This example shows how to combine [Melange](https://github.com/chainguard-dev/melange) and [Apko](https://github.com/chainguard-dev/apko) in a Dagger module:

- build an APK with Melange
- build a container with Apko that installs that APK
- publish the generated container image to a registry

The example module is in `melange/examples/go/main.go`, with packaging and image configuration in `melange/examples/melange.yaml` and `melange/examples/apko.yaml`.

## Prerequisites

- Run commands from the root of the `daggerverse` repository.
- Use a registry address you can push to (for `publish --address`).
- This example config is currently set to `aarch64` in both `melange.yaml` and `apko.yaml`.

## Run The Example

```shell
dagger call \
  -m melange/examples/go \
  --apko-file melange/examples/apko.yaml \
  --melange-config melange/examples/melange.yaml \
  container \
  publish --address <REGISTRY/IMAGE:TAG>
```

## What This Command Does

The command runs the full flow in one call:

1. Melange builds a signed `hello` APK (using the Melange config).
2. Apko builds a container image and installs that package.
3. `container publish --address ...` publishes the generated container image to your registry.

## Notes

- This is an example workflow to demonstrate how Melange and Apko can be combined in a module.
- In this setup, the package build steps are defined in `melange/examples/melange.yaml`.
- The resulting image contents are defined in `melange/examples/apko.yaml`.

For customization, edit those files and rerun the command.
