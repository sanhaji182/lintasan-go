package server

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dashboard/templates/* dashboard/templates/pages/* dashboard/templates/components/*
var dashboardFS embed.FS

type DashboardEngine struct {
	templates *template.Template
}

func NewDashboardEngine() (*DashboardEngine, error) {
	funcMap := template.FuncMap{
		"uppercase": strings.ToUpper,
		"lowercase": strings.ToLower,
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse base
	baseBytes, err := dashboardFS.ReadFile("dashboard/templates/base.gohtml")
	if err != nil {
		return nil, err
	}
	tmpl, err = tmpl.New("base.gohtml").Parse(string(baseBytes))
	if err != nil {
		return nil, err
	}

	// Parse all page templates
	pagesDir, _ := fs.Sub(dashboardFS, "dashboard/templates/pages")
	pageFiles, _ := fs.Glob(pagesDir, "*.gohtml")
	for _, f := range pageFiles {
		data, err := fs.ReadFile(pagesDir, f)
		if err != nil {
			continue
		}
		tmpl, err = tmpl.New("pages/" + f).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	// Parse all component templates
	compsDir, _ := fs.Sub(dashboardFS, "dashboard/templates/components")
	compFiles, _ := fs.Glob(compsDir, "*.gohtml")
	for _, f := range compFiles {
		data, err := fs.ReadFile(compsDir, f)
		if err != nil {
			continue
		}
		tmpl, err = tmpl.New("components/" + f).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	return &DashboardEngine{templates: tmpl}, nil
}

// RenderPage renders a page template within the base layout.
func (d *DashboardEngine) RenderPage(w http.ResponseWriter, pageName string, data any) {
	fullName := "pages/" + pageName + ".gohtml"
	var buf strings.Builder
	err := d.templates.ExecuteTemplate(&buf, fullName, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = d.templates.ExecuteTemplate(w, "base.gohtml", map[string]any{
		"Title":   pageName,
		"Content": template.HTML(buf.String()),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// RenderPartial renders just a component or page partial (for HTMX swaps).
func (d *DashboardEngine) RenderPartial(w http.ResponseWriter, name string, data any) {
	err := d.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
