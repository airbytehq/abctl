package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type FlagContext struct {
	Type    string   `yaml:"type"`
	Include []string `yaml:"include"`
	Serve   any      `yaml:"server"`
}

type Flag struct {
	Name    string       `yaml:"name"`
	Serve   any          `yaml:"serve"`
	Context *FlagContext `yaml:"context,omitempty"`
}

type FlagsYml struct {
	Flags []Flag `yaml:"flags"`
}

func NewFlagsYml(flags ...Flag) *FlagsYml {
	return &FlagsYml{
		Flags: flags,
	}
}

func (f *FlagsYml) Flag(name string, serve any, context *FlagContext) *FlagsYml {
	f.Flags = append(f.Flags, Flag{Name: name, Serve: serve, Context: context})
	return f
}

func (f *FlagsYml) WriteTo(path string) error {
	outf, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create `flags.yml` file at path: %w", err)
	}

	b, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml for `flags.yml`: %w", err)
	}

	_, err = outf.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write file contents to %s: %w", path, err)
	}

	return err
}
