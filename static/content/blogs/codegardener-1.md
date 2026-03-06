# Learning Rust by Breaking It (A Lot) - Codegardener Part 1

Tags: DevLog, Rust

## Why Rust? Why Now?

If you read my last post, you know I've been thinking a lot about the relationship between AI tools and actually getting better as a developer. Part of my push this year is to deliberately pick up harder things, not just ship features, but actually grow. Rust has been on my list for a while. I kept bouncing off it every time I tried. The tutorials made sense in the moment and then evaporated immediately.

So I did what I usually do when something isn't sticking from reading: I picked a project.

The project is called **Codegardener**. The idea came from a real frustration. Every time I join a new codebase, I spend the first week being scared of the wrong files. I'll avoid touching something because it's big and intimidating, then get burned by a tiny 30-line file that silently owns half the application's behavior. What I actually wanted was a tool that could look at a repository's git history and tell me: *these are the files that have seen the most action, have the most people touching them, and whose commit messages tell you nothing about why anything changed.*

Git history is an honest record. Nobody lies to it. High commit counts, many authors, vague messages, these are signals. I wanted something to surface them.

So: Rust, CLI tool, reads git history, outputs a health report. Let's go.

---

## First Session: Making Something Exist

My first goal was embarrassingly small: get a `healthcheck .` command that calls `git rev-list --count HEAD` and prints the commit count. That's it. No analysis. Just prove the thing runs.

For CLI argument parsing, Rust has a library called `clap` which does a lot of heavy lifting through derive macros. I was honestly surprised at how little code this took:

```rust
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(version, about = "Analyzes repository health and complexity patterns")]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    Healthcheck {
        #[arg(default_value = ".")]
        path: PathBuf,
    },
}
```

That's it. `codegardener healthcheck /path/to/repo` works. Version flags, help text, defaults, all included. I expected this to be harder.

Calling git from Rust uses `std::process::Command`, which spawns external processes and captures output:

```rust
let output = Command::new("git")
    .arg("rev-list")
    .arg("--count")
    .arg("HEAD")
    .current_dir(repo_path)
    .output();
```

The `.current_dir()` is important, it's how git knows which repository you're asking about. Without it, git would look at whatever directory the process happens to be running from, which is almost never what you want.

The return type is `Result<Output, std::io::Error>` and this is where Rust immediately becomes different from what I'm used to. Everything fallible returns a `Result`. You have to handle it. No exceptions, no implicit failures, you decide what happens when something goes wrong, every time.

The `?` operator became my most-used tool early on:

```rust
let output = Command::new("git")
    // ...
    .output()
    .map_err(|e| format!("Failed to run git: {}", e))?;
```

`?` means: if this is an error, return it immediately from the function. If it's a success, unwrap it and keep going. It's five lines of match statement compressed into one character. `.map_err()` converts between error types, from `std::io::Error` to `String` in this case, because `?` requires them to match.

First session done. The tool exists and can count commits. Nothing interesting yet.

---

## Second Session: The Wall Every Rust Beginner Hits

I needed a function that returns all the files tracked by git. I wrote this without thinking:

```rust
pub fn get_tracked_files(repo_path: &str) -> Result<Vec<&str>, String> {
    let output_string = String::from_utf8(output.stdout)?;
    let files: Vec<&str> = output_string.lines().collect();
    Ok(files)
}
```

The compiler rejected it:

```
error[E0597]: `output_string` does not live long enough
  |
  |     let files: Vec<&str> = output_string.lines().collect();
  |                             ^^^^^^^^^^^^ borrowed value does not live long enough
  |  }
  |  - `output_string` dropped here while still borrowed
```

My first reaction was basically "what?" It looked fine to me. But what the compiler is saying is actually important: `.lines()` returns string slices that *borrow* from `output_string`. If I return those slices, the caller holds references to memory that gets freed when the function ends. That's a classic use-after-free bug. Rust simply won't compile it.

The fix is to stop borrowing and start owning:

```rust
pub fn get_tracked_files(repo_path: &str) -> Result<Vec<String>, String> {
    let output_string = String::from_utf8(output.stdout)?;
    let files: Vec<String> = output_string
        .lines()
        .map(|line| line.to_string())  // owned copies
        .collect();
    Ok(files)
}
```

`.to_string()` creates a new `String` for each line that the caller fully owns. When `output_string` gets cleaned up at the end of the function, nothing points to it anymore.

I had read about ownership before this. Though the compiler is a pretty good teacher if you're willing to read what it's telling you instead of just googling the error message.

I also set up a `CommitInfo` struct to hold what git gives us per commit:

```rust
pub struct CommitInfo {
    pub hash: String,
    pub author: String,
    pub timestamp: i64,
    pub message: String,
}
```

For a simple data container like this, public fields are fine. No need for getters. Rust doesn't force encapsulation, it just gives you the tools if you need it.

---

## Third Session: Real Data Is Humbling

Session 3 was the first time I pointed the tool at an actual repository, a Go web app with around 400 commits and 94 tracked files. I was curious what it would say.

Three things broke.

### The UTF-8 crash

```
Error: "stream did not contain valid UTF-8"
```

`git ls-files` returns every tracked file. Including PNG images. Including SQLite databases. When I called `fs::read_to_string()` on a PNG, it crashed, binary files aren't UTF-8 text.

This is where I learned the real difference between `?` and `match`. I had been using `?` for everything because it's convenient. But `?` propagates errors upward and stops the function. When a binary file fails to read, that's not a catastrophic failure, it's *expected*. The right call is to handle it locally and keep going:

```rust
// Before: crashes on first binary file
let lines = get_line_count(path_str)?;

// After: logs and continues
match get_line_count(path_str) {
    Ok(lines) => { /* process it */ }
    Err(_) => { /* skip and move on */ }
}
```

Use `?` when you genuinely can't continue, like if the file list itself fails to load. Use `match` when the failure is something you can recover from. That distinction took me a few concrete examples to internalize.

### The path mismatch

```
Error: "No such file or directory (os error 2)"
```

`git ls-files` returns paths relative to the repository root, `src/main.rs`, `internal/handlers/dashboard.go`, that kind of thing. But `fs::read_to_string()` looks for files relative to wherever the *process* is running. I was running Codegardener from my development directory. The files were in a completely different directory.

The file existed at `/home/user/trip-tracker/src/main.rs`. I was asking for `src/main.rs`. Totally different thing.

```rust
use std::path::Path;

let full_path = Path::new(repo_path).join(&file);
let path_str = full_path.to_str()
    .ok_or_else(|| format!("Invalid Unicode in path: {}", file))?;
```

`Path::new().join()` combines paths correctly and handles edge cases that would bite you with naive string concatenation. The `.ok_or_else()` at the end converts the `Option<&str>` from `.to_str()` into a `Result` so I can use `?` again.

Different APIs handle working directories completely differently. I hadn't thought about it at all. Real data caught it.

### Generated files taking over the output

Once the crashes were fixed, the top files in my output were these:

```
static/css/output.css: 23 commits, 1809 lines
templates/trips_templ.go: 20 commits, 1616 lines
```

These are build artifacts. Minified CSS and auto-generated Go templates that get committed alongside the source that drives them. They show high commit counts because every time the real source changed, these regenerated. They're not maintenance hotspots, they're noise.

The fix was a filter that runs before any expensive git calls:

```rust
fn should_skip_file(file_path: &str) -> bool {
    file_path.ends_with("_templ.go")
        || file_path.ends_with(".min.js")
        || file_path.ends_with(".png")
        || file_path.ends_with(".jpg")
        || file_path.ends_with(".webp")
        || file_path.contains("/node_modules/")
}

for file in files {
    if should_skip_file(&file) {
        continue;
    }
    // ... now do the expensive stuff
}
```

Nothing fancy. But this was a good reminder that raw data and meaningful data are not the same thing. A high number means something different depending on context. Generated files made me confront that early.

---

## Where Things Stand

By the end of Session 3 the tool can:
- Walk all tracked files in a repository
- Fetch full commit history per file
- Count unique authors using a `HashSet` (deduplication for free)
- Count lines of source
- Skip binary and generated files without crashing

Not impressive. But every single lesson came from a real failure on a real codebase, not a contrived example. The ownership lesson came from the compiler rejecting a dangling reference. The `?` vs `match` lesson came from a crash that propagated when it should have degraded. The path lesson came from a confusing error message at 11pm.

I keep coming back to something I've noticed: the skills I retain are always attached to a failure I had to dig out of. Reading about ownership is fine. Getting rejected by the compiler and having to understand why is what actually sticks.

---

## Some Honest Thoughts on the Process

This project is deliberately a no-AI-assistance zone for the implementation. I mentioned in the last post that I noticed my programming instincts getting dull when I leaned on AI too much. Writing Rust, a language I don't know, with no safety net has been a good test of that.

The borrow checker errors have been the biggest adjustment. In Go or Python, you just don't have to think about this stuff. In Rust it's constant. At first it felt like the compiler was being pedantic. Now it's starting to feel like it's catching things I genuinely would have missed. The fact that I couldn't compile the wrong version means I couldn't ship it either.

Whether that tradeoff is worth it for every project is a different conversation. For learning? Definitely yes. It forces you to think about memory and ownership in a way that makes you better at understanding code in any language.

---

## What's Next

The next session is where it gets actually interesting. I had all this data, commit counts, author lists, line counts, but no way to make sense of it together. That means designing a risk scoring formula, sorting, and the first moment the output looked genuinely useful.

