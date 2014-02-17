package capstan

import (
	"fmt"
	"os"
	"errors"
	"github.com/kylelemons/go-gypsy/yaml"
	"strings"
)

type Config struct {
	Base    string
	Cmdline string
	Build   string
	Files   map[string]string
}

func ConfigExists(filename string) bool {
	_, err := yaml.ReadFile(filename)
	return err == nil
}

func ReadConfig(filename string) (*Config, error) {
	config, err := yaml.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseConfig(config)
}

func ParseConfig(config *yaml.File) (*Config, error) {
	base, err := config.Get("base")
	if err != nil {
		return nil, err
	}
	cmdline, err := config.Get("cmdline")
	if err != nil {
		return nil, err
	}
	build, _ := config.Get("build")
	filesNode, err := yaml.Child(config.Root, "files")
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	if filesNode != nil {
		filesMap := filesNode.(yaml.Map)
		for key, value := range filesMap {
			scalar := value.(yaml.Scalar)
			files[key] = strings.TrimSpace(scalar.String())
		}
	}
	result := &Config{
		Base:    base,
		Cmdline: cmdline,
		Build:   build,
		Files:   files,
	}
	return result, nil
}

func (config *Config) Check(r *Repo) error {
	if _, err := os.Stat(r.ImagePath(config.Base)); os.IsNotExist(err) {
		return r.PullImage(config.Base)
	}
	for _, value := range config.Files {
		if _, err := os.Stat(value); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("%s: no such file or directory", value))
		}
	}
	return nil
}
