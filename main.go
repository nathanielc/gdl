package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var includeStandard = flag.Bool("s", false, "Include dependencies from the standard Go libraries.")
var includeTest = flag.Bool("t", false, "Include dependencies from the tests.")

func main() {
	flag.Parse()
	args := flag.Args()
	path := "."
	if l := len(args); l == 1 {
		path = args[0]
	} else if l > 1 {
		flag.Usage()
		os.Exit(1)
	}
	deps, err := findDeps(path, *includeStandard, *includeTest)
	if err != nil {
		log.Fatal(err)
	}
	repos, err := findRepos(deps)
	if err != nil {
		log.Fatal(err)
	}
	outFmt := "%-30s%-50s%-10s%-50s\n"
	fmt.Printf(outFmt, "ImportPath", "Repo", "VCS", "Root")
	for i := range deps {
		fmt.Printf(outFmt, deps[i].ImportPath, repos[i].Repo, repos[i].VCS.Name, repos[i].Root)
	}
}
