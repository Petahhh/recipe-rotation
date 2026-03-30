# E2E Endpoint Traffic Test

This folder contains an end-to-end Go test that checks whether traffic is being served from:

- `http://34.60.141.247:80/`

The test expects a successful HTTP `2xx` response. Because nothing is deployed there yet, the test is expected to fail for now.

## Prerequisites

- Go installed (`go version`)

## Run the test

From the repository root (this project currently has no `go.mod`):

```bash
GO111MODULE=off go test ./test/e2e -v
```

Or from this folder:

```bash
GO111MODULE=off go test -v
```
