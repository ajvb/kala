# Contributing to Kala

Kala needs Go 1.26 or later. The repository holds two Go modules: the server
(`github.com/ajvb/kala/v2`, in the root) and the Go client
(`github.com/ajvb/kala/client/v2`, in `client/`). Work for v2 targets the
`v2-dev` branch.

## Building

```bash
make bin/kala
./bin/kala version
```

`docker build -t kala .` builds the container image.

## Running the tests

| Command | What it runs | Needs |
| --- | --- | --- |
| `go test ./...` | The unit tests of the server module (`cd client` for the client module) | Nothing else |
| `go test -race ./...` | The same, with the race detector, as CI runs them | Nothing else |
| `make test-integration` | The integration tests: storage backends and the container helper | Docker |
| `make test` | `go vet`, the linters, then every test in both modules | Docker, [golangci-lint](https://golangci-lint.run/) v2 |
| `make bench` | Each benchmark once | Nothing else |

Integration tests carry the `integration` build tag and start their own
database containers. Without Docker they fail; they never pass without
running.

## Writing tests

- **Name a test for what it checks**, for example
  `TestOverlappingRemoteJobRunsLeaveTheJobUnchanged`. Do not put issue, ticket
  or other reference numbers in test names or comments.
- **Check something specific**, not just "no error".
- **A bug fix comes with a test that fails without the fix.**
- **Control time through the clock.** The scheduling code in `job` reads the
  time and starts timers only through its clock (package `clock`); tests use
  `clock.NewFake` instead of sleeping. CI fails if `job` calls `time.Now`,
  `time.Sleep` or the other time functions directly.
- **Clean up.** Use `t.TempDir()` for files, and stop what a test starts. Do
  not write outside the temporary directory, `testdata/` or a container the
  test started.
- **A flaky test is fixed or removed** in the pull request that notices it.
  Do not retry it until it passes.
- **Backend tests are integration tests:** give them the `integration` build
  tag and start their database with `internal/containers`.

## Fuzzing

Fuzz targets are ordinary Go fuzz tests (`func FuzzXxx(f *testing.F)`).
Every pull request runs each one for 30 seconds
(`.github/scripts/fuzz.sh 30s` runs them all locally). To fuzz one for longer:

```bash
go test ./utils/iso8601 -run '^$' -fuzz '^FuzzFromString$' -fuzztime 10m
```

CI logs are public, so a failing fuzz target must not print its input: use
`internal/fuzzreport` and failure messages without the input. Go saves a
failing input under the package's `testdata/fuzz/`. If it could be a security
problem, do not post it in an issue or pull request: report it through
GitHub's private vulnerability reporting (the repository's Security tab).
Commit the input together with the fix, as a regression test.

## Benchmarks

Benchmarks are Go benchmarks (`func BenchmarkXxx(b *testing.B)`) in the
package they measure. On every pull request CI runs them against the base
branch on the same machine (`.github/scripts/bench.sh`). The check fails if a
benchmark is more than 20% slower than the base, measured again on a re-run.
The comparison is on the run's summary page.

## Web interface libraries

The web interface's third-party libraries are not in the repository.
`make bin/kala` installs them with npm (`make webui-libs`) and embeds them, so
building the web interface from source needs npm. A plain `go build` leaves
them out, and the web interface then says it is not available.
`webui/LIBRARIES.md` says how to update them.

## Architecture

| Package | What it does |
| --- | --- |
| `main.go`, `cmd` | The `kala` command: `serve`, `run` and `version` |
| `api` | The HTTP API (`/api/v1`) and the web interface's routes |
| `job` | Jobs, their schedules, the cache that holds them, the timers that start them and the runner that executes local commands and remote HTTP calls |
| `job/storage/*` | The storage backends: BoltDB, Consul, MongoDB, MySQL, PostgreSQL and Redis |
| `clock` | Real and fake clocks for the scheduling code |
| `utils/iso8601` | The ISO 8601 duration parser used by schedules |
| `webui` | The web interface, embedded in the binary |
| `client` | The Go client module |
| `internal/containers` | Starts database containers for integration tests |
| `internal/fuzzreport` | Reports fuzz failures without their input |
| `internal/version` | The version, commit and build date of the binary |

## Pull requests and CI

CI runs on GitHub Actions (`.github/workflows/ci.yml`): build, unit tests with
the race detector on two Go versions, integration tests, a smoke test, `go
vet`, golangci-lint, the clock-usage check, short fuzzing, the release build
and image, benchmarks and a coverage report. The `ci-ok` check passes only when every required job
does.
