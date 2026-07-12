# snailrace

`snailrace` is a Linux and macOS command benchmarker. It measures elapsed time,
uses OS-reported CPU accounting for the waited process, and samples process-tree
memory, processes, threads, and file descriptors where the OS exposes them.

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

Compare any number of shell commands. A recorded random seed drives
counterbalanced execution blocks to reduce ordering bias:

```sh
./snailrace -n 20 -c 'grep needle data.txt' -c 'rg needle data.txt'
```

JSON reports retain the exact warmup and measurement order as one-based tool
indices. Complete blocks balance every tool across every execution position;
the final partial block is randomized but cannot be fully balanced.

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
and follows terminal resize events. Every fixed-duration run is noninteractive:
it captures geometry once, reuses it for every run, and drains output without
drawing it. `-width` and `-height` can override inherited dimensions when a
comparison must be reproducible across different terminals or hosts.

Useful options:

| Option | Purpose |
|---|---|
| `-c`, `-command` | Shell command; repeat for comparison mode |
| `-label` | Display label; repeat once per command, or name a single command |
| `-prepare` | One-time setup command before warmups |
| `-n`, `-runs` | Number of measured runs; default 10 |
| `-warmups` | Unmeasured warmup runs; default 1 |
| `-interval` | Resource sampling interval; Linux 10 ms, macOS 50 ms |
| `-index` | Balanced index dimensions; comma-separate `time`, `cpu`, `ram`, `disk`; default `time,cpu,ram` |
| `-baseline` | 1-based override; default `0` selects the balanced winner |
| `-f`, `-format` | Saved format; repeat or comma-separate values; default `html` |
| `-o`, `-output` | Save selected formats in this directory |
| `-verbose` | Print full per-tool statistical tables to stdout instead of the compact summary |
| `-show-output` | Forward measured command output to stderr |
| `tui` | Run commands inside a pseudo-terminal |
| `-d`, `-duration` | Fixed TUI duration; required for comparisons |
| `-width`, `-height` | Optional reproducible TUI geometry |
| `-v`, `-version` | Print version and exit |

Command output is discarded by default so it cannot corrupt stdout reports.
Raw observations are retained by JSON reports.
Before measurement starts, a legend maps each tool label to its full command,
colored per tool when stderr is a terminal. Human-readable text reports are
always written to stdout and use ANSI colors when stdout is a terminal. By
default stdout shows a compact summary: per-tool bar charts for the balanced
index, time, CPU, RAM, and disk, each with the mean and standard deviation, bars
colored per tool. `-verbose` prints the full per-tool statistical tables and
baseline-delta tables instead; that detail is also always available in JSON and
HTML reports. `-format` controls additional artifacts created by
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
The interval is reported only with at least two observations.
Critical values are tabulated through 30 degrees of freedom and use a
high-accuracy Cornish-Fisher approximation above that.
Quantiles use R-7 linear interpolation. No observations are removed as
outliers. A confidence interval characterizes run-to-run sampling uncertainty;
it does not remove systematic error or prove that observations are independent.

Comparison percentages use the selected baseline's mean. Better/worse status
requires the paired 95% Student's t interval of same-round differences to be
entirely favorable or unfavorable; otherwise the result is inconclusive. With
fewer than two paired runs it is always inconclusive. Process and thread counts
are labeled only higher/lower because fewer is not inherently better.
Sampled resource metrics are marked `sampling-limited` rather than better/worse
unless every compared run spans two intervals and has two valid samples.
Resource-cost better/worse labels assume every command performs equivalent work.
Intervals are pointwise and unadjusted for multiple comparisons. Comparisons to
an automatically selected baseline are exploratory post-selection inference.
Statistical direction does not imply that an effect is practically important.

The automatic baseline is the lowest balanced index. That index is an
equal-weight geometric mean of normalized cost categories. By default it
includes elapsed time, CPU cost, and RAM cost; linked disk footprint is still
measured and reported but excluded from the composite because static install
size is a different concern than runtime performance. `-index` selects the
included categories from `time`, `cpu`, `ram`, and `disk`. Unavailable or
sampling-limited categories are omitted, as are categories containing
nonpositive values because cost ratios to a zero best value are undefined.
Fixed-duration TUI ranking replaces elapsed/total CPU with one average-CPU
category because the configured active duration is shared.
Reports also name descriptive time, CPU, RAM, and linked-size leaders. Ranking
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

Elapsed time uses Go's monotonic clock. CPU time and OS-reported maximum RSS use
the waited process's resource record. The OS value is a high-water mark, not an
aggregate tree measurement. Aggregate tree RSS is sampled separately and
can double-count shared pages. Other peaks are sampled, so a process that starts
and exits within one sampling interval can be missed. Lower intervals improve
temporal resolution but increase observer CPU use and perturb short benchmarks.
Average CPU is total user plus system CPU time divided by wall time and may
exceed 100% for multithreaded programs; outside equal-duration experiments it
is descriptive rather than inherently better or worse. Mean resident memory is
the arithmetic mean of valid process-tree samples.

Fixed-duration TUI runs require the process to survive to the deadline. Their
reported launch-to-exit window includes the graceful-shutdown interval so wall,
CPU, and sampled resources cover the same interval.
An unlimited interactive TUI result describes that user session and is useful
as a profile, but it is not a reproducible benchmark unless the input is
controlled.

Linux sampling walks every task's `children` file for each discovered process.
The snapshot remains best-effort under process exit and reparenting races.
macOS uses `ps`, so its default interval is deliberately coarser and does not
report open descriptor counts.

Tool metadata estimates the executable size and statically discoverable linked
library and its size, and their deduplicated total footprint. Runtime-loaded
plugins and executables launched later by children are not included in that
static footprint and are identified as a limitation in reports.
Commands that daemonize or escape their process group are unsupported because
their resource use cannot be contained or attributed reliably.

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
