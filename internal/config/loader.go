package config

import (
	"io/fs"
	"regexp"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	"gopkg.in/yaml.v3"
)

//nolint:gocritic // Wrong suggested replacement for regexp
var yamlConfigFileRegexp = regexp.MustCompile(`^(?:[a-z][_a-zA-Z]+?)(?:-(.*?)|)\.(?:yaml|yml)`)

type YamlConfigFSLoader struct{}

var _, _ = any(&YamlConfigFSLoader{}).(ConfigurationParser)

var _ ConfigurationParser = &YamlConfigFSLoader{}

// TODO : Replace configHolder and defaultConfig with a func() ???

//nolint:errcheck,revive // Needed for walkDir
func (this *YamlConfigFSLoader) ParseConfigurationFilesystem(fsys fs.FS, loader func(m map[string]any, profile string)) {
	fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, _ error) error {
		if !entry.Type().IsRegular() {
			return nil
		}

		m := yamlConfigFileRegexp.FindStringSubmatch(entry.Name())
		if len(m) == 0 {
			logger.GetLogger().Debug("Unable to load configuration file", "name", entry.Name())
			return nil
		}

		if data, err := fs.ReadFile(fsys, path); err == nil {
			config := map[string]any{}
			if err := yaml.Unmarshal(data, &config); err == nil {
				logger.GetLogger().Info("Loading configuration", "path", path, "profile", m[1])
				loader(config, m[1])
				/*
					if m[1] == "" {
						if defaultConfig {
							logger.GetLogger().Info("Loading default configuration", "path", path)
							configHolder.LoadDefaultConfig(config)
						} else {
							logger.GetLogger().Info("Loading configuration", "path", path)
							configHolder.LoadConfig(config)
						}
					} else {
						configHolder.LoadConfigForProfile(config, m[1])
					}
				*/
			}
		}
		return nil
	})
}
