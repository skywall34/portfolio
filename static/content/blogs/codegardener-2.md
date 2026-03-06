# When Your Formula Ranks a PNG as the Biggest Risk - Codegardener Part 2

Tags: DevLog, Rust

## Previously

If you missed Part 1, the short version: I'm building a CLI tool in Rust called Codegardener that analyzes git history and surfaces maintenance hotspots. The first three sessions were mostly about getting the plumbing working. By the end I could collect commit data per file, count unique authors, read line counts, and skip binary or generated files without crashing.

Now I had a pile of data and no way to do anything meaningful with it. That's where sessions 4 and 5 come in.

---

## Session 4: You Can't Sort What You Haven't Seen Yet

The immediate problem was output order. I was printing files as I processed them, which meant the output was basically random, just whatever order `git ls-files` returned. I wanted highest-churn files at the top.

The obvious instinct is to sort as you go. It doesn't work. Imagine files arriving one at a time: File A has 10 commits, File B has 20, File C has 15. When you see File A, is it first or last? You have no idea. You haven't seen B or C yet. You can only sort once you've seen everything.

So the structure became: collect everything first, then sort, then print.

```rust
let mut file_analyses: Vec<FileAnalysis> = Vec::new();

for file in files {
    // ... collect data ...
    file_analyses.push(FileAnalysis {
        path: file,
        commits: commits.len(),
        authors: authors.len(),
        lines,
    });
}

file_analyses.sort_by(|a, b| b.commits.cmp(&a.commits));

for analysis in &file_analyses {
    println!("{}: {} commits", analysis.path, analysis.commits);
}
```

The `FileAnalysis` struct is what makes this clean. Before I had it, I was juggling separate variables for path, commit count, author count, and line count. Naming them in a struct meant I could write `analysis.commits` instead of remembering which index of a tuple held what. It sounds minor until you're debugging at midnight.

### Closures for sorting

The sort line deserves a closer look:

```rust
file_analyses.sort_by(|a, b| b.commits.cmp(&a.commits));
```

The `|a, b|` is a closure, basically an anonymous function. `sort_by` takes a comparison function as an argument, and instead of defining a whole named function, you write it inline. Coming from Go this felt a bit foreign at first, but it clicks quickly.

The ordering is `b.cmp(&a)` instead of `a.cmp(&b)`. That reversal is what gives you descending order. `a.cmp(&b)` sorts smallest to largest. Swap them and you get largest to smallest. I got this backwards the first time and had to re-read the output to figure out why the files with one commit were at the top.

One thing Rust does here that I appreciated: it infers the types of `a` and `b` from context. I didn't have to write `|a: &FileAnalysis, b: &FileAnalysis|`. The compiler knows what's in the vector and figures it out. Less boilerplate than I expected.

### Consuming vs borrowing in for loops

There's a subtle thing in the printing loop that caught me once:

```rust
// This consumes the vector, you can't use file_analyses after this
for analysis in file_analyses {
    println!("{}", analysis.path);
}

// This borrows it, file_analyses still exists
for analysis in &file_analyses {
    println!("{}", analysis.path);
}
```

If you loop without the `&`, Rust moves each item out of the vector as you iterate. The vector is gone when the loop ends. If you need the data again afterwards, use `&`. I ran into this early on trying to print the list and then do something else with it and got a compiler error about using a moved value.

---

## Session 5: Okay But What Does the Data Actually Mean?

Having files sorted by raw commit count was a start, but it wasn't the insight I wanted. A file with 50 commits over five years is very different from a file with 50 commits over three months. And a 10-line file with high churn is a different kind of problem than a 1000-line file with the same churn rate, because every change to the small file touches a much higher percentage of it.

I needed a score that captured all of this together.

### Designing the formula

I spent some time thinking about what factors actually matter for maintenance risk:

**Churn rate** (commits per month, not just total commits). A file's lifetime matters. If a file has been around for three years and only has 10 commits, that's a different story than 10 commits in one month. I already had timestamps in `CommitInfo` so this was calculable.

**Author count**. More contributors means more coordination overhead. When only one person has ever touched a file, they understand it deeply. When eight people have all made changes across two years, nobody fully owns it anymore.

**File size** as a complexity proxy. I don't have AST parsing or anything sophisticated. But line count gives a rough sense of how dense a file is. More importantly, I wanted to weight small files higher, because a change to a 6-line file is proportionally much more impactful than a change to a 600-line file.

Combining all three:

```rust
fn calculate_risk_score(churn_rate: f64, authors: usize, lines: usize) -> f64 {
    let complexity_factor = 1.0 + (100.0 / lines.max(1) as f64);
    churn_rate * authors as f64 * complexity_factor
}
```

The complexity factor deserves explanation. For a 6-line file: `1.0 + (100.0 / 6) = 17.7`. For a 100-line file: `1.0 + (100.0 / 100) = 2.0`. For a 1000-line file: `1.0 + (100.0 / 1000) = 1.1`. Small files get amplified. Large files get closer to just `churn * authors`.

### Rust won't let you do math with mixed types

Coming from Go or Python, this tripped me up:

```rust
let risk = churn_rate * authors;  // compile error
```

`churn_rate` is `f64`. `authors` is `usize`. Rust won't multiply them. You have to cast explicitly:

```rust
let risk = churn_rate * authors as f64;
```

In Python this just works and you don't think about it. In Rust, every type conversion is your decision and you write it out. It's more verbose but it also means you're never surprised by silent precision loss or integer overflow. I've been bitten by that in Python before, so honestly the explicitness is growing on me.

### Preventing division by zero with `.max()`

What happens when a file has zero lines? Division by zero. The formula blows up.

The fix is one method:

```rust
lines.max(1)
```

`.max()` returns whichever is larger, so `0.max(1)` gives 1, `50.max(1)` gives 50. The denominator is never zero. It's a small thing but the kind of defensive habit that Rust makes natural because the language constantly pushes you to think about edge cases.

---

## The Image File Incident

I ran the tool on a repository and got this:

```
static/images/flight_path.webp: risk=101.0 (0 lines)
static/images/hero.png: risk=101.0 (0 lines)
```

Image files. At the top of my risk chart. More dangerous than any actual source file.

Here's what happened: image files have zero lines. Zero lines means the complexity factor is `1.0 + (100.0 / 1) = 101.0`. Any commits touching that file got multiplied by 101. One commit by one author was enough to rank it above everything.

The frustrating thing is I had already written a file filter. I just hadn't caught every image format:

```rust
fn should_skip_file(file_path: &str) -> bool {
    file_path.ends_with(".png")
        || file_path.ends_with(".jpg")
        // ... but I forgot .webp, .svg, .gif, .ico
}
```

The fix was straightforward, expand the list:

```rust
fn should_skip_file(file_path: &str) -> bool {
    file_path.ends_with("_templ.go")
        || file_path.ends_with(".min.js")
        || file_path.ends_with(".png")
        || file_path.ends_with(".jpg")
        || file_path.ends_with(".jpeg")
        || file_path.ends_with(".webp")
        || file_path.ends_with(".svg")
        || file_path.ends_with(".gif")
        || file_path.ends_with(".ico")
        || file_path.contains("/node_modules/")
}
```

This isn't the best solution by far. The next iteration if I cared enough would be some sort of configuration or way to know which files are ignorable. Probably something for future sessions. 

---

## The MVP Moment

After fixing the image filter, I ran the tool on the same Go codebase from session 3:

```
============== Files by Risk Score (highest first) ==================

Risk score = churn rate x authors x complexity factor

internal/models/countryaggregation.go: risk=77.9 (5 commits 4.41 per month, 1 authors, 6 lines)
internal/models/timespaceaggregation.go: risk=61.1 (4 commits 4.00 per month, 1 authors, 7 lines)
internal/database/database.go: risk=20.6 (3 commits 3.00 per month, 1 authors, 17 lines)
main.go: risk=4.9 (27 commits 3.53 per month, 1 authors, 372 lines)
```

This actually makes sense. The two model files at the top are tiny, 6 and 7 lines respectively, and they've been changing roughly four times per month. Every change is significant because it's touching most of the file. `main.go` has way more commits overall but it's 372 lines and its score reflects that the per-change impact is much lower.

The formula isn't sophisticated. There's no machine learning, no static analysis, no AST parsing. It's three factors multiplied together with a simple normalization. But when the output matches your intuition about a codebase you already know, that's a good sign that the signals are real.

---

## Where Things Stand After Five Sessions

The tool now does something genuinely useful. Point it at a repository, get a ranked list of files that deserve attention, with the numbers that explain why each one ranked where it did.

What's still missing:

- Commit counts without any context on *what* was in those commits. Two files can both have 10 commits but one of them has messages like "fix", "update", "wip" and the other has actual descriptions. That's a different level of risk.
- No repository-wide summary. You get a file list but no sense of how spread out the risk is, or whether a few files are carrying all the churn versus it being evenly distributed.
- The output just dumps to the terminal. There's no way to share it or look at it outside of a shell session.

Those are the next problems. The commit message one turned into a surprisingly interesting debugging session that I'll write about next time.

---

## Some Thoughts on Formula Design

One thing I keep coming back to is that this formula is deliberately transparent. The output shows the score *and* the numbers that produced it. Anyone looking at the results can verify whether the ranking makes sense. There's no black box.

I think this matters for a tool like this. If it just said "risk: high" with no explanation, you'd either trust it completely or ignore it completely. Showing `churn_rate x authors x complexity_factor` with the actual values means you can look at a result and say "yeah, that tracks" or "that's weird, something else is going on here." The tool is meant to start a conversation, not end one.

That philosophy came directly from a document I wrote at the start of the project, kind of a design contract for myself, that said the tool should be observational and not judgmental. No grades, no verdicts. Just: here's what the data shows, here's why this file keeps coming up.

I found that writing the design intent down before touching any code actually shaped how I built things. When I was tempted to add a "grade" system (A through F or something), going back to that document reminded me that was exactly what I said I wasn't building. Recommended if you're a developer who tends to get carried away with scope.
