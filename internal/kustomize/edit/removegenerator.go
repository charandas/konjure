// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package edit

import (
	"errors"
	"path/filepath"

	"github.com/carbonrelay/konjure/internal/kustomize/edit/kustinternal"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
)

type removeGeneratorOptions struct {
	generatorFilePaths []string
}

// newCmdRemoveGenerator removes the name of a file containing a generator from the kustomization file.
func newCmdRemoveGenerator(fSys filesys.FileSystem) *cobra.Command {
	var o removeGeneratorOptions

	cmd := &cobra.Command{
		Use: "generator",
		Short: "Removes one or more generator file paths from " +
			konfig.DefaultKustomizationFileName(),
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate(args)
			if err != nil {
				return err
			}
			err = o.Complete(cmd, args)
			if err != nil {
				return err
			}
			return o.RunRemoveGenerator(fSys)
		},
	}
	return cmd
}

// Validate validates removeGenerator command.
func (o *removeGeneratorOptions) Validate(args []string) error {
	if len(args) == 0 {
		return errors.New("must specify a generator file")
	}
	o.generatorFilePaths = args
	return nil
}

// Complete completes removeGenerator command.
func (o *removeGeneratorOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// RunRemoveGenerator runs removeGenerator command (do real work).
func (o *removeGeneratorOptions) RunRemoveGenerator(fSys filesys.FileSystem) error {

	mf, err := kustinternal.NewKustomizationFile(fSys)
	if err != nil {
		return err
	}

	m, err := mf.Read()
	if err != nil {
		return err
	}

	generators, err := globPatterns(m.Generators, o.generatorFilePaths)
	if err != nil {
		return err
	}

	if len(generators) == 0 {
		return nil
	}

	newGenerators := make([]string, 0, len(m.Generators))
	for _, generator := range m.Generators {
		if kustinternal.StringInSlice(generator, generators) {
			continue
		}
		newGenerators = append(newGenerators, generator)
	}

	m.Generators = newGenerators
	return mf.Write(m)
}

func globPatterns(generators []string, patterns []string) ([]string, error) {
	var result []string
	for _, pattern := range patterns {
		for _, generator := range generators {
			match, err := filepath.Match(pattern, generator)
			if err != nil {
				return nil, err
			}
			if !match {
				continue
			}
			result = append(result, generator)
		}
	}
	return result, nil
}
