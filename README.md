# snailrace

`snailrace` benchmarks commands on Linux and macOS. It measures time, CPU,
memory, process structure, and static disk footprint.

![snailrace usage animation](/assets/snailrace.svg)

## Install

### Homebrew (macOS / Linux)

```sh
brew install shellcell/tap/snailrace
```

### Linux packages (apt / dnf / apk)

Enable the shellcell repository once — setup instructions at
<https://packages.shellcell.dev> — then:

```sh
sudo apt install snailrace   # Debian / Ubuntu
sudo dnf install snailrace   # Fedora / RHEL
sudo apk add snailrace       # Alpine
```

### Go

```sh
go install github.com/shellcell/snailrace/cmd/snailrace@latest   # Go 1.22+
```

### Prebuilt binaries

Download the archive for your OS/arch from
[Releases](https://github.com/shellcell/snailrace/releases); each contains the
binary, man page, and shell completions. To build from source, see
[Build](#build) below.

## Build

Requires Go 1.22+. macOS also requires a C compiler for the cgo `libproc`
sampler.

```sh
make build
```

## Examples

Benchmark one command:

```sh
./snailrace -n 20 -warmups 3 -- gzip -k sample.txt
```

Compare shell commands:

```sh
./snailrace -n 20 -c 'grep needle data.txt' -c 'rg needle data.txt'
```

An HTML report is saved to the current directory by default. Choose another directory:

```sh
./snailrace -format html -output ./reports -- ./my-program --flag
```

Measure an interactive TUI:

```sh
./snailrace tui -- htop
./snailrace tui -duration 30s -n 5 -c htop -c btop
```

## Options

| Option | Purpose |
|---|---|
| `-c`, `-command` | Shell command; repeat to compare |
| `-label` | Command display label |
| `-prepare` | One-time setup command |
| `-n`, `-runs` | Measured runs; default 10 |
| `-warmups` | Warmup runs; default 1 |
| `-interval` | Sampling interval; default 10 ms |
| `-index` | Index dimensions: `time,cpu,ram,disk` |
| `-baseline` | 1-based baseline; `0` selects the winner |
| `-f`, `-format` | Saved format; default `html` |
| `-o`, `-output` | Report directory; default current directory |
| `-verbose` | Full statistical tables on stdout |
| `-show-output` | Forward command output to stderr |
| `tui` | Run inside a pseudo-terminal |
| `-d`, `-duration` | Fixed TUI duration |
| `-width`, `-height` | TUI geometry |

## Metrics

| Group | Metrics |
|---|---|
| Time | Wall time |
| CPU | User, system, total, average utilization |
| Memory | Mean/peak tree RSS, OS max RSS, peak virtual memory |
| macOS memory | Peak aggregate physical footprint |
| Structure | Peak processes, threads, file descriptors where available |
| Disk | Executable and statically linked dynamic-library files |

Raw runs and summary statistics are available in JSON. Reports include mean,
sample standard deviation, median, p95, range, and a 95% Student's t interval.

## Method

- Runs use randomized counterbalanced order.
- The default index combines normalized time, CPU, and RAM costs.
- RAM cost combines mean and peak RSS.
- Sampling-limited RAM is excluded from the index.
- Ctrl+C keeps completed comparison rounds.
- Command output is discarded unless `-show-output` is set.

## Platform Notes

- Linux reads `/proc` and follows each process's children.
- macOS uses cgo with `proc_listpgrppids`, `proc_pidinfo`, and
  `proc_pid_rusage`.
- macOS physical footprint estimates memory pressure charged to each process.
- Tree RSS can double-count shared pages.
- Sampled peaks can miss activity shorter than the interval.
- macOS does not report file descriptor counts.

## Disk Footprint

- Disk-backed `@rpath`, `@loader_path`, and `@executable_path` libraries are
  resolved recursively.
- macOS dyld shared-cache libraries are listed but excluded from byte totals.
- Shared-cache libraries have no standalone size attributable to one command.
- Runtime plugins and executables launched by children are not included.

Commands that daemonize or escape their process group are unsupported.
