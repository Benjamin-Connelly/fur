package templates

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed *.html
var templateFS embed.FS

// PageTemplates holds separately-parsed templates for each page type.
// Each page template is cloned from the base so that {{define "content"}}
// blocks don't overwrite each other.
var PageTemplates map[string]*template.Template

func init() {
	funcMap := template.FuncMap{
		"add":        func(a, b int) int { return a + b },
		"repeat":     strings.Repeat,
		"trimPrefix": strings.TrimPrefix,
	}

	// Parse the base template first
	baseContent, _ := templateFS.ReadFile("base.html")
	base := template.Must(template.New("base.html").Funcs(funcMap).Parse(string(baseContent)))

	// Each page template gets its own clone of the base
	pages := []string{"directory.html", "markdown.html", "code.html", "graph.html"}
	PageTemplates = make(map[string]*template.Template, len(pages))

	for _, page := range pages {
		t := template.Must(base.Clone())
		pageContent, _ := templateFS.ReadFile(page)
		template.Must(t.Parse(string(pageContent)))
		PageTemplates[page] = t
	}
}
