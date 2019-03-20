package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/semver"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type dependency struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
}

func (dep dependency) ToGoPkgConstraint() (gopkgConstraint, error) {
	// First check if we should pin to a particular SHA
	if prerelease := semver.Prerelease(dep.Version); prerelease != "" {
		// Separate the date from the SHA
		versionComponents := strings.Split(prerelease, "-")
		switch len(versionComponents) {
		case 2:
			// This is not in the Go mod exact SHA format - use the whole prerelease version
			return gopkgConstraint{
				Name:    dep.Path,
				Version: dep.Version,
			}, nil
		case 3:
			// Return the SHA for the specific version of dependency
			sha := versionComponents[2]
			if len(sha) < 40 {
				// Resolve the abbreviated SHA via `go get`
				var err error
				sha, err = dep.resolveAbbreviatedSHA(sha)
				if err != nil {
					return gopkgConstraint{}, errors.Wrapf(err, "error resolving abbreviated SHA for %q", dep.Path)
				}
			}

			// Return the SHA
			return gopkgConstraint{
				Name:     dep.Path,
				Revision: sha,
			}, nil
		default:
			return gopkgConstraint{}, fmt.Errorf("unexpected prerelease format for %s: %q",
				dep.Path, prerelease)
		}
	}

	// If not, we can take the version and constrain to that. If using
	// "+incompatible" syntax, we can strip that off
	return gopkgConstraint{
		Name:    dep.Path,
		Version: "=" + strings.TrimSuffix(dep.Version, "+incompatible"),
	}, nil
}

func (dep dependency) resolveAbbreviatedSHA(revision string) (string, error) {
	tempGoPath := os.Getenv("GOMOD_OVERRIDE_GOPATH")

	if tempGoPath == "" {
		var err error
		tempGoPath, err = ioutil.TempDir("", "gomod-override")
		if err != nil {
			return "", err
		}
		defer func() {
			_ = os.RemoveAll(tempGoPath)
		}()
	}

	args := []string{"get", "-u", "-d", dep.Path}
	if os.Getenv("GOMOD_OVERRIDE_ALLOW_INSECURE") != "" {
		args = append(args[:2], append([]string{"-insecure"}, args[2:]...)...)
	}

	log.Printf("Running go %s in temporary GOPATH: %s", strings.Join(args, " "), tempGoPath)
	goGetCmd := exec.Command("go", args...)

	env := os.Environ()
	for i, key := range env {
		if strings.HasPrefix(key, "GOPATH=") {
			env = append(env[:i], env[i+1:]...)
		}
	}
	env = append(env, fmt.Sprintf("GOPATH=%s", tempGoPath))
	goGetCmd.Env = env

	output, err := goGetCmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(output), fmt.Sprintf("no Go files in %s", tempGoPath)) &&
			!strings.Contains(string(output), fmt.Sprintf("build constraints exclude all Go files")) {
			return "", fmt.Errorf("cannot go get %s:\n%s\n", dep.Path, string(output))
		}
	}

	repo, err := git.PlainOpen(filepath.Join(tempGoPath, "src", dep.Path))
	if err != nil {
		return "", err
	}

	commitIter, err := repo.CommitObjects()
	if err != nil {
		return "", err
	}

	var desiredCommit *object.Commit
	err = commitIter.ForEach(func(commit *object.Commit) error {
		if strings.HasPrefix(commit.Hash.String(), revision) {
			desiredCommit = commit
		}
		return nil
	})

	if desiredCommit == nil {
		return "", fmt.Errorf("no commit maches prefix %s", revision)
	}

	log.Printf("Expanded commit SHA %s to %s", revision, desiredCommit.Hash.String())

	return desiredCommit.Hash.String(), nil
}
