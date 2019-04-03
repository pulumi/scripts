# `gomod-doccopy`

Go modules can be used in a mode which populate a `vendor/` directory inside a
repository in a `GOPATH`, preserving much of the workflow of tools such as
`dep`. Unfortunately, it only copies Go files from packages needed to build.

For bridged Terraform providers, we also need the `website/` directory present
in order to create documentation.

This tool recursively copies the `website/` directory from the machine module
cache (which represents a complete version of the repository) to the
appropriate spot in the `vendor` directory for a given Terraform provider.

## Usage

```
gomod-doccopy -provider terraform-provider-xyz [-org not-terraform-providers] [-v]
```

