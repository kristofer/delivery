package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateDirFallsBackToWorkingDirectoryTemplates(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "templates"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	if got := templateDir("/src/handlers/templates.go"); got != "templates" {
		t.Fatalf("templateDir() = %q, want %q", got, "templates")
	}
}

func TestTemplateDirUsesSourceRelativeTemplatesWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	handlersDir := filepath.Join(tmp, "handlers")
	templatesDir := filepath.Join(tmp, "templates")

	if err := os.MkdirAll(handlersDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	filename := filepath.Join(handlersDir, "templates.go")
	if got := templateDir(filename); got != templatesDir {
		t.Fatalf("templateDir() = %q, want %q", got, templatesDir)
	}
}
