# Maintaining worklog

## Run the tests

```sh
go test ./...
```

The acceptance tests build the CLI and run it as a separate program, with
temporary home, settings, repository, and journal folders. They never touch your
real data.

## Publish a release

Push a version tag from `main`:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions then runs the tests on Linux and macOS. It builds all four
platforms with `scripts/release.sh` and tries each build on its own platform with
`scripts/smoke-test.sh`, in a clean environment without Go. The release is
published only when every step passes.

To try a release build locally:

```sh
scripts/release.sh v0.0.0
scripts/smoke-test.sh dist/worklog_v0.0.0_darwin_arm64 v0.0.0
```
