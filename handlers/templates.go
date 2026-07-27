package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"
	"time"
)

var templates *template.Template

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(filename), "..", "templates")
	funcMap := template.FuncMap{
		"formatDate": func(t interface{}) string {
			switch v := t.(type) {
			case time.Time:
				if v.IsZero() {
					return ""
				}
				return v.Format("2006-01-02")
			default:
				return ""
			}
		},
		"add": func(a, b int) int { return a + b },
	}
	templates = template.Must(
		template.New("").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html")),
	)
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}
