package handlers

import (
	"net/http"
	"time"

	m "github.com/skywall34/portfolio/internal/models"
	"github.com/skywall34/portfolio/templates"
)

type GetHomeHandler struct{}

type GetHomeHandlerParams struct{}

func NewGetHomeHandler() *GetHomeHandler {
	return &GetHomeHandler{}
}

func loadProjects() []m.Project {
	return []m.Project{
		{Title: "Trip Tracker website using Go, Templ, HTMX, and TailwindCSS", Thumbnail: "/static/img/fromntoproject.png", Link: "https://fromnto.cloud"},
		{Title: "Portfolio Website", Thumbnail: "/static/img/portfolioproject.png", Link: "https://github.com/skywall34/portfolio"},
		{Title: "System Monitor TUI", Thumbnail: "/static/img/sysmonproject.png", Link: "https://github.com/skywall34/sysmon"},
	}
}

func loadBlogs() []m.Blog {
	return []m.Blog{
		{Title: "Beginner's Guide to HPCs", Thumbnail: "/static/img/blog1.png", Link: "/blogs/hpc", Tags: []string{"HPC", "Learning", "SLURM"}},
		{Title: "Setting Up SLURM Database (slurmdbd) - HPC Series Part 2", Thumbnail: "/static/img/blog1.png", Link: "/blogs/slurmdb", Tags: []string{"HPC", "SLURM", "Database", "Accounting"}},
		{Title: "Setting Up SLURM REST API (slurmrestd) - HPC Series Part 3", Thumbnail: "/static/img/blog1.png", Link: "/blogs/slurmrestd", Tags: []string{"HPC", "SLURM", "REST API", "JWT", "Authentication"}},
		{Title: "5 Essential Steps to Build a Powerful Trip Tracker Web App with Go and HTMX", Thumbnail: "/static/img/project1.png", Link: "/blogs/triptracker", Tags: []string{"Go", "HTMX", "Web Development", "SQLite", "Travel"}},
		{Title: "Building a System Monitor using Rust", Thumbnail: "/static/img/blog1.png", Link: "/blogs/sysmon", Tags: []string{"Rust", "Linux", "TUI", "Systems"}},
	}
}

func loadSkills() []m.Skill {
	return []m.Skill{
		{Name: "GoLang", Icon: "https://cdn.simpleicons.org/go"},
		{Name: "Kubernetes", Icon: "https://cdn.simpleicons.org/kubernetes/326CE5"},
		{Name: "Python", Icon: "https://cdn.simpleicons.org/python"},
		{Name: "Typescript", Icon: "https://cdn.simpleicons.org/typescript"},
		{Name: "Kotlin", Icon: "https://cdn.simpleicons.org/kotlin"},
		{Name: "Rust", Icon: "https://cdn.simpleicons.org/rust/ffffff"},
		{Name: "Kafka", Icon: "https://cdn.simpleicons.org/apachekafka/ffffff"},
	}
}

func (h *GetHomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	today := time.Now()

	data := m.PageData{
		Title:    "Mike Shin",
		Projects: loadProjects(),
		Skills:   loadSkills(),
		Blogs:    loadBlogs(),
		Today:    today,
	}

	err := templates.Home(data).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
