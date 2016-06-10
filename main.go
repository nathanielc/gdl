package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

var includeStandard = flag.Bool("s", false, "Include dependencies from the standard Go libraries.")
var includeTest = flag.Bool("t", false, "Include dependencies from the tests.")
var includeRootDepsOnly = flag.Bool("r", false, "Include only dependencies at the root of a repo.")

func main() {
	flag.Parse()
	args := flag.Args()
	var paths []string
	if l := len(args); l > 0 {
		paths = args
	} else {
		paths = []string{"."}
	}
	deps, err := findDeps(*includeStandard, *includeTest, paths...)
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
