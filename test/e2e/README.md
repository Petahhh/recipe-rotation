# E2E Endpoint Traffic Test

This folder contains an end-to-end Go test that checks whether traffic is being served from:

- `http://34.60.141.247:80/`

The test expects a successful HTTP `2xx` response and a body that contains the text `recipe rotation` (e.g. from your nginx site).

## Prerequisites

- Go installed (`go version`)

## Run the test

From the repository root:

```bash
go test ./test/e2e -v
```

Or from this folder:

```bash
go test -v
```
