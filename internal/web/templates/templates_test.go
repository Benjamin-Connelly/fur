package templates

import (
	"strings"
	"testing"
)

// TestPageTemplatesParse confirms init() parsed every page template against the
// base layout (guards template parse-ability and exercises the funcMap).
func TestPageTemplatesParse(t *testing.T) {
	want := []string{"directory.html", "markdown.html", "code.html", "graph.html"}
	for _, name := range want {
		tmpl, ok := PageTemplates[name]
		if !ok || tmpl == nil {
			t.Errorf("PageTemplates[%q] missing", name)
			continue
		}
		// Each page must carry the base layout definition.
		if tmpl.Lookup("base.html") == nil {
			t.Errorf("%q is not cloned from base.html", name)
		}
	}

	// Render the directory template with a minimal data set to prove the parsed
	// funcMap (add/repeat/trimPrefix) and base layout execute.
	var sb strings.Builder
	data := map[string]any{
		"Title":       "t",
		"Path":        "/",
		"Entries":     []any{},
		"Breadcrumbs": []any{},
	}
	if err := PageTemplates["directory.html"].ExecuteTemplate(&sb, "base.html", data); err != nil {
		t.Fatalf("executing directory template: %v", err)
	}
	if sb.Len() == 0 {
		t.Error("directory template rendered empty output")
	}
}
