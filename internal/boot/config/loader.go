package config

import (
	"fmt"
	"io/fs"
	"regexp"

	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	"gopkg.in/yaml.v3"
)

// yamlConfigFileRegexp matches configuration file names and extracts optional profile names.
//
//nolint:gocritic // Wrong suggested replacement for regexp
var yamlConfigFileRegexp = regexp.MustCompile(`^(?:[a-z][_a-zA-Z]+?)(?:-(.*?)|)\.(?:yaml|yml)`)

// YamlConfigFSLoader implements core.ConfigurationParser for YAML files located in an fs.FS.
type YamlConfigFSLoader struct{}

var _, _ = any(&YamlConfigFSLoader{}).(ConfigurationParser)

var _ ConfigurationParser = &YamlConfigFSLoader{}

// TODO : Replace configHolder and defaultConfig with a func() ???

// ParseConfigurationFilesystem walks fsys, unmarshaling any matching YAML configuration
// files and invoking loader with the parsed map and extracted profile name.
// Returns an error if filesystem walking, file reading, or YAML unmarshaling fails.
func (this *YamlConfigFSLoader) ParseConfigurationFilesystem(fsys fs.FS, loader func(m map[string]any, profile string)) Error {
	err := fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		m := yamlConfigFileRegexp.FindStringSubmatch(entry.Name())
		if len(m) == 0 {
			services.Get().Logger.Debug("Unable to load configuration file", "name", entry.Name())
			return nil
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		config := map[string]any{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}
		services.Get().Logger.Info("Loading configuration", "path", path, "profile", m[1])
		loader(config, m[1])
		return nil
	})
	if err != nil {
		return NewErrorFrom(err)
	}
	return nil
}
