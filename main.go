package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"gh_stats/api"
	"gh_stats/config"
	"gh_stats/model"
	"gh_stats/stats"
	"gh_stats/templates"
)

func main() {
	start := time.Now()
	config.LoadEnv()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Load templates
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	// Repos caching
	var repos []model.Repository
	var stats *stats.RenderedStats
	var reposMutex sync.Mutex

	go func() {
		defer logTime("Api call", time.Now())
		var client = api.NewGithubClient()
		fetchedRepos := api.GetTopRepositories(client)

		// log.Printf("fetched %d repos", fetchedRepos)
		reposMutex.Lock()
		repos = fetchedRepos
		stats = client.Stats.GetRenderedStats()
		reposMutex.Unlock()
	}()

	// Define the handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reposMutex.Lock()
		defer reposMutex.Unlock()

		data := templates.GetData(repos, stats)
		tmpl.Execute(w, data)
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started in %v, listening on port %s\n", time.Since(start), port)
	http.ListenAndServe(":"+port, nil)
}

func logTime(name string, start time.Time) {
	elapsed := time.Since(start)
	log.Printf("%s took: %s \n", name, elapsed)
}
