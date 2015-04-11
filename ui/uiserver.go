package ui

import (
	"html/template"
	"net/http"

	"github.com/222Labs/common/go/logging"
	"github.com/222Labs/common/go/templateutil"
)

var (
	log = logging.GetLogger("ui")

	templateDir = "/home/ajvb/Code/kala/src/ui/templates/"
	templates   = MustParseTemplates(templateDir)
)

// Helpers

func MustParseTemplates(templateDir string) *template.Template {
	files, err := templateutil.RecursiveHTMLFinder(templateDir)
	if err != nil {
		log.Fatalf("Error while loading in template: %s", err)
	}
	return template.Must(template.ParseFiles(files...))
}

func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl+".html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Handlers

func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO - Remove, only for dev.
	templates = MustParseTemplates(templateDir)

	renderTemplate(w, "dashboard")
}
