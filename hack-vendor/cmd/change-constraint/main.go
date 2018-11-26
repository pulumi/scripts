// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

func main() {
	var name string
	var revision string
	var serverPrefix string
	var gopkgFile string

	flag.StringVar(&name, "name", "", "the name of the project to modify")
	flag.StringVar(&revision, "revision", "", "the revision of the project to pin to")
	flag.StringVar(&serverPrefix, "serverPrefix", "http://localhost.pulumi.engineering/", "the url of a git server that exposes $GOPATH")
	flag.StringVar(&gopkgFile, "file", "Gopkg.toml", "the path to the Gopkg.toml to modify")
	flag.Parse()

	if name == "" {
		fmt.Fprintf(os.Stderr, "error: must provide package to modify with -name\n")
		os.Exit(1)
	}

	if revision == "" {
		fmt.Fprintf(os.Stderr, "error: must provide version to use with -version\n")
		os.Exit(1)
	}

	if !strings.HasSuffix(serverPrefix, "/") {
		serverPrefix = serverPrefix + "/"
	}

	b, err := ioutil.ReadFile(gopkgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var gopkg Gopkg

	if err := toml.Unmarshal(b, &gopkg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	hadExisting := false
	for idx, override := range gopkg.Override {
		if override.Name == name {
			newSource := serverPrefix + override.Name

			if override.Source != "" {
				newSource = serverPrefix + override.Source
			}

			gopkg.Override[idx] = Override{
				Name:     name,
				Revision: revision,
				Source:   newSource,
			}

			hadExisting = true
			break
		}
	}

	if !hadExisting {
		gopkg.Override = append(gopkg.Override, Override{
			Name:     name,
			Revision: revision,
			Source:   serverPrefix + name,
		})
	}

	for idx, constraint := range gopkg.Constraint {
		if constraint.Name == name {
			gopkg.Constraint = append(gopkg.Constraint[:idx], gopkg.Constraint[idx+1:]...)
			break
		}
	}

	f, err := os.OpenFile(gopkgFile, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(gopkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Gopkg struct {
	Required   []string   `toml:"required"`
	Ignored    []string   `toml:"ignored"`
	Constraint []Override `toml:"constraint"`
	Override   []Override `toml:"override"`
}

type Override struct {
	Name     string `toml:"name"`
	Version  string `toml:"version,omitempty"`
	Branch   string `toml:"branch,omitempty"`
	Revision string `toml:"revision,omitempty"`
	Source   string `toml:"source,omitempty"`
}
