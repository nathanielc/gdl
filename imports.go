package main

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// Taken from golang.org/go/cmd/go
// A Package describes a single package found in a directory.
type Package struct {
	// Note: These fields are part of the go command's public API.
	// See list.go. It is okay to add fields, but not to change or
	// remove existing ones. Keep in sync with list.go
	Dir           string `json:",omitempty"` // directory containing package sources
	ImportPath    string `json:",omitempty"` // import path of package in dir
	ImportComment string `json:",omitempty"` // path in import comment on package statement
	Name          string `json:",omitempty"` // package name
	Doc           string `json:",omitempty"` // package documentation string
	Target        string `json:",omitempty"` // install path
	Shlib         string `json:",omitempty"` // the shared library that contains this package (only set when -linkshared)
	Goroot        bool   `json:",omitempty"` // is this package found in the Go root?
	Standard      bool   `json:",omitempty"` // is this package part of the standard Go library?
	Stale         bool   `json:",omitempty"` // would 'go install' do anything for this package?
	StaleReason   string `json:",omitempty"` // why is Stale true?
	Root          string `json:",omitempty"` // Go root or Go path dir containing this package
	ConflictDir   string `json:",omitempty"` // Dir is hidden by this other directory
	BinaryOnly    bool   `json:",omitempty"` // package cannot be recompiled
	// Source files
	GoFiles        []string `json:",omitempty"` // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles       []string `json:",omitempty"` // .go sources files that import "C"
	IgnoredGoFiles []string `json:",omitempty"` // .go sources ignored due to build constraints
	CFiles         []string `json:",omitempty"` // .c source files
	CXXFiles       []string `json:",omitempty"` // .cc, .cpp and .cxx source files
	MFiles         []string `json:",omitempty"` // .m source files
	HFiles         []string `json:",omitempty"` // .h, .hh, .hpp and .hxx source files
	FFiles         []string `json:",omitempty"` // .f, .F, .for and .f90 Fortran source files
	SFiles         []string `json:",omitempty"` // .s source files
	SwigFiles      []string `json:",omitempty"` // .swig files
	SwigCXXFiles   []string `json:",omitempty"` // .swigcxx files
	SysoFiles      []string `json:",omitempty"` // .syso system object files added to package
	// Cgo directives
	CgoCFLAGS    []string `json:",omitempty"` // cgo: flags for C compiler
	CgoCPPFLAGS  []string `json:",omitempty"` // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string `json:",omitempty"` // cgo: flags for C++ compiler
	CgoFFLAGS    []string `json:",omitempty"` // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string `json:",omitempty"` // cgo: flags for linker
	CgoPkgConfig []string `json:",omitempty"` // cgo: pkg-config names
	// Dependency information
	Imports []string `json:",omitempty"` // import paths used by this package
	Deps    []string `json:",omitempty"` // all (recursively) imported dependencies
	// Error information
	Incomplete bool            `json:",omitempty"` // was there an error loading this package or dependencies?
	Error      *PackageError   `json:",omitempty"` // error loading this package (not dependencies)
	DepsErrors []*PackageError `json:",omitempty"` // errors loading dependencies
	// Test information
	TestGoFiles  []string `json:",omitempty"` // _test.go files in package
	TestImports  []string `json:",omitempty"` // imports from TestGoFiles
	XTestGoFiles []string `json:",omitempty"` // _test.go files outside package
	XTestImports []string `json:",omitempty"` // imports from XTestGoFiles

	Vendored bool
}

// A PackageError describes an error loading information about a package.
type PackageError struct {
	ImportStack   []string // shortest path from package named on command line to this one
	Pos           string   // position of error
	Err           string   // the error itself
	isImportCycle bool     // the error is an import cycle
	hard          bool     // whether the error is soft or hard; soft errors are ignored in some places
}

// List names of packages from import paths.
func listPackages(importPaths ...string) ([]string, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	args := append([]string{"list", "-e"}, importPaths...)
	cmd := exec.Command("go", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "initializing stdout for go list cmd")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting go list cmd")
	}
	packages := make([]string, 0, len(importPaths))
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		packages = append(packages, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "uncountered err reading command output")
	}
	if err := cmd.Wait(); err != nil {
		return nil, errors.Wrap(err, "go list cmd failed")
	}
	return packages, nil
}

// List information on a specific package or a wildcard match.
func listPackageDetails(currentPath string, importPaths ...string) (map[string]*Package, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	vendoredPath := path.Join(currentPath, "vendor")
	args := append([]string{"list", "-e", "-json"}, importPaths...)
	cmd := exec.Command("go", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "initializing stdout for go list cmd")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting go list cmd")
	}
	packages := make(map[string]*Package)
	dec := json.NewDecoder(stdout)
	for dec.More() {
		p := &Package{}
		if err := dec.Decode(p); err != nil {
			return nil, errors.Wrap(err, "invalid json go list cmd")
		}
		// Rewrite vendored packages
		if strings.HasPrefix(p.ImportPath, vendoredPath) {
			p.ImportPath = p.ImportPath[len(vendoredPath)+1:]
			p.Vendored = true
		}
		packages[p.ImportPath] = p
	}
	if err := cmd.Wait(); err != nil {
		return nil, errors.Wrap(err, "go list cmd failed")
	}
	return packages, nil
}

type Packages []*Package

func (p Packages) Len() int           { return len(p) }
func (p Packages) Less(i, j int) bool { return p[i].ImportPath < p[j].ImportPath }
func (p Packages) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func findDeps(standards, tests bool, importPaths ...string) (Packages, error) {
	currentPackages, err := listPackages(".")
	if err != nil {
		return nil, errors.Wrap(err, "listing current package")
	}
	if len(currentPackages) != 1 {
		return nil, errors.New("extra results getting current package")
	}
	currentPackage := currentPackages[0]
	packages, err := listPackageDetails(currentPackage, importPaths...)
	if err != nil {
		return nil, errors.Wrap(err, "listing packages")
	}

	// List of all deps
	deps := make(Packages, 0, len(packages)*3)
	// Keep track of included deps
	included := make(map[string]bool, len(packages)*3)
	// Helper to add dep
	addDep := func(dp *Package) {
		if !included[dp.ImportPath] && (standards || !dp.Standard) && !strings.HasPrefix(dp.ImportPath, currentPackage) {
			deps = append(deps, dp)
		}
		// Mark as included, even if not actaully added because now we know it won't need to be added.
		included[dp.ImportPath] = true
	}
	if tests {
		testPackageSet := make(map[string]struct{})
		for _, pkg := range packages {
			for _, list := range [][]string{pkg.TestImports, pkg.XTestImports} {
				for _, path := range list {
					if _, ok := packages[path]; !ok {
						testPackageSet[path] = struct{}{}
					}
				}
			}
		}
		testImports := make([]string, len(testPackageSet))
		for path := range testPackageSet {
			testImports = append(testImports, path)
		}
		testPackages, err := listPackageDetails(currentPackage, testImports...)
		if err != nil {
			return nil, errors.Wrap(err, "listing test packages")
		}
		for path, pkg := range testPackages {
			packages[path] = pkg
			addDep(pkg)
		}
	}
	// Set of deps that where not already listed
	missingSet := make(map[string]struct{})

	// Collect all deps
	for _, pkg := range packages {
		for _, dep := range pkg.Deps {
			dp, ok := packages[dep]
			if !ok {
				missingSet[dep] = struct{}{}
			} else {
				addDep(dp)
			}
		}

	}

	// List any dep packages there weren't already listed
	if len(missingSet) > 0 {
		paths := make([]string, len(missingSet))
		for path := range missingSet {
			paths = append(paths, path)
		}
		dps, err := listPackageDetails(currentPackage, paths...)
		if err != nil {
			return nil, errors.Wrap(err, "listing missing packages")
		}
		for _, dp := range dps {
			addDep(dp)
		}
	}
	sort.Sort(deps)

	return deps, nil
}
