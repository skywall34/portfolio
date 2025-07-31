package handlers

import (
	"net/http"
	"time"

	m "github.com/skywall34/portfolio/internal/models"
	"github.com/skywall34/portfolio/templates"
)

type GetHomeHandler struct {}

type GetHomeHandlerParams struct {}

func NewGetHomeHandler() *GetHomeHandler {
	return &GetHomeHandler{}
}


func loadProjects() []m.Project {
    return []m.Project{
        {Title: "Trip Tracker website using Go, Templ, HTMX, and TailwindCSS", Thumbnail: "/static/img/project1.png", Link: "https://fromnto.cloud"},
    }
}

func loadBlogs() []m.Blog {
	return []m.Blog{
		{Title: "Beginner's Guide to HPCs", Thumbnail: "/static/img/blog1.png", Link: "http://localhost:3000/blogs/hpc"},
	}
}

func loadSkills() []m.Skill {
	return []m.Skill {
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
        Title:     "Mike Shin",
        Projects:  loadProjects(),
		Skills:	   loadSkills(),
		Blogs:     loadBlogs(),
        Today:     today,
    }

	err := templates.Home(data).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}