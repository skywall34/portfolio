package models

import "time"

type PageData struct {
    Title    string
    Projects []Project
    Blogs    []Blog
    Skills   []Skill
    Today    time.Time
}

type Blog struct {
    ID        string   `json:"id"`
    Title     string   `json:"title"`
    Thumbnail string   `json:"thumbnail"`
    Link      string   `json:"link"`
    Date      string   `json:"date"`
    Tags      []string `json:"tags"`
    Summary   string   `json:"summary"`
    Image     string   `json:"image"`
    PrevPost  string   `json:"prev_post,omitempty"`
    NextPost  string   `json:"next_post,omitempty"`
}

type Project struct {
    Title     string // project title
    Thumbnail string // path to thumbnail image
    Link      string // Link to deployed project if exists
}

type Skill struct {
    Name    string
    Icon    string // path to Icon image
}