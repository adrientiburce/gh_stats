package templates

import (
	"gh_stats/model"
	"gh_stats/stats"
	"html/template"
	"log"
	"net/http"
)

type TemplateData struct {
	Repositories []model.Repository
	Stats        *stats.RenderedStats
}

func GetData(repo []model.Repository, stats *stats.RenderedStats) *TemplateData {
	return &TemplateData{
		Repositories: repo,
		Stats:        stats,
	}
}

func RenderTemplate(w http.ResponseWriter, data *TemplateData) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
		return
	}

	// fmt.Printf("Rendering template with repositories: %+v\n", repos) // Debug
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Printf("Render error: %v", err)
	}
}
