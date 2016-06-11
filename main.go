package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func usage() {
	u := `Usage: gdl [OPTIONS] [PACKAGES..]

	List dependencies of Go packages.
	This utility is a light wrapper around the 'go list' command,
	intended to provide easy access to the dependencies of a project.

Examples:

	List all dependencies of the current package.
	
		gdl
	
	List all dependencies of the current package and all sub packages, (including any vendored packages).

		gdl ./...
	
	List all dependencies of the local sub package ./cmd/foo package.

		gdl ./cmd/foo
	
	List all dependencies of the current package and all sub packages skipping any locally vendored packages.

		gdl -no-vendored ./...

Options:
`
	fmt.Fprintf(os.Stderr, u)
	flag.PrintDefaults()
}

var includeStandard = flag.Bool("std", false, "Include dependencies from the standard Go libraries.")
var includeTest = flag.Bool("test", false, "Include dependencies from tests files.")
var includeRootDepsOnly = flag.Bool("repo", false, "Include only the first dependency per repo.")
var skipVendored = flag.Bool("no-vendored", false, "Skip any packages that are vendored below the current package.")

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	var paths []string
	if l := len(args); l > 0 {
		paths = args
	} else {
		paths = []string{"."}
	}
	deps, err := findDeps(*includeStandard, *includeTest, *skipVendored, paths...)
	if err != nil {
		log.Fatal(err)
	}
	repos, err := findRepos(deps)
	if err != nil {
		log.Fatal(err)
	}

	rows := make([][]string, 1, len(deps)+1)
	rows[0] = []string{
		"ImportPath",
		"Vendored",
		"Root",
		"VCS",
		"Repo",
		"Error",
	}
	roots := make(map[string]bool, len(repos))
	rootOnly := *includeRootDepsOnly
	for i := range deps {
		if rootOnly && deps[i].ImportPath != repos[i].Root && roots[repos[i].Root] && !deps[i].Standard {
			continue
		}
		roots[repos[i].Root] = true
		v := "no"
		if deps[i].Vendored {
			v = "yes"
		}
		errStr := ""
		if deps[i].Error != nil {
			n := strings.IndexByte(deps[i].Error.Err, '\n')
			errStr = deps[i].Error.Err[:n]
		}
		rows = append(rows, []string{
			deps[i].ImportPath,
			v,
			repos[i].Root,
			repos[i].VCS.Name,
			repos[i].Repo,
			errStr,
		})
	}
	printTable(rows)
}

func printTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	cols := make([]int, len(rows[0]))
	for _, row := range rows {
		for c, col := range row {
			if l := len(col); l > cols[c] {
				cols[c] = l + 1
			}
		}
	}

	colFmts := make([]string, len(cols))
	for i, col := range cols {
		colFmts[i] = fmt.Sprintf("%%-%ds", col+1)
	}

	for _, row := range rows {
		for c, col := range row {
			fmt.Printf(colFmts[c], col)
		}
		fmt.Println()
	}
}
