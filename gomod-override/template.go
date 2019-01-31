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
	Name            string `toml:"name"`
	Branch          string `toml:"branch"`
	Revision        string `toml:"revision"`
	Version         string `toml:"version"`
	Source          string `toml:"source"`
	GomodOverride   bool
	GomodOverridden bool
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
	}
	return constraint, nil
}
