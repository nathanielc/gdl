package main

import (
	"encoding/json"
	"os/exec"
	"sort"

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
}

// A PackageError describes an error loading information about a package.
type PackageError struct {
	ImportStack   []string // shortest path from package named on command line to this one
	Pos           string   // position of error
	Err           string   // the error itself
	isImportCycle bool     // the error is an import cycle
	hard          bool     // whether the error is soft or hard; soft errors are ignored in some places
}

// List information on a specific package or a wildcard match.
func listPackages(importPaths ...string) (map[string]*Package, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	args := append([]string{"list", "-json"}, importPaths...)
	cmd := exec.Command("go", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrapf(err, "initializing stdout for go list cmd with import path %v", importPaths[0])
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrapf(err, "starting go list cmd for import path %v", importPaths[0])
	}
	packages := make(map[string]*Package)
	dec := json.NewDecoder(stdout)
	for dec.More() {
		p := &Package{}
		if err := dec.Decode(p); err != nil {
			return nil, errors.Wrapf(err, "invalid json go list cmd for import path %v", importPaths[0])
		}
		packages[p.ImportPath] = p
	}
	if err := cmd.Wait(); err != nil {
		return nil, errors.Wrapf(err, "go list cmd failed for import path %v", importPaths[0])
	}
	return packages, nil
}

type Packages []*Package

func (p Packages) Len() int           { return len(p) }
func (p Packages) Less(i, j int) bool { return p[i].ImportPath < p[j].ImportPath }
func (p Packages) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func findDeps(importPath string, standards, tests bool) (Packages, error) {
	packages, err := listPackages(importPath)
	if err != nil {
		return nil, errors.Wrapf(err, "listing packages for path %s", importPath)
	}

	deps := make(Packages, 0, len(packages)*3)

	packageList := make([]string, 0, len(packages))
	for path := range packages {
		packageList = append(packageList, path)
	}
	finished := make(map[string]bool)

	missing := []string{}

	for {
		i := len(packageList)
		if i == 0 {
			break
		}
		path := packageList[i-1]
		finished[path] = true
		pkg := packages[path]
		packageList = packageList[:i-1]
		for _, dep := range pkg.Deps {
			dp, ok := packages[dep]
			if !ok {
				missing = append(missing, dep)

			} else {
				if standards || !dp.Standard {
					deps = append(deps, dp)
				}
			}
		}

		if len(missing) > 0 {
			dps, err := listPackages(missing...)
			if err != nil {
				return nil, errors.Wrapf(err, "listing dependent package for path %s", importPath)
			}
			for dep, dp := range dps {
				// We only need the information about the package
				// It doesn't need to be added to packageList since dependencies
				// have already been fully resolved.
				packages[dep] = dp
				if standards || !dp.Standard {
					deps = append(deps, dp)
				}
			}
		}
		if tests {
			testImports := append(pkg.TestImports, pkg.XTestImports...)
			for _, testImport := range testImports {
				tpkgs, err := listPackages(testImport)
				if err != nil {
					return nil, errors.Wrapf(err, "listing test packages for path %s", testImport)
				}
				for tpath, tpkg := range tpkgs {
					if finished[tpath] {
						continue
					}
					packageList = append(packageList, tpath)
					if standards || !tpkg.Standard {
						deps = append(deps, tpkg)
					}
				}
			}
		}
	}
	sort.Sort(deps)

	return deps, nil
}
