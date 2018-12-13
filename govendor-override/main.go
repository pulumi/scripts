package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/golang/dep"
	"github.com/golang/dep/gps"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type gopkgConstraint struct {
	Name                  string `toml:"name"`
	Branch                string `toml:"branch"`
	Revision              string `toml:"revision"`
	Version               string `toml:"version"`
	Source                string `toml:"source"`
	GovendorOverride      bool
	GovendorOverridden    bool
	GovendorExcludePrefix []string
}

// govendorFile is a complete vendor/vendor.json file.  See https://github.com/kardianos/vendor-spec for details.
type govendorFile struct {
	Package []govendorPackage
}

// govendorPackage is an individual lock for a package.  Note that Govendor specifically permits different locks
// for packages within the same repo; Dep does not.  See https://github.com/kardianos/vendor-spec for details.
type govendorPackage struct {
	Path         string
	Version      string
	VersionExact string
	Revision     string
	RevisionTime time.Time
}

func fetchGovendorFile(sm gps.SourceManager, constraint gopkgConstraint) (govendorFile, error) {
	// Pull the project identifier and version out of the constraint.
	projectIdentifier := gps.ProjectIdentifier{
		ProjectRoot: gps.ProjectRoot(constraint.Name),
		Source:      constraint.Source,
	}

	var matcher gps.Constraint
	switch {
	case constraint.Branch != "":
		matcher = gps.NewBranch(constraint.Branch)
	case constraint.Version != "":
		if v, err := gps.NewSemverConstraintIC(constraint.Version); err == nil {
			matcher = v
		} else {
			matcher = gps.NewVersion(constraint.Version)
		}
	case constraint.Revision != "":
		matcher = gps.Revision(constraint.Revision)
	default:
		matcher = gps.Any()
	}

	versions, err := sm.ListVersions(projectIdentifier)
	if err != nil {
		return govendorFile{}, err
	}
	gps.SortPairedForUpgrade(versions)

	var version gps.Version
	for _, v := range versions {
		if matcher.Matches(v) {
			log.Printf("chose %v@%v\n", constraint.Name, v)
			version = v
			break
		}
	}
	if version == nil {
		return govendorFile{}, errors.Errorf("no version found for %v with constraint %v", constraint.Name, matcher)
	}

	// First export the project so that we can get at its vendor directory.
	exportDir, err := ioutil.TempDir("", "govendor-override")
	if err != nil {
		return govendorFile{}, err
	}
	defer os.RemoveAll(exportDir)

	if err = sm.ExportProject(context.TODO(), projectIdentifier, version, exportDir); err != nil {
		return govendorFile{}, err
	}

	// Now open and parse the govendor file.
	govendorJson, err := os.Open(filepath.Join(exportDir, "vendor", "vendor.json"))
	if err != nil {
		return govendorFile{}, err
	}
	defer govendorJson.Close()

	var gov govendorFile
	if err = json.NewDecoder(govendorJson).Decode(&gov); err != nil {
		return govendorFile{}, err
	}

	return gov, nil
}

func decodeConstraint(tree *toml.Tree) (gopkgConstraint, error) {
	var constraint gopkgConstraint
	if err := tree.Unmarshal(&constraint); err != nil {
		return gopkgConstraint{}, err
	}

	if metadata, ok := tree.Get("metadata").(*toml.Tree); ok {
		constraint.GovendorOverride = metadata.Has("govendor-override")
		// There was a bug in an older version of this govendor-override that misspelled overridden - check for
		// that for a while until any produces files are likely out of use.
		constraint.GovendorOverridden = metadata.Has("govendor-overridden") || metadata.Has("govendor-overriden")

		if metadata.Has("govendor-exclude-prefixes") {
			toExclude, err := interfaceToStringArray(metadata.Get("govendor-exclude-prefixes"))
			if err != nil {
				return gopkgConstraint{}, err
			}

			constraint.GovendorExcludePrefix = toExclude
		}
	}
	return constraint, nil
}

// Convert an interface{} (as provided by the TOML library) to a []string
func interfaceToStringArray(input interface{}) ([]string, error) {
	interfaceArray, ok := input.([]interface{})
	if !ok {
		return nil, errors.New("expected a string array")
	}

	var stringArray []string
	for _, item := range interfaceArray {
		if stringItem, ok := item.(string); ok {
			stringArray = append(stringArray, stringItem)
		} else {
			return nil, errors.New("expected a string array")
		}
	}
	return stringArray, nil
}

func main() {
	// First, read the Gopkg.toml.
	gopkgTree, err := toml.LoadReader(os.Stdin)
	if err != nil {
		log.Fatalf("error decoding Gopkg.toml: %v", err)
	}

	// Then, find any constraints that have "govendor-override" metadata.
	var inject []gopkgConstraint
	if constraints, ok := gopkgTree.Get("constraint").([]*toml.Tree); ok {
		for _, raw := range constraints {
			c, err := decodeConstraint(raw)
			if err != nil {
				log.Fatalf("error decoding constraint: %v", err)
			}
			if c.GovendorOverride {
				inject = append(inject, c)
			}
		}
	}

	// Create a dep source manager.  This allows us to use the same logic dep does for canonicalizing
	// packages to project names.
	ctx := &dep.Ctx{
		GOPATH: os.Getenv("GOPATH"),
	}
	sm, err := ctx.SourceManager()
	if err != nil {
		log.Fatalf("error creating golang/dep source manager: %v", err)
	}

	govendorFiles := make([]govendorFile, len(inject))
	for i, constraint := range inject {
		log.Printf("fetching govendor information for %v", constraint.Name)
		gov, err := fetchGovendorFile(sm, constraint)
		if err != nil {
			log.Fatalf("error fetching govendor information for %v: %v", constraint.Name, err)
		}
		govendorFiles[i] = gov
	}

	// Now go through and canonicalize each package to its project.  Note that there may be many package
	// directives in a given vendor.json file.  We will just emit project-level references.  In the event
	// that there are contradictions, we will issue a warning, but generally choose the latest.
	var projects []gps.ProjectRoot
	versions := make(map[gps.ProjectRoot]govendorPackage)
	for i, gov := range govendorFiles {
		// This is not ideal but avoids restructuring for now - look up the inject metadata by parallel array index
		ignored := inject[i].GovendorExcludePrefix

		for _, pkg := range gov.Package {
			ignoreThisPackage := false
			for _, ignore := range ignored {
				if strings.HasPrefix(pkg.Path, ignore) {
					ignoreThisPackage = true
				}
			}
			if ignoreThisPackage {
				fmt.Fprintf(os.Stderr, "info: ignoring package %s\n", pkg.Path)
				continue
			}

			proj, err := sm.DeduceProjectRoot(pkg.Path)
			if err != nil {
				log.Fatalf("error deducing project root from path %s: %v", pkg.Path, err)
			}

			v, has := versions[proj]
			if has {
				// We've seen this already.  If the versions agree, no problem, we can keep going along on our
				// merry old way.  Otherwise, problems abound; if its time is sooner, it wins.  Either way, warn.
				if pkg.Version != v.Version || pkg.Revision != v.Revision {
					var choose string
					if pkg.RevisionTime.After(v.RevisionTime) {
						choose = pkg.Path
						versions[proj] = pkg
					} else {
						choose = v.Path
					}
					fmt.Fprintf(os.Stderr, "warning: %s version conflict (%s@%s%s != %s@%s%s); chose %s (sooner)\n",
						proj, v.Path, v.Version, v.Revision, pkg.Path, pkg.Version, pkg.Revision, choose)
				}
			} else {
				// We haven't seen this yet; simply append it.
				versions[proj] = pkg
				projects = append(projects, proj)
			}
		}
	}

	// Finally, scrape old injected overrides from the Gopkg.toml, then write the tree, sort the deps, and inject new
	// overrides.
	if overrides, ok := gopkgTree.Get("override").([]*toml.Tree); ok {
		newOverrides := make([]*toml.Tree, 0, len(overrides))
		for _, raw := range overrides {
			c, err := decodeConstraint(raw)
			if err != nil {
				log.Fatalf("error decoding override: %v", err)
			}
			if !c.GovendorOverridden {
				newOverrides = append(newOverrides, raw)
			}
		}
		gopkgTree.Set("override", newOverrides)
	}
	gopkgStr, err := gopkgTree.ToTomlString()
	if err != nil {
		log.Fatalf("writing new Gopkg.toml: %v", err)
	}
	fmt.Print(strings.TrimLeftFunc(gopkgStr, unicode.IsSpace))

	if len(projects) != 0 {
		sort.Slice(projects, func(i, j int) bool {
			return projects[i] < projects[j]
		})

		fmt.Printf("\n# NOTE: the following overrides were injected by govendor-override. It may be necessary to")
		fmt.Printf("\n# remove some of these overrides in order to produce a buildable vendor tree.")
		for _, proj := range projects {
			fmt.Printf("\n")
			v := versions[proj]
			fmt.Printf("[[override]]\n")
			fmt.Printf("  name = \"%s\"\n", proj)
			if v.Version != "" {
				if v.Version == "master" {
					fmt.Printf("  branch = \"%s\"\n", v.Version)
				} else {
					fmt.Printf("  version = \"=%s\"\n", v.Version)
				}
			} else {
				fmt.Printf("  revision = \"%s\"\n", v.Revision)
			}
			fmt.Printf("  [override.metadata]\n")
			fmt.Printf("    govendor-overridden=true\n")
		}
	}
}
