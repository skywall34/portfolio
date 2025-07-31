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
    Title     string
    Thumbnail string
    Link      string
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