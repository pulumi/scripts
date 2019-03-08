package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang/dep"
)

func main() {
	toOverride, templateText, err := readTemplate(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	ctx := &dep.Ctx{
		GOPATH: os.Getenv("GOPATH"),
	}
	sm, err := ctx.SourceManager()
	if err != nil {
		log.Fatalf("error creating golang/dependency source manager: %v", err)
	}

	dependencyInfo, err := fetchDependencyInfo(sm, toOverride)
	if err != nil {
		log.Fatal(err)
	}

	dependencies, err := parseDependencies(dependencyInfo)
	if err != nil {
		log.Fatal(err)
	}

	var overrides []gopkgConstraint
	for _, dependency := range dependencies {
		override, err := dependency.ToGoPkgConstraint()
		if err != nil {
			log.Fatal(err)
		}

		shouldIgnore := false
		for _, toIgnore := range toOverride.GomodExcludePrefix {
			if strings.HasPrefix(override.Name, toIgnore) {
				shouldIgnore = true
			}
		}

		if shouldIgnore {
			log.Printf("Ignoring module because of gomod-exclude-prefixes: %s", override.Name)
		} else {
			overrides = append(overrides, override)
		}
	}

	fmt.Print(string(templateText))
	writeOverrides(overrides)
}

func writeOverrides(overrides []gopkgConstraint) {
	fmt.Printf("\n# NOTE: this Gopkg.toml file was constructed using gomod-override\n")

	for _, override := range overrides {
		fmt.Printf("\n")
		fmt.Printf("[[override]]\n")
		fmt.Printf("  name = %q\n", override.Name)
		if override.Version != "" {
			fmt.Printf("  version = %q\n", override.Version)
		} else {
			fmt.Printf("  revision = %q\n", override.Revision)
		}
		fmt.Printf("  [override.metadata]\n")
		fmt.Printf("    gomod-overridden=true\n")
	}
}
