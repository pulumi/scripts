package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/dep"
	"github.com/golang/dep/gps"
	"github.com/pkg/errors"

	"github.com/pulumi/scripts/gomod-override/modfile"
	"github.com/pulumi/scripts/gomod-override/module"
	"github.com/pulumi/scripts/gomod-override/semver"
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
		log.Fatalf("error creating golang/dep source manager: %v", err)
	}

	gomodData, err := fetchGoModData(sm, toOverride)
	if err != nil {
		log.Fatal(err)
	}

	overrides, err := readGoMod(bytes.NewReader(gomodData))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(templateText))
	writeOverrides(overrides)
}

func buildGopkgConstraint(req module.Version) (gopkgConstraint, error) {
	// First check if we should pin to a particular SHA
	if prerelease := semver.Prerelease(req.Version); prerelease != "" {
		// Separate the date from the SHA
		versionComponents := strings.Split(prerelease, "-")
		if len(versionComponents) != 3 {
			return gopkgConstraint{}, fmt.Errorf("unexpected prerelease format: %q", prerelease)
		}

		// Return the SHA
		return gopkgConstraint{
			Name:     req.Path,
			Revision: versionComponents[2],
		}, nil
	}

	// If not, we can take the version and constrain to that. If using
	// "+incompatible" syntax, we can strip that off
	return gopkgConstraint{
		Name:    req.Path,
		Version: "=" + strings.TrimSuffix(req.Version, "+incompatible"),
	}, nil
}

func readGoMod(reader io.Reader) ([]gopkgConstraint, error) {
	modFileData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read go.mod data")
	}

	modFile, err := modfile.Parse("go.mod", modFileData, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse go.mod data")
	}

	modFile.SortBlocks()

	var overrides []gopkgConstraint
	for _, req := range modFile.Require {
		override, err := buildGopkgConstraint(req.Mod)
		if err != nil {
			return nil, errors.Wrap(err, "cannot build override from module information")
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
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

func fetchGoModData(sm gps.SourceManager, constraint gopkgConstraint) ([]byte, error) {
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
		return nil, err
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
		return nil, errors.Errorf("no version found for %v with constraint %v",
			constraint.Name, matcher)
	}

	// First export the project so that we can get at its vendor directory.
	exportDir, err := ioutil.TempDir("", "gomod-override")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(exportDir)
	}()

	if err = sm.ExportProject(context.TODO(), projectIdentifier, version, exportDir); err != nil {
		return nil, err
	}

	// Read all of the go.mod file and return it
	gomodData, err := ioutil.ReadFile(filepath.Join(exportDir, "go.mod"))
	if err != nil {
		return nil, err
	}

	return gomodData, nil
}
