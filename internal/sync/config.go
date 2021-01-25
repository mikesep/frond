package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const syncConfigFile = "frond.sync.yaml"

var errNoConfigFileFound = fmt.Errorf("no sync config file found")

func findConfigFile(dir string) (string, error) {
	var err error

	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not getwd: %w", err)
		}
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed filepath.Abs: %w", err)
	}

	for {
		path := filepath.Join(dir, syncConfigFile)
		_, err := os.Stat(path)
		if err == nil {
			// fmt.Printf("DEBUG: found config at %q\n", path)
			return path, nil
		}

		if !os.IsNotExist(err) {
			return "", fmt.Errorf("unexpected error type: %T %w", err, err)
		}

		if filepath.Dir(dir) == dir {
			return "", errNoConfigFileFound
		}

		dir = filepath.Dir(dir)
	}
}

//------------------------------------------------------------------------------

type syncConfig struct {
	GitHub *gitHubConfig `yaml:"github"`
}

func parseConfig(r io.Reader) (syncConfig, error) {
	var cfg syncConfig

	dec := yaml.NewDecoder(r)
	dec.SetStrict(true)

	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decoding error: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func parseConfigFromFoundFile() (syncConfig, error) {
	path, err := findConfigFile("")
	if err != nil {
		return syncConfig{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return syncConfig{}, err
	}
	defer file.Close()

	return parseConfig(file)
}

func writeConfig(cfg syncConfig) error {
	file, err := os.Create(syncConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	defer enc.Close()

	return enc.Encode(cfg)
}

//------------------------------------------------------------------------------

func validateConfig(cfg syncConfig) error {
	// TODO owner or owners but not both
	// server required
	return nil
}
