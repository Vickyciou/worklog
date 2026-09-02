# worklog

`worklog` is a portable command-line tool for recording and rendering work journals.

Building and testing require Go 1.23 or newer. The compiled CLI does not require a
Go installation at runtime.

## Build

From the repository root, build the CLI with:

```sh
go build -o ./bin/worklog ./cmd/worklog
```

## Acceptance tests

Run the external-process acceptance suite with:

```sh
go test ./...
```

The suite compiles its own CLI executable and runs it with isolated temporary home,
configuration, state, Git repository, and journal destination directories.
