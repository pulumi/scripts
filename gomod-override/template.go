package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type gopkgConstraint struct {
	Name               string `toml:"name"`
	Branch             string `toml:"branch"`
	Revision           string `toml:"revision"`
	Version            string `toml:"version"`
	Source             string `toml:"source"`
	GomodOverride      bool
	GomodOverridden    bool
	GomodExcludePrefix []string
}

func readTemplate(source io.Reader) (gopkgConstraint, []byte, error) {
	templateText, err := ioutil.ReadAll(source)
	if err != nil {
		return gopkgConstraint{}, nil, errors.Wrap(err, "error reading template TOML")
	}

	gopkgTree, err := toml.LoadReader(bytes.NewReader(templateText))
	if err != nil {
		return gopkgConstraint{}, nil, errors.Wrap(err, "error decoding template TOML")
	}

	// Then, find the first constraint that has "gomod-override" metadata and return it
	if constraints, ok := gopkgTree.Get("constraint").([]*toml.Tree); ok {
		for _, raw := range constraints {
			c, err := decodeConstraint(raw)
			if err != nil {
				log.Fatalf("error decoding constraint: %v", err)
			}
			if c.GomodOverride {
				return c, templateText, nil
			}
		}
	}

	return gopkgConstraint{}, nil, errors.New("no package has gomod-override specified")
}

func decodeConstraint(tree *toml.Tree) (gopkgConstraint, error) {
	var constraint gopkgConstraint
	if err := tree.Unmarshal(&constraint); err != nil {
		return gopkgConstraint{}, err
	}

	if metadata, ok := tree.Get("metadata").(*toml.Tree); ok {
		constraint.GomodOverride = metadata.Has("gomod-override")

		if metadata.Has("gomod-exclude-prefixes") {
			toExclude, err := interfaceToStringArray(metadata.Get("gomod-exclude-prefixes"))
			if err != nil {
				return gopkgConstraint{}, err
			}

			constraint.GomodExcludePrefix = toExclude
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
