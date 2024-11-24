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
)

func main() {
	start := time.Now()
	config.LoadEnv()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Load templates
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	// Repos caching
	var repos []api.Repository
	var reposMutex sync.Mutex

	go func() {
		defer logTime("Api call", time.Now())
		fetchedRepos := api.GetTopRepositories()

		// log.Printf("fetched %d repos", fetchedRepos)
		reposMutex.Lock()
		repos = fetchedRepos
		reposMutex.Unlock()
	}()

	// Define the handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reposMutex.Lock()
		defer reposMutex.Unlock()

		// fmt.Printf("passing template repositories: %+v\n", repos) // Debug
		tmpl.Execute(w, repos)
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
