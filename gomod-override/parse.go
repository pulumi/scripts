package main

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

func parseDependencies(dependencyInfo []byte) ([]dependency, error) {
	var dependencies []dependency
	dec := json.NewDecoder(bytes.NewReader(dependencyInfo))
	for {
		var dep dependency

		err := dec.Decode(&dep)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "error parsing JSON from go list")
		}

		dependencies = append(dependencies, dep)
	}

	// The first JSON object is the module itself, which we don't need
	if len(dependencies) < 2 {
		return []dependency{}, nil
	}

	return dependencies[1:], nil

}
