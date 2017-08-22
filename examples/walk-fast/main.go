package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karrick/godirwalk"
)

func main() {
	dirname := "."
	if len(os.Args) > 1 {
		dirname = os.Args[1]
	}
	if err := godirwalk.WalkFileMode(filepath.Clean(dirname), callback); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func callback(osPathname string, mode os.FileMode) error {
	fmt.Printf("%s %s\n", mode, osPathname)
	return nil
}