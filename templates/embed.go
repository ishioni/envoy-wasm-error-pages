package templates

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed *.html
var TemplatesFS embed.FS

func GetTemplate(theme string) ([]byte, error) {
	filename := theme
	if len(filename) < 5 || filename[len(filename)-5:] != ".html" {
		filename = filename + ".html"
	}

	data, err := TemplatesFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", theme, err)
	}
	return data, nil
}

func GetTemplateNames() ([]string, error) {
	entries, err := fs.ReadDir(TemplatesFS, ".")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 5 && e.Name()[len(e.Name())-5:] == ".html" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}
