package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

var (
	Config Conf
)

type Conf struct {
	Listen string `yaml:"listen"`

	Smtp SMTP `yaml:"smtp"`

	Users []USER `yaml:"users"`
}

type SMTP struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     string    `yaml:"port"`
	From     string `yaml:"from"`
	Password string `yaml:"password"`
	TLS      bool   `yaml:"tls"`
	STARTTLS bool   `yaml:"starttls"`
}

type USER struct {
	UID   int    `yaml:"uid"`
	Email string `yaml:"email"`
}

func LoadConfig(filename string) (err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(content, &Config)
	if err != nil {
		return
	}
	return nil
}
