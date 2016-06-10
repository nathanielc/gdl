package main

import (
	"github.com/pkg/errors"
	"golang.org/x/tools/go/vcs"
)

func findRepos(packages []*Package) ([]*vcs.RepoRoot, error) {
	repos := make([]*vcs.RepoRoot, len(packages))
	for i, pkg := range packages {
		if pkg.Standard {
			repos[i] = &vcs.RepoRoot{
				VCS:  &vcs.Cmd{Name: "None"},
				Repo: "standard",
				Root: "standard",
			}
		} else {
			repo, err := vcs.RepoRootForImportPath(pkg.ImportPath, false)
			if err != nil {
				return nil, errors.Wrapf(err, "could not determine repo for %s", pkg.ImportPath)
			}
			repos[i] = repo
		}
	}
	return repos, nil
}
