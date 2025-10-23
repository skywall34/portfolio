# Building SysMon: A High-Performance Terminal System Monitor in Rust

System monitoring is a fundamental task for any developer or system administrator. While tools like `htop` and `btop` are excellent, I wanted to build my own terminal-based system monitor to deeply understand system metrics, practice Rust, and create something tailored to my workflow. The result is **SysMon** - a fast, modular TUI application that monitors CPU, memory, processes, disk usage, and even lets you browse your filesystem.

In this post, I'll walk through why I built SysMon, how I designed its architecture, the implementation process from start to finish, and the critical performance optimization that saved the project from unusability.

## Motivation: Why Build Another System Monitor?

The idea for SysMon came from a simple observation: I was constantly switching between multiple terminal tools to get a complete picture of my system's health. `htop` for processes, `df -h` for disk usage, `ncdu` for directory exploration. What if I could combine these into one cohesive, keyboard-driven interface?

Beyond practicality, I had three learning goals:

1. **Master Terminal UI Development**: Learn the ratatui framework and build a polished, responsive TUI
2. **Understand System Programming**: Get hands-on experience with system APIs for CPU, memory, processes, and filesystems
3. **Practice Rust Idioms**: Apply ownership, borrowing, error handling, and performance optimization in a real-world project

The requirements were clear:
- Real-time monitoring of CPU (per-core), memory, and processes
- Interactive disk usage explorer with filesystem navigation
- Smooth 60 FPS UI with efficient rendering
- Keyboard-driven navigation with search/filter capabilities
- Fast enough to handle systems with hundreds of processes and large directory trees

## Architecture: Designing for Modularity and Performance

Before writing code, I sketched out the architecture with a focus on **separation of concerns** and **performance-first design**.

### Core Principles

**1. Modular Design**

I organized the codebase into three main components:

```
src/
├── main.rs          # Application state, event loop, terminal management
├── ui.rs            # TUI rendering (all ratatui code)
└── metrics/
    ├── mod.rs       # Metrics aggregator
    ├── system.rs    # CPU, memory, processes
    ├── disk.rs      # Disk and mount points
    └── directory.rs # Directory navigation and size calculation
```

Each module has a single responsibility:
- **Metrics modules** collect and process system data
- **UI module** renders data to the terminal
- **Main module** coordinates everything and handles user input

**2. Performance-First Approach**

From the start, I knew performance would be critical. The design includes:

- **Query Caching**: Filter operations cache results based on search query strings
- **Reference Returns**: Functions return slices (`&[T]`) instead of cloning vectors
- **Separate Update Intervals**: System metrics update every 500ms, but UI refreshes at 60 FPS
- **Scroll Windowing**: Only render visible rows, not the entire list

**3. Rust Idioms**

I leveraged Rust's strengths:
- **RAII Pattern**: A `RawModeGuard` struct ensures terminal cleanup even on panic
- **Default Trait**: Idiomatic initialization for metrics structs
- **Saturating Arithmetic**: Prevent overflow in size calculations
- **Result Types**: Graceful error handling for I/O operations

### Data Flow

The application follows a simple event loop pattern:

```
┌─────────────────────────────────────┐
│      60 FPS Event Loop              │
│                                     │
│  1. Check if 500ms elapsed          │
│     └─> Update metrics if needed    │
│                                     │
│  2. Filter data based on search     │
│     └─> Use cached results          │
│                                     │
│  3. Render UI (always)              │
│     └─> Only visible items          │
│                                     │
│  4. Poll for keyboard input         │
│     └─> Update state                │
│                                     │
│  5. Loop or exit                    │
└─────────────────────────────────────┘
```

This decoupling of metrics updates from UI refresh is crucial - system calls are expensive, but we want instant feedback to user input.

## Implementation: Building SysMon Step by Step

### Phase 1: Terminal Foundation and Basic Monitoring

**Setting Up the Terminal**

I started with terminal initialization using `crossterm`:

```rust
enable_raw_mode()?;
execute!(io::stdout(), EnterAlternateScreen)?;
let terminal = Terminal::new(CrosstermBackend::new(io::stdout()))?;
```

The key insight here was implementing a `RawModeGuard` using Rust's Drop trait:

```rust
struct RawModeGuard;

impl Drop for RawModeGuard {
    fn drop(&mut self) {
        let _ = disable_raw_mode();
    }
}
```

This ensures the terminal is restored even if the application panics - critical for maintaining a usable shell.

**System Metrics Collection**

Using the `sysinfo` crate, I built the `SystemMetrics` struct in `src/metrics/system.rs`:

```rust
pub struct SystemMetrics {
    pub cpu_average: f32,
    pub cpu_per_core: Vec<(String, f32)>,
    pub total_memory: u64,
    pub used_memory: u64,
    pub processes: Vec<ProcessInfo>,
    // Caching fields
    filtered_processes: Vec<ProcessInfo>,
    last_process_query: String,
}
```

The `update()` method refreshes all CPU cores and memory stats, then collects process information. The critical optimization here is the filter caching:

```rust
pub fn filter_processes(&mut self, query: &str) -> &[ProcessInfo] {
    if query.is_empty() {
        return &self.processes;
    }

    // Only re-filter if query changed
    if query != self.last_process_query {
        self.filtered_processes = self.processes
            .iter()
            .filter(|p| /* matching logic */)
            .cloned()
            .collect();
        self.last_process_query = query.to_string();
    }

    &self.filtered_processes  // Return reference, no cloning
}
```

This pattern gives us O(1) performance for cache hits during real-time filtering.

**Building the UI**

With `ratatui`, I created a two-panel layout in `src/ui.rs`:

- **Left panel**: CPU gauge (with color-coded thresholds), per-core bars, and memory statistics
- **Right panel**: Process table with columns for PID, name, CPU%, memory, and status

The color-coding provides instant visual feedback:

```rust
fn get_cpu_color(usage: f32) -> Color {
    if usage < 30.0 { Color::Green }
    else if usage < 60.0 { Color::Yellow }
    else if usage < 80.0 { Color::LightRed }
    else { Color::Red }
}
```

### Phase 2: Adding Disk Monitoring

The next major feature was disk usage monitoring. I created `src/metrics/disk.rs` with similar caching patterns:

```rust
pub struct DiskInfo {
    pub name: String,
    pub mount_point: String,
    pub filesystem: String,
    pub total_space: u64,
    pub available_space: u64,
    pub is_removable: bool,
}
```

Key methods include:
- `disk_used_space()`: Calculates used space as `total - available`
- `disk_usage_percent()`: Percentage calculation with proper formatting
- `format_bytes()`: Human-readable size formatting (K, M, G, T)

The disk UI shows a comprehensive table with mount points, filesystem types, sizes, and usage percentages - all sortable by different criteria.

### Phase 3: Directory Browser and Tree Traversal

The most ambitious feature was the interactive directory browser. Users can press Enter on a disk to browse its contents, navigate into subdirectories, and see file/folder sizes.

The `DirectoryNavigator` struct in `src/metrics/directory.rs` handles:

```rust
pub struct DirectoryEntry {
    pub name: String,
    pub path: PathBuf,
    pub size: u64,           // Recursive for directories
    pub is_directory: bool,
}
```

Key navigation methods:
- `enter_directory()`: Scans a directory and calculates sizes
- `exit_to_parent()`: Navigate up one level
- `exit_to_mount_points()`: Return to disk overview

The entries are always sorted with:
1. ".." (parent directory) at the top
2. Directories (with "/" suffix) before files
3. Alphabetical order within each group

### Phase 4: Polish and User Experience

The final phase involved:

- **Search Mode**: Press `/` to enter filter mode with real-time results
- **Multiple Sort Modes**: CPU, memory, PID, name for processes; various criteria for disks
- **Process Management**: Kill processes with `k` key (with confirmation)
- **Responsive Navigation**: Arrow keys with smooth scrolling
- **Page Switching**: Tab to switch between System Monitor and Disk Usage views

I also added comprehensive help text in the footer showing context-sensitive key bindings.

## The Critical Challenge: Directory Browsing Performance

Everything worked beautifully until I tested the directory browser on my home directory (600GB of data). **The application froze for minutes.**

### Root Cause: Recursive Size Calculation

The problem was in my initial implementation of directory size calculation. When entering a directory, I was recursively traversing **every subdirectory** to calculate total sizes:

```rust
// BEFORE: Recursive (SLOW)
let size = if is_directory {
    Self::calculate_directory_size(&entry_path).unwrap_or(0)  // Full recursion!
} else {
    metadata.len()
};
```

For a directory with hundreds of gigabytes spread across thousands of subdirectories, this meant reading metadata for **millions of files** - all synchronously, blocking the UI thread.

**Complexity**: O(n) where n = total files in entire subtree
**Time**: 2-5 minutes for large directories
**User Experience**: Application appears crashed

### Solution: Shallow Size Calculation

The fix was to switch from **recursive (deep)** to **shallow** size calculation. Instead of traversing subdirectories, I only count immediate files:

```rust
// AFTER: Shallow (FAST)
let size = if is_directory {
    Self::calculate_shallow_directory_size(&entry_path).unwrap_or(0)
} else {
    metadata.len()
};

fn calculate_shallow_directory_size(path: &PathBuf) -> Result<u64, std::io::Error> {
    let mut total_size = 0u64;
    for entry in fs::read_dir(path)? {
        let metadata = entry?.metadata()?;
        // Only count files, not subdirectories
        if !metadata.is_dir() {
            total_size = total_size.saturating_add(metadata.len());
        }
    }
    Ok(total_size)
}
```

**Complexity**: O(n) where n = immediate files in directory
**Time**: < 100ms even for large directories
**User Experience**: Instant response

I updated the UI column header to "SIZE (files only)" to make this behavior clear. Users who want to see total subdirectory sizes can navigate into those directories.

### Performance Comparison

| Approach | Test Case | Time | Result |
|----------|-----------|------|--------|
| **Recursive** | 600GB home directory | 2-5 minutes | Frozen, appears crashed |
| **Shallow** | Same directory | < 100ms | Instant, responsive |

This optimization saved the project from being unusable for real-world scenarios.

## Results and Lessons Learned

### What Went Well

**1. Modular Architecture**: The separation between metrics, UI, and application logic made debugging and optimization straightforward. When I needed to fix the directory performance issue, I only touched `src/metrics/directory.rs`.

**2. Caching Strategy**: Query caching proved invaluable. During real-time filtering, users typically type one character at a time - the cache hit rate is very high since only the last character changes.

**3. Rust's Safety**: Rust's borrow checker caught several bugs during development, particularly around mutable references in filtering functions. What would have been runtime crashes in C became compile-time errors.

**4. RAII Pattern**: The `RawModeGuard` saved me countless times during development when code panicked, ensuring my terminal always stayed usable.

### What I Learned

**1. Profile Before Optimizing**: I initially spent time optimizing the process filtering logic, but the real bottleneck was directory traversal. Always measure first.

**2. User Experience Trumps Accuracy**: The shallow directory size calculation is technically "less accurate," but the instant responsiveness is worth the tradeoff. Users can still drill down to see details.

**3. Separation of Update and Render**: Decoupling metrics updates (500ms) from UI refresh (60 FPS) was crucial for both performance and responsiveness. Expensive system calls don't block user input.

**4. Error Handling Strategy**: Graceful degradation (skip unreadable files rather than failing) makes the tool more robust. Using `unwrap_or(Ordering::Equal)` for NaN handling in float comparisons prevents panics.

### Future Enhancements

There's still room for improvement:

- **Async Directory Calculation**: Show shallow sizes immediately, then update with deep sizes in the background
- **Historical Graphs**: Track CPU/memory over time with sparkline charts
- **Network Monitoring**: Add network I/O per process
- **Configuration File**: User-customizable color thresholds and update intervals

## Conclusion

Building SysMon was an excellent learning experience in systems programming, terminal UI design, and performance optimization. The project demonstrates several important software engineering principles:

- **Modularity**: Clear separation of concerns makes code maintainable and testable
- **Performance Awareness**: Understanding algorithmic complexity prevents catastrophic performance issues
- **User-Centric Design**: Sometimes "good enough fast" beats "perfectly accurate and slow"
- **Rust Idioms**: RAII, borrowing, and Result types lead to safer, more robust code

The final application runs smoothly even on systems with hundreds of processes and massive directory trees, provides a responsive 60 FPS UI, and handles errors gracefully.

If you're interested in building your own TUI applications, I highly recommend the `ratatui` framework - it provides powerful widgets and layout primitives while letting you focus on your application logic. The `sysinfo` crate makes cross-platform system metrics collection trivial.

The complete source code and architecture documentation are available in the repository. Feel free to explore, learn from it, or use it as a foundation for your own system monitoring tool.
