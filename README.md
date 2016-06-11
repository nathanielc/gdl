# gdl

A tool for listing Go dependencies.
This utility is a light wrapper around the 'go list' command, intended to provide easy access to the dependencies of a project.
The utility is `vendor` aware meaning it will correctly find and interpret any dependencies that you may have vendored, independent of the vendoring method.

# Examples

List dependencies of the current package.

    gdl

List dependencies of the current package and all sub packages, (including any vendored packages).

    gdl ./...

Using this project as an example:

```
$ gdl ./... # from within $GOPATH/src/github.com/nathanielc/gdl
ImportPath                 Vendored  Root                   VCS  Repo                               Error
github.com/pkg/errors      yes       github.com/pkg/errors  Git  https://github.com/pkg/errors
golang.org/x/tools/go/vcs  no        golang.org/x/tools     Git  https://go.googlesource.com/tools
```

List dependencies of the local sub package ./cmd/foo package.

    gdl ./cmd/foo

List dependencies of the current package and all sub packages skipping any locally vendored packages.

    gdl -no-vendored ./...

List dependencies and test dependencies of the current package and all sub packages.

    gdl -test ./...

List only the first dependency per VCS repo.

    gdl -repo ./...

List dependencies including dependencies from the standard Go library.

    gdl -std ./...


And putting it all together, list all dependencies in such away that you can script vendoring of dependencies.

    gdl -no-vendored -repo -test ./...

# Installation

Install via Go:

    go get -u github.com/nathanielc/gdl

## Arch Linux

AUR package coming soon

# Motivation

I find most tools to manage dependencies for a Go project opaque and inconsistent.
Instead of trying to solve the entire problem of Go dependency management this tool tries to improve one aspect, visibility into dependencies of a Go project.
The utility is written with scripting in mind, so that the heavy lifting can be performed by external tools.

