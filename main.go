package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
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
	paths := make([]string, 0, len(deps))
	for path := range deps {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Printf("%s\n", path)
	}
}
