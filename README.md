# recipe-rotation-2

Small Go web app: recipe rotation home page and a recipe bank (create and list recipes in memory).

## Prerequisites

- [Go](https://go.dev/dl/) **1.25** or newer (match `go` version in [`go.mod`](go.mod)).

## Build

From the repository root:

```bash
go build -o recipe-rotation ./cmd
```

This produces an executable named `recipe-rotation` in the current directory.

## Run locally

```bash
go run ./cmd
```

Or run the binary you built:

```bash
./recipe-rotation
```

By default the server listens on **port 8080** (`http://127.0.0.1:8080/`).

To use another port, set `PORT` to the numeric port only (the app prepends `:`):

```bash
PORT=3000 go run ./cmd
```

Stop the server with Ctrl+C.

## Tests (optional)

```bash
go test ./...
```

The `test/e2e` package may call a remote URL; see `test/e2e/README.md` if those tests fail locally.
