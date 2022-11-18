package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Aws              *AWSsqsConfig `yaml:"aws"`
	LogFilePath      string        `yaml:"logFile"`
	ClientsInputPath string        `yaml:"clientsInputPath"`
}

type AWSsqsConfig struct {
	QueueUrl     string `yaml:"url"`
	Region       string `yaml:"region"`
	ClientId     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	ClientToken  string `yaml:"clientToken"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Substitute from environemental vars
	confContent := []byte(os.ExpandEnv(string(data)))

	config := &Config{}

	err = yaml.Unmarshal(confContent, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
