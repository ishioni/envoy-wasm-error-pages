// Copyright 2020-2024 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"strings"
)

// Config represents the plugin configuration
type Config struct {
	Theme       string
	ShowDetails bool
}

// Parse parses the configuration from YAML content
func Parse(yamlContent []byte) (*Config, error) {
	cfg := &Config{
		Theme:       "cats", // Default to cats theme
		ShowDetails: true,   // Default to true
	}

	// Simple YAML parser for show_details field
	lines := strings.Split(string(yamlContent), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Parse theme
		if strings.HasPrefix(line, "theme:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "theme:"))
			cfg.Theme = value
		}

		// Parse show_details
		if strings.HasPrefix(line, "show_details:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "show_details:"))
			cfg.ShowDetails = value == "true"
		}
	}

	return cfg, nil
}
