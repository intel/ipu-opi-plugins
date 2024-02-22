package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Host struct {
		Ip string `yaml:"ip"`
	} `yaml:"host"`
	ACC struct {
		Ip string `yaml:"ip"`
	} `yaml:"acc"`
}

func readConfig(cfg *Config, path string) {
	f, err := os.Open(path)
	if err != nil {
		println(err.Error())
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		println(err.Error())
	}
}
