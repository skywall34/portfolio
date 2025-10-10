package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/skywall34/portfolio/internal/handlers"
)

func main() {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// UI Handlers
	mux.HandleFunc("/", handlers.NewGetHomeHandler().ServeHTTP)

	// API Calls
	mux.Handle("/blogs/", handlers.NewGetBlogHandler())

	// Server
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%s", appPort),
		Handler: mux,
	}

	fmt.Printf("Server running on port :%s\n", appPort)
	server.ListenAndServe()
}
