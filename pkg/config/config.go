package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
)

type APIConfig struct {
	Port int `json:"port"`
}

type DatabaseConfig struct {
	Hostname      string `json:"hostname"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Database      string `json:"database"`
	Port          int64  `json:"port"`
	EncryptionKey string `json:"encryption_key"`
}

type Databases struct {
	Gamejam *DatabaseConfig
}

type Config struct {
	API       *APIConfig `json:"api"`
	Databases *Databases `json:"databases"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}