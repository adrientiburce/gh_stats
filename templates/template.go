package templates

import (
	"gh_stats/api"
	"html/template"
	"log"
	"net/http"
)

func RenderTemplate(w http.ResponseWriter, repos []api.Repository) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
		return
	}

	// fmt.Printf("Rendering template with repositories: %+v\n", repos) // Debug
	err = tmpl.Execute(w, repos)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Printf("Render error: %v", err)
	}
}
