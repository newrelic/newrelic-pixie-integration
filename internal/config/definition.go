package config

import (
	"github.com/newrelic/newrelic-pixie-integration/internal/script"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const scriptExtension = ".yaml"

func ReadScriptDefinitions(dir string) ([]*script.ScriptDefinition, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var l []*script.ScriptDefinition
	for _, file := range files {
		if strings.HasSuffix(file.Name(), scriptExtension) {
			description, err := readScriptDefinition(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			l = append(l, description)
		}
	}
	return l, nil
}

func readScriptDefinition(path string) (*script.ScriptDefinition, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var definition script.ScriptDefinition
	err = yaml.Unmarshal(content, &definition)
	if err != nil {
		return nil, err
	}
	return &definition, nil
}
