package server

import (
	"regexp"
	"strings"
)

const WebURLVariablePrefix = "NWM_WEB_URL_"

var nonVariableCharacters = regexp.MustCompile(`[^A-Z0-9]+`)

func WebURLVariable(packageName string) string {
	return WebURLVariablePrefix + nonVariableCharacters.ReplaceAllString(strings.ToUpper(packageName), "_")
}

func ApplyWebURLs(directory string, urls map[string]string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	config, err := LoadConfig(directory)
	if err != nil {
		return nil, err
	}
	environment := make([]string, 0, len(urls))
	for packageName, url := range urls {
		variable := WebURLVariable(packageName)
		if err := config.Set("custom_settings."+variable, url); err != nil {
			return nil, err
		}
		environment = append(environment, variable+"="+url)
	}
	if err := config.Save(); err != nil {
		return nil, err
	}
	return environment, nil
}
