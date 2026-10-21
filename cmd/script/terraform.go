package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var terraformTemplates embed.FS

var templates = template.Must(template.New("terraform").Funcs(template.FuncMap{
	"ruleName": formatRuleName,
	"quote": func(value string) string {
		return strings.NewReplacer("${", "$${", "%{", "%%{").Replace(strconv.Quote(value))
	},
}).ParseFS(terraformTemplates, "templates/*.tmpl"))

func writeTerraformFile(directory, resourceName, templateName string, data any) error {
	var content bytes.Buffer
	if err := templates.ExecuteTemplate(&content, templateName, data); err != nil {
		return fmt.Errorf("generar %s: %w", resourceName, err)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("crear directorio Terraform: %w", err)
	}
	path := filepath.Join(directory, resourceName+".tf")
	if err := os.WriteFile(path, content.Bytes(), 0o644); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	return nil
}
