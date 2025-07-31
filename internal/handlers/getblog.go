package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/skywall34/portfolio/templates"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

type GetBlogHandler struct{}

func NewGetBlogHandler() *GetBlogHandler {
	return &GetBlogHandler{}
}

func Unsafe(html string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		_, err = io.WriteString(w, html)
		return
	})
}

func (h *GetBlogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// URL path: /blogs/{id}
	slug := r.URL.Path[len("/blogs/"):]
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	mdPath := filepath.Join("static", "content", "blogs", fmt.Sprintf("%s.md", slug))

	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	gmarkdown := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(true),
				),
			),
		),
	)

	var htmlBuf bytes.Buffer

	if err := gmarkdown.Convert(mdContent, &htmlBuf); err != nil {
		http.Error(w, "Error rendering markdown", http.StatusInternalServerError)
		return
	}

	// Create an unsafe component containing raw HTML.
	content := Unsafe(htmlBuf.String())

	err = templates.BlogPage(content).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}
