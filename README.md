# snailrace

`snailrace` is a Linux and macOS command benchmarker. It measures elapsed and
CPU time exactly from child-process accounting and samples process-tree memory,
processes, threads, and file descriptors where the OS exposes them.

## Build

Go 1.22 or newer is required. The default build strips paths, symbols, and
DWARF data to keep the binary small. The Makefile ignores a stale inherited
`GOROOT` and lets the selected `go` executable locate its own toolchain.

```sh
make build
```

## Use

Benchmark a program directly, without shell parsing:

```sh
./snailrace -n 20 -warmups 3 -- gzip -k sample.txt
```

Compare any number of shell commands. Execution order rotates each round to
distribute ordering bias:

```sh
./snailrace -n 20 -c 'grep needle data.txt' -c 'rg needle data.txt'
```

Create a self-contained HTML report:

```sh
./snailrace -format html -output ./reports -- ./my-program --flag
```

Create a report directory containing separate SVG chart files:

```sh
./snailrace -format svg -output ./reports -c 'grep x data' -c 'rg x data'
```

Measure one interactive TUI until the user exits it:

```sh
./snailrace tui -- htop
./snailrace tui -- vim file.txt
```

Compare TUIs for an equal fixed duration:

```sh
./snailrace tui -duration 30s -n 5 -c htop -c btop
```

Interactive TUI mode uses a real PTY, inherits the current terminal dimensions,
and follows terminal resize events. Fixed-duration comparisons capture geometry
once, reuse it for every run, and drain output without drawing it. `-width` and
`-height` can override inherited dimensions when a comparison must be
reproducible across different terminals or hosts.

Useful options:

| Option | Purpose |
|---|---|
| `-n`, `-runs` | Number of measured runs; default 10 |
| `-warmups` | Unmeasured warmup runs; default 1 |
| `-interval` | Resource sampling interval; Linux 10 ms, macOS 50 ms |
| `-c`, `-command` | Shell command; repeat for comparison mode |
| `-label` | Short display label; repeat in command order |
| `-prepare` | One-time setup command before warmups |
| `-format` | Saved format; repeat or comma-separate values; default `html` |
| `-output` | Save selected formats in this directory |
| `-show-output` | Forward measured command output to stderr |
| `-baseline` | 1-based override; default `0` selects the balanced winner |
| `tui` | Run commands inside a pseudo-terminal |
| `-duration` | Fixed TUI duration; required for comparisons |
| `-width`, `-height` | Optional reproducible TUI geometry |

Command output is discarded by default so it cannot corrupt stdout reports.
Raw observations are retained by JSON reports.
Human-readable text reports are always written to stdout and use ANSI colors
when stdout is a terminal. `-format` controls additional artifacts created by
`-output`; each path is announced on stderr. Markdown reports live in a report
directory and reference separate SVG files in its `charts` subdirectory. SVG
output creates that charts directory without report tables.
When stderr is an interactive terminal, measurement progress shows the active
tool, run or warmup iteration, rolling expected wall/CPU/RSS values, and ETA.
Below it, one snail track per command reserves 75% of its distance for completed
measurements and 25% for its rolling balanced index. Better expected commands
advance farther, heavier RAM use selects from eight trail weights, and the final
paths remain visible. The cursor is hidden during redraws and restored below the
finished race.
Progress is suppressed while an interactive TUI owns the terminal or command
output is being forwarded.
Text output uses status colors when stdout is a terminal and remains plain when
redirected. Markdown reports reference charts and never embed SVG markup.

Saved report names contain the measured tool and measurement start time, for
example `snail-my-program-20260711-140509.html`. Comparison names include the
first two tools, such as `snail-grep-vs-ripgrep-20260711-140509.txt`.

## Statistics

Reports include minimum, maximum, arithmetic mean, sample standard deviation,
median, p95, and a two-sided 95% Student's t confidence interval for the mean.
Quantiles use R-7 linear interpolation. No observations are removed as
outliers. A confidence interval characterizes run-to-run sampling uncertainty;
it does not remove systematic error or prove that observations are independent.

Comparison percentages use the selected baseline's mean. Better/worse status
requires the paired 95% Student's t interval of same-round differences to be
entirely favorable or unfavorable; otherwise the result is inconclusive. With
fewer than two paired runs it is always inconclusive. Process and thread counts
are labeled only higher/lower because fewer is not inherently better.
Sampled resource metrics are marked `sampling-limited` rather than better/worse
if any compared run lasts less than two process-sampling intervals.
Resource-cost better/worse labels assume every command performs equivalent work.

The automatic baseline is the lowest balanced index. That index is an
equal-weight geometric mean of normalized elapsed time, CPU cost, RAM cost, and
linked disk footprint; unavailable or sampling-limited categories are omitted.
Reports also name independent time, CPU, RAM, and linked-size winners. Ranking
cells show the measured value, percentage from that category's
best result, and category rank.

Comparison charts begin with grouped, zero-centered `Δ%` forest plots using
paired 95% confidence whiskers. Tradeoff scatter plots compare time, CPU, RAM,
and linked size with the lower-left corner as the efficient region. Absolute
distribution panels then show every run as a dot, the mean as a diamond, and
the mean confidence interval as a whisker. Available metrics remain visible
even when values are zero or equal. HTML and SVG reports use a high-contrast
Nord-derived tool palette consistently in command legends, ranking bars,
scatter legends, chart labels, and live progress. Colors remain stable by
command order; baseline badges and references use Frost blue.

Elapsed time uses Go's monotonic clock. CPU time and fallback maximum RSS use
the OS child resource record. Other peaks are sampled, so a process that starts
and exits within one sampling interval can be missed. Lower intervals improve
temporal resolution but increase observer CPU use and perturb short benchmarks.
Average CPU is total user plus system CPU time divided by wall time and may
exceed 100% for multithreaded programs. Mean resident memory is the arithmetic
mean of equally spaced process-tree samples.

For fixed-duration TUI runs, measurement ends when termination begins. CPU
accounting necessarily includes the usually small graceful-shutdown interval.
An unlimited interactive TUI result describes that user session and is useful
as a profile, but it is not a reproducible benchmark unless the input is
controlled.

Linux sampling walks only `/proc/<pid>/task/<pid>/children` for the measured
tree. macOS uses `ps`, so its default interval is deliberately coarser and does
not report open descriptor counts.

Tool metadata records the executable size, every statically discoverable linked
library and its size, and their deduplicated total footprint. Runtime-loaded
plugins and executables launched later by children are not included in that
static footprint and are identified as a limitation in reports.

## Architecture

The source is split by responsibility:

| Package | Responsibility |
|---|---|
| `cmd/snailrace` | Minimal executable entry point |
| `internal/app` | CLI validation and orchestration |
| `internal/analysis` | Balanced ranking and baseline selection |
| `internal/runner` | Command execution and round scheduling |
| `internal/platform` | Linux and macOS host/process probes |
| `internal/model` | Measurement data and statistical calculations |
| `internal/report` | Text, JSON, Markdown, HTML, and SVG rendering |
| `internal/style` | Shared terminal and report color palette |
