# htmx-portfolio

Hello There!

This project is a personal portfolio website built with Go, HTMX, and Tailwind CSS. I'm using it mostly to host my projects and blogs, so feel free to read through it if this interests you.

## About Me

**Mike Shin**

- **Email:** [doshinkorean@gmail.com](mailto:doshinkorean@gmail.com)
- **LinkedIn:** [https://www.linkedin.com/in/shindohyun/](https://www.linkedin.com/in/shindohyun/)
- **GitHub:** [github.com/skywall34](https://github.com/skywall34)

---

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

Make sure you have the following tools installed:

- [Go](https://golang.org/dl/) (version 1.22 or later)
- [Air](https://github.com/cosmtrek/air) for live reloading
- [Tailwind CSS CLI](https://tailwindcss.com/docs/installation)
- [templ](https://github.com/a-h/templ) for HTML templating

### Installation & Running

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/yourusername/htmx-portfolio.git
    cd htmx-portfolio
    ```

2.  **Install Go dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Generate HTML components:**
    The `templ` files in the `/templates` directory are used to generate Go code for the UI. Run the following command to generate them:

    ```bash
    templ generate
    ```

4.  **Build Tailwind CSS:**
    To compile the utility classes into a static CSS file, run the Tailwind CLI:

    ```bash
    ./tailwindcss -i ./static/css/input.css -o ./static/css/output.css
    ```

5.  **Run the development server:**
    This project uses `air` for live reloading of the Go application. It automatically recompiles and restarts the server when it detects changes in your Go files.
    ```bash
    air
    ```
    The application will be available at `http://localhost:8080`.

### Development Workflow

For an efficient development workflow, you can run the `templ` and `tailwindcss` commands with a `--watch` flag in separate terminal sessions to automatically regenerate files on change:

- **Watch for template changes:**

  ```bash
  templ generate --watch
  ```

- **Watch for CSS changes:**

  ```bash
  ./tailwindcss -i ./static/css/input.css -o ./static/css/output.css --watch
  ```

- **Run the Go server with live reload:**
  ```bash
  air
  ```

---

## Project Structure

```
.
├── internal/         # Internal application logic (handlers, models)
├── static/           # Static assets (CSS, images, JS)
├── templates/        # templ files for HTML components
├── .air.toml         # Air configuration for live reloading
├── go.mod            # Go module definition
├── main.go           # Main application entry point
├── tailwind.config.js # Tailwind CSS configuration
└── README.md         # This file
```
