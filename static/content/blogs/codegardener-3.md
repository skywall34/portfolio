# The Bug Where Everything Was Suspiciously Perfect - Codegardener Part 3

Tags: DevLog, Rust

## Previously

Quick recap: I'm building Codegardener, a CLI tool that analyzes git history to surface maintenance hotspots. By the end of Part 2, it had a working risk formula and sorted output. Files with high churn, many authors, and small size bubble to the top. It ran on a real repository and the results made sense.

Now the MVP was done, which usually means it's time to make it actually usable. Sessions 6 through 9 were all polish, architecture, and a debugging session I'm still a little embarrassed about.

---

## Session 6: Getting the Output Out of the Terminal

The tool printed to the terminal and that was it. If you wanted to share results with someone or attach them to a pull request, you were copying text from a shell window. Not ideal.

The goals for this session: limit the output to the top 10 files instead of dumping everything, add some summary stats at the top, and write the report to a markdown file.

### Top N with `.take()`

Limiting the output is one line:

```rust
for analysis in file_analyses.iter().take(10) {
    // print it
}
```

`.take(N)` yields at most N items from the iterator and stops. If the collection has fewer than N items it just yields all of them without panicking. I kept waiting for a footgun here and there wasn't one. Nice.

### Aggregating stats

For the summary header I needed total commits across all files and the highest risk score. Total commits:

```rust
let total_commits: usize = file_analyses.iter().map(|a| a.commits).sum();
```

The type annotation on `total_commits` is required here. The compiler can't infer what type `.sum()` should produce because the same method works for many numeric types. You have to tell it.

Finding the max risk score was more interesting. My first attempt:

```rust
let max_risk = file_analyses.iter().map(|a| a.risk_score).max();
```

Compiler error. `f64` doesn't implement the trait required for `.max()`. The reason is specific to floating point: `f64` has a value called `NaN` (Not a Number), which is what you get from operations like `0.0 / 0.0`. `NaN` breaks total ordering because `NaN > 5.0` is neither true nor false, it's just undefined. Rust's `.max()` requires total ordering, meaning every pair of values must have a definitive answer. Since `f64` can't guarantee that, `.max()` isn't available on it.

The workaround is `.fold()`, which is a manual accumulation:

```rust
let max_risk = file_analyses.iter()
    .map(|a| a.risk_score)
    .fold(0.0_f64, f64::max);
```

`.fold()` starts with an initial value (`0.0`) and applies a function to each element, carrying the result forward. `f64::max` is the function, it takes two floats and returns the larger one. The end result is the largest value in the iterator. It's a bit more verbose than `.max()` but not complicated once you've seen it once.

### The lifetime error, again

To print the repository name at the top of the report, I tried to extract the directory name from the path:

```rust
let repo_name = Path::new(s)
    .canonicalize()
    .ok()
    .and_then(|p| p.file_name())   // returns a reference into p
    .and_then(|n| n.to_str())
    .unwrap_or(s);
```

Compiler error:

```
cannot return value referencing function parameter `p`
```

The issue: `p.file_name()` returns a reference that borrows from `p`. But `p` only exists inside that closure. When the closure ends, `p` is dropped, and the reference goes with it. Rust catches this because it's a use-after-free waiting to happen.

The fix is to store `p` somewhere that outlives the chain:

```rust
let canonical_path = std::fs::canonicalize(s)
    .unwrap_or_else(|_| PathBuf::from(s));

let repo_name = canonical_path
    .file_name()
    .and_then(|n| n.to_str())
    .unwrap_or(s);
```

Now `canonical_path` lives long enough for the reference to be valid. This is the same ownership lesson from Part 1, just showing up in a different shape. Data you're borrowing from has to outlive the borrow. The compiler keeps enforcing this and I keep finding new situations where I've forgotten it.

### Writing to a file

`writeln!` works exactly like `println!` but takes a file handle as its first argument:

```rust
use std::fs::File;
use std::io::Write;

let mut file = File::create(&report_path)?;
writeln!(file, "# Codegardener Health Report")?;
writeln!(file, "**Repository:** {}", repo_name)?;
```

Each `writeln!` returns a `Result` because writing to disk can fail, disk full, permissions, etc. The `?` propagates those errors up. If any write fails, the whole function returns the error. The design decision I made here was that a failed file write should be a warning, not a crash. Terminal output is the primary thing, the file is supplementary. So the call site uses `if let Err(e)` instead of `?`:

```rust
if let Err(e) = write_markdown_report(...) {
    eprintln!("Warning: Could not write report: {}", e);
}
```

The tool keeps running. The user still gets their terminal output. The file just isn't there.

---

## Session 7: The Bug Where Everything Was Suspiciously Perfect

One of the things I wanted to surface was commit message quality. The idea, which I called a "Narrative Heuristic" in my design doc, is that files with high churn and vague commit messages are a particular kind of maintenance hazard. Not just "this changes a lot" but "this changes a lot and nobody explained why."

Vague messages are things like "fix", "update", "wip", "refactor" with no further context. I wrote a function to detect them:

```rust
fn is_vague_message(msg: &str) -> bool {
    let vague_patterns = ["^fix", "^update", "^wip", "^refactor$"];
    vague_patterns.iter().any(|p| msg.starts_with(p))
}
```

And a function to calculate the percentage per file:

```rust
fn calculate_vague_messages(commits: &Vec<CommitInfo>) -> f64 {
    if commits.is_empty() { return 0.0; }
    let vague_count = commits.iter()
        .filter(|c| is_vague_message(&c.message))
        .count();
    vague_count as f64 / commits.len() as f64 * 100.0
}
```

I ran it. Every single file showed `0.00% vague`. Every one.

My first thought was that maybe this codebase just had unusually disciplined commit messages. I pulled up the git log. It did not. There were plenty of "fix", "update", and "wip" commits in there.

Something was wrong.

I added a debug print inside `is_vague_message`:

```rust
fn is_vague_message(msg: &str) -> bool {
    let vague_patterns = ["^fix", "^update", "^wip", "^refactor$"];
    let result = vague_patterns.iter().any(|p| msg.starts_with(p));
    if result {
        println!("DEBUG matched: '{}'", msg);
    }
    result
}
```

Ran it again. No debug output appeared at all. Nothing was matching, ever.

I stared at this for a while. Then I looked at the patterns again:

```rust
["^fix", "^update", "^wip", "^refactor$"]
```

`^` and `$` are regex metacharacters. `^` means "start of string" in a regex. But `.starts_with()` is not regex. It's a literal string check. It was looking for commit messages that literally start with the character `^`. No commit message starts with a caret. Nothing would ever match.

The fix:

```rust
let vague_patterns = ["fix", "update", "wip", "refactor"];
```

Just remove the metacharacters. `.starts_with("fix")` checks whether the string literally starts with "fix", which is exactly what I wanted.

After the fix:

```
internal/database/database.go: risk=20.6 (3 commits 3.00 per month, 66.67% vague, 1 authors, 17 lines)
```

Two out of three commits to that file had vague messages. The tool was working, I had just been searching for `^fix` as a literal string.

The lesson here isn't a Rust lesson specifically, it's a debugging lesson. When every value is the same suspiciously round number, something is broken. Zero is a valid answer sometimes, but `0.00%` across 94 files with a visibly messy git history is a sign to go look at what's actually happening. Adding a debug print to see if the function ever fires at all took 30 seconds and immediately showed the problem.

---

## Session 8: The Output Needed Context

Running the tool on a repository gives you a ranked list of files, but without knowing anything about the repository, the numbers are hard to interpret. Is a churn rate of 3.5 commits/month high or low? It depends entirely on how active the project is overall.

I added a repository-level summary section:

```
============= Codegardener Health Report ===================

Repository: trip-tracker
Age:        11.8 months
Commits:    406
Files:      94
Highest risk: 77.9

Churn Distribution:
  Average:    1.38 commits/month
  Top 10%:    2.69+ commits/month (9 files)
  Bottom 50%: <1.00 commits/month (47 files)

Hotspot Concentration:
  Top 5 files  = 4% of all commits
  Top 10 files = 8% of all commits
```

Now the individual file scores have context. If the average churn rate is 1.38 and a file has 4.41, you know it's an outlier relative to this specific codebase.

### Moving interpretation into its own module

This required a second pass over all the file data after collection, calculating distribution statistics. Putting this logic in `main.rs` was getting crowded, and more importantly, it's conceptually different from the per-file collection work. Collecting git facts is one thing. Deriving what those facts mean across the whole repository is another.

I moved it to `src/interpretation.rs`:

```rust
pub struct RepoStats {
    pub repo_age: f64,
    pub total_commits: usize,
    pub total_files: usize,
    pub avg_churn_rate: f64,
    pub top_ten_percent_churn_rate: f64,
    pub top_ten_percent_file_count: usize,
    pub bottom_fifty_percent_churn_rate: f64,
    pub bottom_fifty_percent_file_count: usize,
    pub top_five_files_percentage: f64,
    pub top_ten_files_percentage: f64,
}

pub fn calculate_repo_stats(
    file_analyses: &[FileAnalysis],
    first_commit_timestamp: i64,
) -> RepoStats {
    // ...
}
```

One thing `interpretation.rs` needs is the `FileAnalysis` struct, which is defined in `main.rs`. In Rust, a child module can access items from its parent using `super::`:

```rust
// in interpretation.rs
use super::FileAnalysis;
```

`super` refers to the parent module. It feels a bit unusual at first but makes sense once you think about the module tree. The alternative would be moving `FileAnalysis` into its own module and importing it everywhere, which felt like unnecessary ceremony for a struct that's genuinely at the center of the whole pipeline.

The pipeline is now: Repository, Git Analysis, Metrics, Interpretation, Report. Each stage has a clear job and doesn't reach backward into the previous one. Interpretation doesn't call git. The report stage doesn't do math. When something goes wrong it's much clearer which stage is responsible.

---

## Session 9: A Small Flag with an Unexpected Quirk

The last thing I added was a `--write-file` flag so users can suppress the markdown output if they just want terminal results:

```bash
codegardener healthcheck . --write-file false
```

The clap definition looked obvious:

```rust
#[arg(long, default_value_t = true)]
write_file: bool,
```

Running it:

```
error: unexpected argument 'false' found
```

Turns out clap treats `bool` arguments as flags by default, meaning `--write-file` with no value sets it to true, and its absence means false. There's no way to pass `false` explicitly. The word "false" after the flag is just seen as an unexpected positional argument.

To make `--write-file false` and `--write-file true` both work as explicit values, you need one extra attribute:

```rust
#[arg(long, default_value_t = true, num_args = 1)]
write_file: bool,
```

`num_args = 1` tells clap to consume exactly one following value and parse it as the field's type. Now it works as expected.

---

## Where Codegardener Is Now

After nine sessions, running `codegardener healthcheck .` gives you:

- A repository overview with age, total commits, churn distribution, and hotspot concentration
- A ranked list of the 10 highest-risk files with the numbers that explain each ranking, churn rate, vague commit percentage, author count, and line count
- A markdown report written to disk, or skipped with `--write-file false`

The whole thing is a single binary. No runtime, no configuration required, no internet connection. Point it at any git repository and it works.

---

## Wrapping Up the Series

Looking back across all three posts, the things that actually taught me Rust weren't the tutorials or the book chapters. They were:

The compiler rejecting code. Having to understand *why* before I could fix it.

The NaN explanation for why `f64` doesn't have `.max()`. I would have cargo-culted my way around that in any other language. In Rust the type system forced me to confront it.

The regex bug where every metric was 0.00%. Not a Rust lesson, just a debugging lesson, but I found it because I was reading real output instead of assuming the code worked.

The `num_args = 1` clap thing. Tiny, but it made me read the library docs carefully for the first time instead of just copying an example.

None of these are things you get from building toy examples. They all came from building something real and running it against real repositories. That's been the consistent theme across this whole project: the gap between "the code compiles" and "the output makes sense" is where the learning actually happens.

If you want to follow along or try it yourself, the project is on GitHub. It's still rough in places and there's plenty left to build, but it works and it's taught me more Rust in nine sessions than I managed to pick up in a year of occasional tutorials.
