# DiskBenchmark

DiskBenchmark is a native desktop and command-line utility for measuring
sequential and random file I/O. Both clients support concurrent workers,
configurable queue depth, buffered or direct I/O, latency percentiles, and
complete JSON or CSV reports. The desktop application and CLI use the same Go
benchmark engine, workload selection, validation, and report renderers.

> **Warning:** The benchmark writes at least the configured working-set size.
> Duration-based and fsync workloads can write substantially more. 
> Repeated large tests require enough free space in the target directory.

## Building

Go 1.25 or newer is required. Building the desktop application also requires
Node.js 20 or newer, npm, Wails v2.15.0, and the platform-native prerequisites
listed below.

Download the module dependencies once after cloning:

```text
go mod download
```

Install frontend dependencies after cloning or whenever `package-lock.json`
changes:

```text
cd frontend
npm ci
cd ..
```

Platform-specific source files are selected automatically by Go build tags. The
project does not require CGO.

### Native builds

Build on Windows with PowerShell:

```powershell
New-Item -ItemType Directory -Force .\dist | Out-Null
go build -trimpath -o .\dist\diskbenchmark.exe .\cmd\diskbenchmark
```

Build on Linux:

```bash
mkdir -p dist
go build -trimpath -o dist/diskbenchmark ./cmd/diskbenchmark
```

Build on an Intel or Apple Silicon Mac. Go automatically selects the native
architecture:

```bash
mkdir -p dist
go build -trimpath -o dist/diskbenchmark ./cmd/diskbenchmark
```

The macOS binaries are not code-signed or notarized. Distributing them to other
machines may therefore require normal Apple signing and notarization steps.

### Cross-compilation

Because the program is pure Go, Windows, Linux, and macOS binaries can be built
from any of those platforms. From PowerShell:

```powershell
New-Item -ItemType Directory -Force .\dist | Out-Null
$env:CGO_ENABLED = "0"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -o .\dist\diskbenchmark-windows-amd64.exe .\cmd\diskbenchmark

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -o .\dist\diskbenchmark-linux-amd64 .\cmd\diskbenchmark

$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -trimpath -o .\dist\diskbenchmark-darwin-amd64 .\cmd\diskbenchmark

$env:GOOS = "darwin"
$env:GOARCH = "arm64"
go build -trimpath -o .\dist\diskbenchmark-darwin-arm64 .\cmd\diskbenchmark

Remove-Item Env:GOOS
Remove-Item Env:GOARCH
Remove-Item Env:CGO_ENABLED
```

From Bash:

```bash
mkdir -p dist

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -trimpath -o dist/diskbenchmark-windows-amd64.exe ./cmd/diskbenchmark

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -o dist/diskbenchmark-linux-amd64 ./cmd/diskbenchmark

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
  go build -trimpath -o dist/diskbenchmark-darwin-amd64 ./cmd/diskbenchmark

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
  go build -trimpath -o dist/diskbenchmark-darwin-arm64 ./cmd/diskbenchmark
```

Set `GOARCH=arm64` in the Windows or Linux command when an ARM64 binary is
needed. Cross-compilation verifies that the target-specific code compiles, but
the resulting binary must be run and tested on its target operating system.

### Desktop application

The desktop application uses Wails v2, React, TypeScript, and Vite. Install the
matching Wails CLI and verify native prerequisites:

```text
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
wails doctor
```

Platform requirements:

| Platform | Native requirements |
| --- | --- |
| Windows | A supported 64-bit Windows release, Microsoft Edge WebView2 Runtime, and a Go toolchain. WebView2 is included with current Windows 10/11 installations and can also be installed from Microsoft. NSIS is optional and only required for `wails build -nsis`. |
| Linux | GTK 3, WebKitGTK development headers, a C compiler, and `pkg-config`. Debian/Ubuntu releases typically provide these through `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` or `libwebkit2gtk-4.1-dev`, `build-essential`, and `pkg-config`; use the WebKitGTK package available for the target distribution. |
| macOS | A supported macOS release and Xcode Command Line Tools. Wails uses the system WebKit framework. |

Start development mode from the project root. This launches Vite and opens the
native Wails window:

```text
wails dev
```

Create a native production build on the target operating system:

```text
wails build -trimpath -platform windows/amd64
```

Use the corresponding native platform value on Linux or macOS. Wails writes the
application to `build/bin` and embeds the compiled frontend, so the production
binary does not require a Vite development server. Build GUI artifacts on the
target operating system because each platform requires its native WebView and
toolchain.

Useful frontend checks can also be run independently:

```text
cd frontend
npm test
npm run typecheck
npm run build
```

The CLI remains a first-class interface and is independently buildable from
`cmd/diskbenchmark`. Its flags and structured output are suitable for scripts,
automation, and environments where a native WebView is unavailable.

## Running

### Desktop GUI

Start a production Windows build with:

```powershell
.\build\bin\DiskBenchmark.exe
```

In the application:

1. Choose a target directory and working-set and block sizes.
2. Select a built-in workload or a versioned JSON workload suite.
3. Expand **Advanced options** to configure iterations, duration, concurrency,
   direct I/O, cache control, verification, and retained files.
4. Review the write-impact warning. Elevated-impact configurations require an
   additional confirmation.
5. Start the benchmark and monitor its lifecycle, iteration, workload, elapsed
   time, and progress. Cancellation may wait for the active operating-system I/O
   call before cleanup finishes.
6. Review exact summary and per-iteration values, environment and storage
   details, verification status, and any retained paths.
7. Export the complete report as JSON or CSV. Export errors are shown in the
   application and are never reported as successful writes.

Direct I/O and cache-control requirements are explicit. The GUI reports the
native operating-system error instead of silently falling back to buffered I/O.

### Command-line interface

On Windows:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP `
  -size 1GiB -block-size 1MiB
```

On Linux or macOS:

```bash
./dist/diskbenchmark -path "${TMPDIR:-/tmp}" \
  -size 1GiB -block-size 1MiB
```

Run with platform-native direct I/O on Windows:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB -block-size 1MiB `
  -io-mode direct
```

Run with platform-native direct I/O on Linux or macOS:

```bash
./dist/diskbenchmark -path "${TMPDIR:-/tmp}" \
  -size 1GiB -block-size 1MiB -io-mode direct
```

Run a concurrent random workload with 70 percent reads on Windows:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB -block-size 4KiB `
  -workload random -random-read-percent 70 -random-seed 42 `
  -workers 4 -queue-depth 8
```

Write a machine-readable report:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 64MiB -workload all `
  -output json > result.json

.\dist\diskbenchmark.exe -path $env:TEMP -size 64MiB -workload all `
  -output csv > result.csv
```

Supported size suffixes are `B`, `KiB`, `MiB`, `GiB`, and `TiB`. Units are
binary: one MiB is 1,048,576 bytes.

## Testing

Run the full test suite natively on Windows, Linux, and macOS:

```text
go test ./...
```

The full suite includes unit tests, file lifecycle integration tests, concurrent
I/O tests, structured-output tests, and a small native direct-I/O benchmark.
The direct-I/O test uses Go's system temporary directory. That directory must be
on a filesystem that supports the platform's direct-I/O mechanism.

To place temporary test files on a specific Windows volume:

```powershell
$env:TEMP = "D:\Temp"
$env:TMP = "D:\Temp"
go test ./...
```

To select the temporary filesystem on Linux or macOS:

```bash
TMPDIR=/path/to/test-volume go test ./...
```

Run only the native direct-I/O integration test:

```text
go test -v ./benchmark -run TestRunnerDirectIO
```

Useful additional checks:

```text
# Repeat tests to expose nondeterministic concurrency failures.
go test -count=10 ./...

# Report package-level statement coverage.
go test -cover ./...

# Run static analysis.
go vet ./...
```

Run the race detector when a supported C toolchain is installed:

```text
go test -race ./...
```

The race detector requires CGO and a working C compiler, even though normal
builds do not. Tests must be executed natively for each operating system to
exercise its direct-I/O backend; a cross-compiled test binary cannot be run on
the build host.

Run the frontend unit and component tests with mocked Wails bindings and runtime
events:

```text
cd frontend
npm test
```

Filesystem and benchmark behavior remains covered by the Go service and engine
tests rather than browser mocks.

## Options

| Option | Default | Description |
| --- | --- | --- |
| `-path` | `.` | Directory where temporary test files are created |
| `-size` | `1GiB` | Bytes processed by each workload |
| `-block-size` | `1MiB` | Bytes in each I/O operation |
| `-iterations` | `1` | Number of complete write/read runs |
| `-warmup-iterations` | `0` | Number of unmeasured warm-up runs |
| `-sync` | `true` | Include `File.Sync` in the write measurement |
| `-keep-file` | `false` | Retain generated test files for inspection |
| `-verify-data` | `false` | Verify every byte after measured workloads |
| `-workload` | `sequential` | Run `sequential`, `random`, `mixed`, `overwrite`, `fsync`, or `all` |
| `-duration` | `0` | Minimum duration for each workload; zero processes the working set once |
| `-suite` | none | Load workloads from a versioned JSON suite file |
| `-workers` | `1` | Workers assigned to separate contiguous file regions |
| `-queue-depth` | `1` | Maximum concurrent operations per worker |
| `-random-read-percent` | `50` | Percentage of random operations that are reads |
| `-random-seed` | `1` | Seed used to create the repeatable random plan |
| `-io-mode` | `buffered` | Use `buffered` or platform-native `direct` I/O |
| `-cache-control` | `off` | Cache policy before reads: `off`, `attempt`, or `require` |
| `-output` | `text` | Render the report as `text`, `json`, or `csv` |

Temporary files use unique names and are removed after each iteration unless
`-keep-file` is set. Cleanup failures are reported as benchmark errors.

## Workload patterns

The built-in workload selections are:

| Selection | Behavior |
| --- | --- |
| `sequential` | Sequential write followed by sequential read |
| `random` or `mixed` | Seeded random access using the configured read/write percentage |
| `overwrite` | Populate the file outside the timed section, then overwrite existing blocks sequentially |
| `fsync` | Write sequential blocks and call `File.Sync` after every block; sync latency is included per operation |
| `all` | Run sequential write/read, random mixed, overwrite, and fsync-per-block workloads |

`overwrite` and standalone `sequential-read` suite workloads populate an empty
test file before timing begins. The fsync workload always synchronizes every
block, independently of the final `-sync` setting.

### Duration-based workloads

By default, each workload processes `-size` bytes once. Set `-duration` to repeat
the same working set until at least the requested duration has elapsed:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB `
  -workload random -duration 30s
```

Every worker completes one full pass before the deadline is honored. This keeps
the test file fully initialized and means a slow first pass can exceed the
requested duration. For duration-based tests, `-size` is the working-set size;
reported bytes and operations can be larger. Final synchronization remains part
of write-workload duration.

### Versioned workload suites

Use `-suite` to execute an ordered workload definition:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB `
  -suite .\examples\workload-suite.json
```

The current schema version is `1`:

```json
{
  "version": 1,
  "workloads": [
    {
      "type": "sequential",
      "duration": "2s"
    },
    {
      "type": "random",
      "name": "database-style mixed I/O",
      "duration": "5s",
      "random_read_percent": 70,
      "random_seed": 42
    },
    {
      "type": "overwrite"
    },
    {
      "type": "fsync-write",
      "name": "durable block writes"
    }
  ]
}
```

Supported suite types are `sequential`, `sequential-write`,
`sequential-read`, `random`, `overwrite`, and `fsync-write`. A `sequential`
entry expands to separate write and read workloads. Optional `duration` values
use Go duration syntax. Random entries can override `random_read_percent` and
`random_seed`; omitted values use the command-line configuration.

Suite parsing rejects unknown fields, unsupported versions, invalid settings,
trailing JSON values, empty workload lists, and duplicate result names.
Per-workload duration overrides the global `-duration`. Custom `name` values are
useful when a suite contains multiple instances of one workload type. `-suite`
and `-workload` cannot be supplied together.

## Interpreting results

The program reports throughput, IOPS, elapsed time, operation counts, the
read/write mix, and average, p50, p95, p99, and maximum operation latency. With
multiple iterations it also reports minimum, average, and maximum throughput
and IOPS.

Random workloads first populate the test file outside the timed section, then
access every block exactly once per complete pass in a seeded random order. The
requested read/write percentage is converted to an exact per-pass operation
count, rounded to the nearest whole operation. Reusing a seed, file size, and
block size produces the same operation plan.

Each worker owns a contiguous region of the test file. Queue depth controls the
number of concurrent I/O lanes in each worker, so the maximum number of
in-flight calls is `workers * queue-depth`. Total bytes and operation counts do
not change when concurrency changes.

The read workload is performed immediately after the write workload. In
buffered mode, operating system and device caches can therefore make read
results substantially faster than uncached storage. Direct mode bypasses the
operating system's normal file cache as described below, but device controller
and drive caches may still affect results.

Close other I/O-heavy programs, use a data size larger than available filesystem
cache where practical, and compare results using the same path, size, block size,
I/O mode, and synchronization setting.

## Direct I/O

`-io-mode direct` selects the native cache-bypassing mechanism for the current
platform:

| Platform | Mechanism | Alignment |
| --- | --- | --- |
| Windows | `FILE_FLAG_NO_BUFFERING`, `FILE_FLAG_WRITE_THROUGH`, and overlapped I/O | Logical sector size reported by the target volume |
| Linux | `O_DIRECT` | Filesystem block size reported by `statfs` |
| macOS | `F_NOCACHE` | No additional buffer or request-size alignment required |

On Windows and Linux, both `-size` and `-block-size` must be exact multiples of
the alignment reported for the target path. Every worker and queue-depth lane
uses a separately allocated, correctly aligned buffer. The effective I/O mode
and alignment are included in text, JSON, and CSV reports.

Direct I/O is explicit and never silently falls back to buffered I/O. If the
target filesystem does not support the platform mechanism, the benchmark exits
with the operating-system error. This is common for some network, virtual, and
userspace filesystems. Choose `-io-mode buffered` when direct access is not
available.

The macOS `F_NOCACHE` mechanism disables caching for the file but is not
identical to Linux `O_DIRECT` or Windows unbuffered I/O. Cross-platform
comparisons should therefore record the operating system and mechanism and
should not assume identical kernel behavior.

## Cache control

`-cache-control` controls what the benchmark does before a timed workload that
contains reads:

| Mode | Behavior |
| --- | --- |
| `off` | Perform no additional cache-control operation. This is the default. |
| `attempt` | Try the safe platform mechanism and continue if it is unsupported or fails. Record the exact outcome in the measurement. |
| `require` | Require cache control or direct-I/O bypass. Abort the benchmark if the operation is unsupported or fails. |

The platform behavior is:

| Platform | Buffered-I/O mechanism | Notes |
| --- | --- | --- |
| Windows | None | Windows has no safe, unprivileged per-file cache-eviction API. `attempt` reports `unsupported`; `require` fails. Use `-io-mode direct` to bypass the normal cache. |
| Linux | `fsync` followed by `posix_fadvise(..., FADV_DONTNEED)` | This asks the kernel to discard cached pages. It is advisory, so `applied` means the request succeeded, not that every page was guaranteed to be evicted. |
| macOS | `fsync` followed by `fcntl(..., F_NOCACHE, 1)` | This disables caching for subsequent access through the benchmark file descriptor; it is not a general cache purge. |

When `-io-mode direct` is selected, cache-control `attempt` and `require` are
satisfied by the direct-I/O cache bypass and report `bypassed`. The benchmark
never uses privileged global cache purges and never silently labels an
unsupported or failed operation as a cold-cache run.

Cache-control mode, API, status, and error detail are included in text, JSON,
and CSV measurements. Random workloads with zero percent reads report cache
control as `not_applicable`.

## Latency histogram

Latency percentiles use a fixed-memory logarithmic histogram instead of storing
one duration for every I/O operation. Each workload execution uses 2,017
buckets, approximately 16 KiB of counters, whether it performs ten operations
or billions. Operation results are streamed through a bounded channel and are
aggregated immediately.

Average and maximum latency and the sample count are retained directly.
Percentiles are reported as the upper bound of a logarithmic bucket, with 32
sub-buckets per power-of-two range. This bounds percentile resolution while
preventing benchmark memory use from growing with the operation count. Reports
include both `sample_count` and `bucket_count` so consumers can identify the
aggregation method.

## Safety and environment checks

Before every warm-up and measured iteration, DiskBenchmark checks the free space
available to the current user on the target filesystem. The iteration is
rejected before its temporary file is created when `-size` exceeds the available
space. This is a point-in-time preflight check; other processes can still consume
space while a benchmark is running, so write errors remain possible and are
reported normally.

Reports include available bytes before and after the complete benchmark. Every
measured iteration also records:

- Logical test-file size.
- Allocated on-disk bytes.
- Whether allocated bytes are smaller than logical bytes, indicating sparse-file
  or filesystem-compression behavior.

### Warm-up iterations

Use warm-up runs to initialize device, filesystem, and runtime state before
collecting reported measurements:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB `
  -warmup-iterations 2 -iterations 5
```

Warm-ups execute the selected workloads with the configured I/O and cache
policies, but they are excluded from iteration results and summaries. Warm-up
files are always deleted, even when `-keep-file` is selected, and optional data
verification is performed only for measured iterations. The report records the
number of successfully completed warm-ups.

### Data verification

Enable post-benchmark integrity verification with:

```powershell
.\dist\diskbenchmark.exe -path $env:TEMP -size 1GiB -verify-data
```

Verification reads the complete test file after all timed workloads and compares
it byte-for-byte with the deterministic data pattern used for writes. It uses
aligned buffers in direct-I/O mode, supports cancellation, and reports the first
incorrect byte offset and values. Verification time and bytes are reported
separately and never included in workload throughput, IOPS, or latency.

## Design

The `benchmark` package owns configuration, execution, environment metadata,
metrics, and workload implementations. Workloads implement a small `Workload`
interface and can be passed to `benchmark.NewRunner`, allowing later workload
types to reuse deterministic concurrency, iteration handling, and file
lifecycle. The `output` package renders reports without affecting benchmark
execution, and the `units` package handles size parsing and presentation.

JSON reports contain the complete configuration, runtime and machine metadata,
filesystem information when available, all iteration measurements, and
summaries. CSV reports contain one measurement row per workload and additional
summary rows. Diagnostic warnings are written to standard error so redirected
JSON and CSV remain valid.

## Privacy

Benchmark configuration, measurements, machine and filesystem metadata, and
generated reports remain on the local computer. DiskBenchmark does not upload
benchmark data. Data leaves the computer only when the user explicitly exports
and shares a report or retained benchmark file.