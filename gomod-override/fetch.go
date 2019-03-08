package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/golang/dep/gps"
	"github.com/pkg/errors"
)

// fetchDependencyInfo uses a SourceManager (from dep) to download the repository
// specified in constraint. It then exports the repository to a temporary
// directory (outside of GOPATH), and runs `go list -json -m all` (with GO111MODULE=on)
// in that temporary directory. It returns the standard output of that command,
// which is a JSON representation of the dependencies for the module.
func fetchDependencyInfo(sm gps.SourceManager, constraint gopkgConstraint) ([]byte, error) {
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

	// Run `go list -m all` and read it's stdout to get the complete list of dependencies
	golistCmd := exec.Command("go", "list", "-json", "-m", "all")
	golistCmd.Dir = exportDir
	golistCmd.Env = os.Environ()

	output, err := golistCmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "error running go list to determine complete dependency list")
	}

	return output, nil
}
