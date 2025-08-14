package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"regexp"

	"github.com/a-h/templ"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/skywall34/portfolio/internal/models"
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

	// Parse blog metadata from markdown frontmatter
	blog, content, err := h.parseBlogPost(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Add navigation links
	blog.PrevPost, blog.NextPost = h.getAdjacentPosts(slug)

	// Render with enhanced template
	err = templates.BlogPage(blog, content).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

func (h *GetBlogHandler) parseBlogPost(slug string) (models.Blog, templ.Component, error) {
	mdPath := filepath.Join("static", "content", "blogs", fmt.Sprintf("%s.md", slug))
	
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		return models.Blog{}, nil, err
	}

	content := string(mdContent)
	
	// Extract metadata from content
	blog := models.Blog{
		ID: slug,
		Title: h.extractTitle(content),
		Tags: h.extractTags(content),
		Date: h.extractDate(content),
	}

	// Remove metadata lines from content for markdown conversion
	markdownContent := h.cleanContentForMarkdown(content)

	// Convert markdown to HTML
	htmlContent := h.convertMarkdown(markdownContent)
	
	return blog, htmlContent, nil
}

func (h *GetBlogHandler) extractTitle(content string) string {
	// Look for first # heading
	re := regexp.MustCompile(`(?m)^# (.+)$`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return "Untitled"
}

func (h *GetBlogHandler) extractTags(content string) []string {
	// Look for "Tags: tag1, tag2, tag3"
	re := regexp.MustCompile(`(?m)^Tags:\s*(.+)$`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		tagStr := strings.TrimSpace(matches[1])
		tags := strings.Split(tagStr, ",")
		var cleanTags []string
		for _, tag := range tags {
			cleanTags = append(cleanTags, strings.TrimSpace(tag))
		}
		return cleanTags
	}
	return []string{}
}

func (h *GetBlogHandler) extractDate(content string) string {
	// For now, return empty string - can be enhanced later with frontmatter
	return ""
}

func (h *GetBlogHandler) cleanContentForMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		// Skip Tags line
		if strings.HasPrefix(line, "Tags:") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	
	return strings.Join(cleanLines, "\n")
}

func (h *GetBlogHandler) convertMarkdown(content string) templ.Component {
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
	if err := gmarkdown.Convert([]byte(content), &htmlBuf); err != nil {
		return Unsafe("Error rendering markdown: " + err.Error())
	}

	return Unsafe(htmlBuf.String())
}

func (h *GetBlogHandler) getAdjacentPosts(currentSlug string) (prevPost, nextPost string) {
	// Get list of all blog files
	blogsDir := filepath.Join("static", "content", "blogs")
	files, err := os.ReadDir(blogsDir)
	if err != nil {
		return "", ""
	}

	var blogSlugs []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".md") {
			slug := strings.TrimSuffix(file.Name(), ".md")
			blogSlugs = append(blogSlugs, slug)
		}
	}

	// Find current position
	currentIndex := -1
	for i, slug := range blogSlugs {
		if slug == currentSlug {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return "", ""
	}

	// Get adjacent posts
	if currentIndex > 0 {
		prevPost = blogSlugs[currentIndex-1]
	}
	if currentIndex < len(blogSlugs)-1 {
		nextPost = blogSlugs[currentIndex+1]
	}

	return prevPost, nextPost
}
