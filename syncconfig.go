package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

const syncConfigFile = "frond.sync.yaml"

type syncConfig struct {
	GitHub *syncConfigGitHub `yaml:"github"`
}

type syncConfigGitHub struct {
	Server               string  `yaml:"server"`
	SearchQuery          string  `yaml:"searchQuery"`
	SingleDirForAllRepos bool    `yaml:"singleDirForAllRepos,omitempty"`
	OrgPrefixSeparator   *string `yaml:"orgPrefixSeparator,omitempty"`
}

func readSyncConfig() (syncConfig, error) {
	var cfg syncConfig

	file, err := os.Open(syncConfigFile)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	dec.SetStrict(true)

	err = dec.Decode(&cfg)
	return cfg, err
}

func writeSyncConfig(cfg syncConfig) error {
	file, err := os.Create(syncConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	defer enc.Close()

	return enc.Encode(cfg)
}
