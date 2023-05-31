// SPDX-FileCopyrightText: 2021 Michael Seplowitz
// SPDX-License-Identifier: MIT

package sync

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const syncConfigFile = "frond.sync.yaml"

var errNoConfigFileFound = fmt.Errorf("no sync config file found")

// workDir should be absolute
func findConfigFile(workDir string) (string, error) {
	if workDir == "" {
		panic("empty workDir")
	}
	if !filepath.IsAbs(workDir) {
		panic(fmt.Sprintf("workDir should be absolute, got %q", workDir))
	}

	curDir := workDir

	for {
		path := filepath.Join(curDir, syncConfigFile)
		_, err := os.Stat(path)
		if err == nil {
			return path, nil
		}

		if !os.IsNotExist(err) {
			return "", fmt.Errorf("unexpected error type: %T %w", err, err)
		}

		if filepath.Dir(curDir) == curDir {
			return "", errNoConfigFileFound
		}

		curDir = filepath.Dir(curDir)
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

func parseConfigFromFile(path string) (syncConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return syncConfig{}, err
	}
	defer file.Close()

	return parseConfig(file)
}

func writeConfig(cfg syncConfig) error {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	file, err := os.Create(syncConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(&buf)
	nullKeySuffix := []byte(": null")
	for scanner.Scan() {
		if bytes.HasSuffix(scanner.Bytes(), nullKeySuffix) {
			fmt.Fprintf(file, "%s:\n", scanner.Bytes()[:len(scanner.Bytes())-len(nullKeySuffix)])
			continue
		}

		fmt.Fprintf(file, "%s\n", scanner.Bytes())
	}
	return scanner.Err()
}

//------------------------------------------------------------------------------

func validateConfig(cfg syncConfig) error {
	if cfg.GitHub != nil {
		return cfg.GitHub.validate()
	}

	return fmt.Errorf("empty config")
}
