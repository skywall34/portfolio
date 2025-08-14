# Building a Trip Tracker Web App with Go, Templ, and HTMX

In this tutorial, we'll build a simple trip tracking web application using Go's standard library, Templ for templates, HTMX for dynamic interactions, and Tailwind CSS for styling. By the end, you'll have a working web app that can create, read, update, and delete trips.

## What We'll Build

A basic trip tracker where users can:

- View a list of their trips
- Add new trips with origin and destination
- Edit existing trips
- Delete trips

## Prerequisites

- Go 1.23+ installed
- Basic knowledge of Go and HTML
- SQLite3 installed

## Project Setup

Let's start by creating our project structure:

```
trip-tracker/
├── main.go
├── go.mod
├── internal/
│   ├── database/
│   │   ├── database.go
│   │   └── schema.sql
│   └── handlers/
│       ├── home.go
│       ├── trips.go
│       └── trip_handlers.go
├── templates/
│   ├── layout.templ
│   └── trips.templ
└── static/
    ├── css/
    │   ├── input.css
    │   └── output.css
    └── js/
        └── htmx.min.js
```

## Step 1: Initialize the Project

```bash
mkdir trip-tracker && cd trip-tracker
go mod init trip-tracker
```

Install required dependencies:

**Install SQLite (Quick and reliable storage):**

```bash
sudo apt update
sudo apt install sqlite3
sqlite3 --version
```

**Install Templ (For Template generation):**

```bash
go get github.com/a-h/templ/cmd/templ@latest
```

## Step 2: Database Setup

Create the database schema in `internal/database/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS trips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    origin TEXT NOT NULL,
    destination TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert some sample data
INSERT INTO trips (title, origin, destination, start_date, end_date) VALUES
('Tokyo Adventure', 'LAX', 'NRT', '2024-03-15', '2024-03-25'),
('European Tour', 'JFK', 'LHR', '2024-06-01', '2024-06-15');
```

Create the database connection in `internal/database/database.go`:

```go
package database

import (
    "database/sql"
    "os"

    _ "github.com/mattn/go-sqlite3"
)

type Trip struct {
    ID          int    `json:"id"`
    Title       string `json:"title"`
    Origin      string `json:"origin"`
    Destination string `json:"destination"`
    StartDate   string `json:"start_date"`
    EndDate     string `json:"end_date"`
    CreatedAt   string `json:"created_at"`
}

type DB struct {
    conn *sql.DB
}

func NewDB() (*DB, error) {
    conn, err := sql.Open("sqlite3", "./trips.db")
    if err != nil {
        return nil, err
    }

    db := &DB{conn: conn}
    if err := db.createTables(); err != nil {
        return nil, err
    }

    return db, nil
}

func (db *DB) createTables() error {
    schema, err := os.ReadFile("internal/database/schema.sql")
    if err != nil {
        return err
    }

    _, err = db.conn.Exec(string(schema))
    return err
}

func (db *DB) GetAllTrips() ([]Trip, error) {
    rows, err := db.conn.Query("SELECT id, title, origin, destination, start_date, end_date, created_at FROM trips ORDER BY start_date DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var trips []Trip
    for rows.Next() {
        var trip Trip
        err := rows.Scan(&trip.ID, &trip.Title, &trip.Origin, &trip.Destination, &trip.StartDate, &trip.EndDate, &trip.CreatedAt)
        if err != nil {
            return nil, err
        }
        trips = append(trips, trip)
    }

    return trips, nil
}

func (db *DB) CreateTrip(trip Trip) error {
    _, err := db.conn.Exec(
        "INSERT INTO trips (title, origin, destination, start_date, end_date) VALUES (?, ?, ?, ?, ?)",
        trip.Title, trip.Origin, trip.Destination, trip.StartDate, trip.EndDate,
    )
    return err
}

func (db *DB) UpdateTrip(trip Trip) error {
    _, err := db.conn.Exec(
        "UPDATE trips SET title = ?, origin = ?, destination = ?, start_date = ?, end_date = ? WHERE id = ?",
        trip.Title, trip.Origin, trip.Destination, trip.StartDate, trip.EndDate, trip.ID,
    )
    return err
}

func (db *DB) DeleteTrip(id int) error {
    _, err := db.conn.Exec("DELETE FROM trips WHERE id = ?", id)
    return err
}

func (db *DB) GetTripByID(id int) (Trip, error) {
    var trip Trip
    err := db.conn.QueryRow(
        "SELECT id, title, origin, destination, start_date, end_date, created_at FROM trips WHERE id = ?", id,
    ).Scan(&trip.ID, &trip.Title, &trip.Origin, &trip.Destination, &trip.StartDate, &trip.EndDate, &trip.CreatedAt)

    return trip, err
}
```

## Step 3: Templ Templates

First, install the Templ CLI:

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

Create the base layout in `templates/layout.templ`:

```templ
package templates

templ Layout(contents templ.Component, title string) {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>{ title } - Trip Tracker</title>
        <script src="https://unpkg.com/htmx.org@1.9.10"></script>
        <script src="https://cdn.tailwindcss.com"></script>
    </head>
    <body class="bg-gray-50 min-h-screen">
        <nav class="bg-blue-600 text-white p-4">
            <div class="container mx-auto">
                <h1 class="text-2xl font-bold">Trip Tracker</h1>
            </div>
        </nav>

        <main class="container mx-auto p-4">
            @contents
        </main>
    </body>
    </html>
}
```

The @contents will serve all the other template compnents, allowing you to have a nice base for your entire website.

Create the trips template in `templates/trips.templ`:

```templ
package templates

import "trip-tracker/internal/database"

templ TripsPage(trips []database.Trip) {
    <div class="max-w-4xl mx-auto">
        <div class="flex justify-between items-center mb-6">
            <h2 class="text-3xl font-bold text-gray-800">My Trips</h2>
            <button
                hx-get="/trips/new"
                hx-target="#modal"
                class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg">
                Add New Trip
            </button>
        </div>

        <div id="trips-list" class="space-y-4">
            @TripsList(trips)
        </div>
    </div>

    <!-- Modal for forms -->
    <div id="modal"></div>
}

templ TripsList(trips []database.Trip) {
    for _, trip := range trips {
        @TripCard(trip)
    }
}

templ TripCard(trip database.Trip) {
    <div class="bg-white rounded-lg shadow-md p-6 border-l-4 border-blue-500">
        <div class="flex justify-between items-start">
            <div>
                <h3 class="text-xl font-semibold text-gray-800 mb-2">{ trip.Title }</h3>
                <div class="text-gray-600 space-y-1">
                    <p><span class="font-medium">Route:</span> { trip.Origin } → { trip.Destination }</p>
                    <p><span class="font-medium">Dates:</span> { trip.StartDate } to { trip.EndDate }</p>
                </div>
            </div>
            <div class="flex space-x-2">
                <button
                    hx-get={ "/trips/" + templ.EscapeString(string(rune(trip.ID))) + "/edit" }
                    hx-target="#modal"
                    class="text-blue-600 hover:text-blue-800">
                    Edit
                </button>
                <button
                    hx-delete={ "/trips/" + templ.EscapeString(string(rune(trip.ID))) }
                    hx-target="closest div"
                    hx-swap="outerHTML"
                    hx-confirm="Are you sure you want to delete this trip?"
                    class="text-red-600 hover:text-red-800">
                    Delete
                </button>
            </div>
        </div>
    </div>
}

templ TripForm(trip database.Trip, isEdit bool) {
    <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4">
        <div class="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 class="text-xl font-semibold mb-4">
                if isEdit {
                    Edit Trip
                } else {
                    Add New Trip
                }
            </h3>

            <form
                if isEdit {
                    hx-put={ "/trips/" + templ.EscapeString(string(rune(trip.ID))) }
                } else {
                    hx-post="/trips"
                }
                hx-target="#trips-list"
                hx-swap="innerHTML">

                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Title</label>
                        <input
                            type="text"
                            name="title"
                            value={ trip.Title }
                            required
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Origin</label>
                        <input
                            type="text"
                            name="origin"
                            value={ trip.Origin }
                            required
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Destination</label>
                        <input
                            type="text"
                            name="destination"
                            value={ trip.Destination }
                            required
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
                        <input
                            type="date"
                            name="start_date"
                            value={ trip.StartDate }
                            required
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">End Date</label>
                        <input
                            type="date"
                            name="end_date"
                            value={ trip.EndDate }
                            required
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    </div>
                </div>

                <div class="flex justify-end space-x-3 mt-6">
                    <button
                        type="button"
                        onclick="document.getElementById('modal').innerHTML = ''"
                        class="px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50">
                        Cancel
                    </button>
                    <button
                        type="submit"
                        class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
                        if isEdit {
                            Update Trip
                        } else {
                            Create Trip
                        }
                    </button>
                </div>
            </form>
        </div>
    </div>
}
```

Here, we split trips into multiple components, each with their own role:

- TripsPage: Main trip page, hosts the actual component and base page when going to /trips
- TripsList: Lists component, holds the html/css needed as the list wrapper for all the TripCard components
- TripCard: HTML component to hold the actual trip information
- TripForm: Form component to submit trips to the website

Generate the template files. These files will be generate to `.go` files. Do not try to edit them yourself!:

```bash
templ generate
```

## Step 4: HTTP Handlers

Create the main handlers in `internal/handlers/trips.go`:

```go
package handlers

import (
    "net/http"
    "strconv"

    "trip-tracker/internal/database"
    "trip-tracker/templates"
)

type TripHandler struct {
    db *database.DB
}

func NewTripHandler(db *database.DB) *TripHandler {
    return &TripHandler{db: db}
}

func (h *TripHandler) HomePage(w http.ResponseWriter, r *http.Request) {
    trips, err := h.db.GetAllTrips()
    if err != nil {
        http.Error(w, "Failed to fetch trips", http.StatusInternalServerError)
        return
    }

    component := templates.TripsPage(trips)
    templates.Layout(component, "Home").Render(r.Context(), w) // The TripsPage component is rendered under the Layout component
}

func (h *TripHandler) GetTrips(w http.ResponseWriter, r *http.Request) {
    trips, err := h.db.GetAllTrips()
    if err != nil {
        http.Error(w, "Failed to fetch trips", http.StatusInternalServerError)
        return
    }

    templates.TripsList(trips).Render(r.Context(), w)
}

func (h *TripHandler) NewTripForm(w http.ResponseWriter, r *http.Request) {
    trip := database.Trip{}
    templates.TripForm(trip, false).Render(r.Context(), w)
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

    trip := database.Trip{
        Title:       r.FormValue("title"),
        Origin:      r.FormValue("origin"),
        Destination: r.FormValue("destination"),
        StartDate:   r.FormValue("start_date"),
        EndDate:     r.FormValue("end_date"),
    }

    if err := h.db.CreateTrip(trip); err != nil {
        http.Error(w, "Failed to create trip", http.StatusInternalServerError)
        return
    }

    // Clear the modal and refresh the trips list
    w.Header().Set("HX-Trigger", "closeModal")
    h.GetTrips(w, r)
}

func (h *TripHandler) EditTripForm(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/trips/"):]
    idStr = idStr[:len(idStr)-len("/edit")]

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid trip ID", http.StatusBadRequest)
        return
    }

    trip, err := h.db.GetTripByID(id)
    if err != nil {
        http.Error(w, "Trip not found", http.StatusNotFound)
        return
    }

    templates.TripForm(trip, true).Render(r.Context(), w)
}

func (h *TripHandler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
    idStr := r.PathValue("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid trip ID", http.StatusBadRequest)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

    trip := database.Trip{
        ID:          id,
        Title:       r.FormValue("title"),
        Origin:      r.FormValue("origin"),
        Destination: r.FormValue("destination"),
        StartDate:   r.FormValue("start_date"),
        EndDate:     r.FormValue("end_date"),
    }

    if err := h.db.UpdateTrip(trip); err != nil {
        http.Error(w, "Failed to update trip", http.StatusInternalServerError)
        return
    }

    w.Header().Set("HX-Trigger", "closeModal")
    h.GetTrips(w, r)
}

func (h *TripHandler) DeleteTrip(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/trips/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid trip ID", http.StatusBadRequest)
        return
    }

    if err := h.db.DeleteTrip(id); err != nil {
        http.Error(w, "Failed to delete trip", http.StatusInternalServerError)
        return
    }

    // Return empty response to remove the element
    w.WriteHeader(http.StatusOK)
}
```

## Step 5: Main Application

Create `main.go`:

```go
package main

import (
    "log"
    "net/http"
    "os"
    "strings"

    "trip-tracker/internal/database"
    "trip-tracker/internal/handlers"
)

func main() {
    // Initialize database
    db, err := database.NewDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }

    // Initialize handlers
    tripHandler := handlers.NewTripHandler(db)

    // Routes
    http.HandleFunc("/", tripHandler.HomePage)
    http.HandleFunc("/trips/new", tripHandler.NewTripForm)

    // Trip collection endpoints
    http.HandleFunc("GET /trips", tripHandler.GetTrips)
    http.HandleFunc("POST /trips", tripHandler.CreateTrip)

    // Individual trip endpoints
    http.HandleFunc("PUT /trips/{id}", tripHandler.UpdateTrip)
    http.HandleFunc("DELETE /trips/{id}", tripHandler.DeleteTrip)

    // Trip edit form endpoint
    http.HandleFunc("GET /trips/{id}/edit", tripHandler.EditTripForm)

    // Get port from environment or default to 8080
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Step 6: Running the Application

Generate templates and run the server:

```bash
# Generate Templ templates
templ generate

# Run the application
go run main.go
```

Visit `http://localhost:8080` to see your trip tracker in action!

## How It Works

### HTMX Magic

- **hx-get**: Fetches content and replaces target elements
- **hx-post/hx-put/hx-delete**: Sends form data using different HTTP methods
- **hx-target**: Specifies where to put the response
- **hx-swap**: Controls how content is replaced

### Templ Benefits

- Type-safe templates with Go syntax
- Compile-time validation
- No runtime template parsing overhead
- IntelliSense support in modern editors

### Database Layer

- Simple CRUD operations with SQLite
- Prepared statements for security
- Clean separation of concerns

## What's Next?

This basic setup gives you a solid foundation. In future posts, we could explore:

- **Middleware**: Authentication, logging, and CORS
- **Advanced HTMX**: Real-time updates, infinite scroll
- **Database Migrations**: Versioned schema changes
- **Testing**: Unit and integration tests
- **Deployment**: Docker containers and cloud hosting

The beauty of this stack is its simplicity - you get modern web app features without the complexity of heavy frameworks. Go's standard library, combined with HTMX and Templ, creates a powerful and maintainable web application.
