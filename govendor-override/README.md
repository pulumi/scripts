# govendor-override

`govendor-override` is a tool to produce `Gopkg.toml` constraint files for use with [`dep`][dep] from
the `vendor.json` file contained in repositories which use [`govendor`][govendor] for dependency
management,  in order to constrain the dependencies used in a larger project to those locked by
Govendor. This is useful for building Pulumi providers based on the Terraform bridge, where the
known good set of dependencies are those vendored into the upstream Terraform provider, and other
dependencies for the most part have versions which float with respect to upstream.

It is known not work with repositories which _rely_ on the ability of `govendor` to vendor different
packages from the same repository at different versions.

## Building

`govendor-override` uses Go Modules to manage it's own dependencies.

_Note: With Go 1.11.x, you must enable Go modules by setting `GO111MODULE=on` in your environment if
the repository is checked out in the `GOPATH` format._

Build `govendor-override` by changing into the checkout directory, and running `go install`. This
will install to `$(go env GOPATH)/bin` by default, though the build path can be modified using the
`-O` flag to `go install`.

## Using `govendor-override`

`govendor-override` reads a template `Gopkg.toml` file via `stdin`, and emits a new `Gopkg.toml` file
to `stdout` containing the new dependencies. Instructions for which repositories to extract
dependencies from are contained in the `constraint.metadata` sections of the TOML file.

For example, an input template file like the following indicates that we want to emit constraints
for dependencies of `github.com/pulumi/terraform-provider-aws`, with the exception of dependencies
which have a path prefixed with `github.com/golang/protobuf`.

```toml
[[constraint]]
  branch = "master"
  name = "github.com/pulumi/pulumi"

[[constraint]]
  branch = "master"
  name = "github.com/pulumi/pulumi-terraform"

[[constraint]]
  branch = "pulumi-master"
  name = "github.com/terraform-providers/terraform-provider-aws"
  source = "github.com/pulumi/terraform-provider-aws"

  [constraint.metadata]
    govendor-override = true
    govendor-exclude-prefixes = ["github.com/golang/protobuf"]
```

Assuming this template file is stored in the target repository as `Gopkg.template.toml`, the
following workflow in a repository using `dep`, but pulling in a project using `govendor` will
produce a usable `Gopkg.toml` file which can then be used to lock floating dependencies via `dep
ensure`:

```shell
$ rm Gopkg.{lock,toml}
$ govendor-override < Gopkg.template.toml > Gopkg.toml
# Warnings and information are emmited to stderr
$ dep ensure -v

$ go build
```

## Warning

There may be constraints that you do not want to include, as they prevent a build working. Add new
prefixes to the `govendor-exclude-prefixes` array for the target constraint in order to prevent
these from being generated.

This process can take several attempts to get right at first, but assuming no major dependency
changes is mostly mechanical afterwards.

[dep]: https://github.com/golang/dep
[govendor]: https://github.com/kardianos/govendor
