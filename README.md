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

### Arch (AUR)

With an AUR helper (e.g. `paru` or `yay`):

```sh
paru -S snailrace       # builds from source
paru -S snailrace-bin   # prebuilt binary
```

Or manually:

```sh
git clone https://aur.archlinux.org/snailrace.git   # or snailrace-bin.git
cd snailrace && makepkg -si
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

Use `-no-save` when only stdout output is wanted.

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
| `-no-save` | Do not save report files |
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
Every format identifies the dimensions actually included in the balanced index
and records sampling or metric-availability caveats.

## Method

- Runs use randomized counterbalanced order.
- The default index combines normalized time, CPU, and RAM costs.
- RAM cost combines mean and peak RSS.
- Sampling-limited RAM is excluded from the index.
- Commands with non-zero measured exits continue running but are excluded from rankings.
- If none of the selected dimensions is usable, reports mark the balanced ranking unavailable.
- Ctrl+C keeps completed comparison rounds.
- Command output is discarded unless `-show-output` is set.
- Native executables run through the inspected artifact and are identity/hash-checked after measurement.
- Scripts preserve their original invocation path so `$0`-relative behavior is unchanged.

Saved reports are staged before publication and receive a numeric suffix rather
than overwriting an existing report with the same timestamp.

## Platform Notes

- Linux reads `/proc` and measures every member of the command's process group,
  including descendants reparented while the benchmark is running.
- macOS uses cgo with `proc_listpgrppids`, `proc_pidinfo`, and
  `proc_pid_rusage`.
- macOS physical footprint estimates memory pressure charged to each process.
- Physical-footprint validity is tracked per run; unavailable samples are not treated as zero.
- Tree RSS can double-count shared pages.
- Sampled peaks can miss activity shorter than the interval.
- macOS does not report file descriptor counts.

## Disk Footprint

- For a simple external `-c` command, disk footprint describes its leading executable;
  shell builtins describe the shell itself.
- Disk-backed `@rpath`, `@loader_path`, and `@executable_path` libraries are
  resolved recursively.
- macOS dyld shared-cache libraries are listed but excluded from byte totals.
- Shared-cache libraries have no standalone size attributable to one command.
- Runtime plugins and executables launched by children are not included.

Commands that daemonize or escape their process group are unsupported.
