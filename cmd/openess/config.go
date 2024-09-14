package main

import (
	"encoding/json"
	"openess/internal/collector"
	"openess/internal/export"
	"io"
	"os"
)

type Config struct {
	BindPort   int
	DeviceAddr string
	ProtoPath  string
	Collector  collector.Config
	Export     export.Config
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
