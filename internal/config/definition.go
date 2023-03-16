package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/newrelic/newrelic-pixie-integration/internal/script"
)

const scriptExtension = ".yaml"

// ReadScriptDefinitions reads the script definition from the given directory path.
// Only .yaml files are read and subdirectories are not traversed.
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
