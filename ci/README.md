# Local Woodpecker CI

This folder contains everything needed to run a local Woodpecker CI stack and a pipeline that runs the e2e test in `test/e2e/e2e_test.go`.

## Files

- `deploy_local_woodpecker.sh`: one-command setup and lifecycle script
- `docker-compose.yml`: local services (`woodpecker-server`, `woodpecker-agent`)
- `.woodpecker.yml`: pipeline definition for the e2e test

## Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose plugin)
- `openssl`

## 1) Start the local CI stack

From repo root:

```bash
chmod +x ci/deploy_local_woodpecker.sh
./ci/deploy_local_woodpecker.sh up
```

This command:

- Creates `ci/.env` on first run
- Generates `WOODPECKER_AGENT_SECRET`
- Starts Woodpecker on `http://localhost:8000`

## 2) Expose local Woodpecker to GitHub

GitHub must reach your Woodpecker server to deliver webhooks, so expose `localhost:8000` through a public tunnel.

### Option A: ngrok

```bash
ngrok http 8000
```

Use the HTTPS forwarding URL from ngrok, for example:

- `https://abc123.ngrok-free.app`

### Option B: cloudflared

```bash
cloudflared tunnel --url http://localhost:8000
```

Use the generated `https://...trycloudflare.com` URL.

### Set `WOODPECKER_HOST` to your public URL

Edit `ci/.env` and replace:

```bash
WOODPECKER_HOST=http://localhost:8000
```

with:

```bash
WOODPECKER_HOST=https://your-public-url.example
```

Restart services after changing it:

```bash
./ci/deploy_local_woodpecker.sh restart
```

## 3) Configure GitHub OAuth for Woodpecker

Woodpecker needs a forge OAuth app before login works.

1. In GitHub, create an OAuth App:
   - Homepage URL: `https://your-public-url.example`
   - Authorization callback URL: `https://your-public-url.example/authorize`
2. Copy the generated client ID and client secret.
3. Edit `ci/.env` and set:
   - `WOODPECKER_GITHUB_CLIENT`
   - `WOODPECKER_GITHUB_SECRET`
4. Restart services:

```bash
./ci/deploy_local_woodpecker.sh restart
```

## 4) Connect GitHub repo and enable webhooks

1. Open your public Woodpecker URL (`https://your-public-url.example`)
2. Login with your GitHub account
3. Activate your repository in Woodpecker
4. Ensure the GitHub OAuth app has permissions to manage hooks (`admin:repo_hook`)
5. Confirm a webhook was added on GitHub:
   - Repository -> Settings -> Webhooks
   - URL should point to your public Woodpecker webhook endpoint

## 5) Add pipeline to your repository

Woodpecker reads `.woodpecker.yml` from the repository root by default.
This project keeps the pipeline in `ci/.woodpecker.yml`.

Create a symlink (or copy) at repo root:

```bash
ln -sf ci/.woodpecker.yml .woodpecker.yml
```

## 6) Trigger pipeline from GitHub

Push a commit or open a pull request on GitHub.

This pipeline is configured to run on:

- `push`
- `pull_request`

## 7) Configure secure GCP secrets in Woodpecker

Do not commit cloud credentials to git. Add them as repository secrets in Woodpecker:

1. In GCP: **IAM & Admin** → **Service accounts** → pick or create a service account → **Keys** → **Add key** → **JSON** (only if your org allows key creation).
2. Open your repo in Woodpecker → **Settings** → **Secrets**.
3. Add secret **`gcp_sa_key_json`**: paste the **entire** contents of that JSON key file (one secret value, multiline JSON is fine).
4. Add secret **`gcp_project_id`**: your GCP **project ID** (not display name).

The pipeline writes the JSON to a short-lived temp file, runs `gcloud auth activate-service-account`, sets the project, then uses `gcloud compute ssh` to reach the VM.

Target VM:

- instance: `recipe-rotation`
- zone: `us-central1-b`
- external IP: `34.60.141.247`

**IAM (tune to your setup):** the service account needs enough access to resolve the instance and open SSH (often **Compute Instance Admin** plus OS Login / SSH-related roles, or equivalent custom roles). If `sudo -n` fails in preflight, grant passwordless sudo for that principal on the VM or use a different deploy method.

Remove or rotate the key in GCP if it is ever exposed; prefer narrow roles and a dedicated deploy-only service account.

## 8) Lock down a public Woodpecker instance

If your Woodpecker URL is public, use these settings in `ci/.env`:

```bash
WOODPECKER_OPEN=false
WOODPECKER_ADMIN=your-github-username
```

- `WOODPECKER_OPEN=false` disables open self-registration.
- `WOODPECKER_ADMIN` restricts admin privileges to your account.

Then restart:

```bash
./ci/deploy_local_woodpecker.sh restart
```

Also recommended:

- Keep repositories private unless intentionally public.
- Limit who can activate repositories in Woodpecker.
- Rotate cloud keys periodically and use least-privilege IAM roles.

## 9) Secret leakage and pipeline logs

The deploy step writes the GCP key from a Woodpecker secret to a temporary file and removes it at the end of the step. The key value is not echoed to logs by default.

To keep it safe:

- Do not add `set -x` in pipeline commands.
- Do not print environment variables or `cat` the key file.
- Restrict secret exposure for untrusted pull requests/forks in Woodpecker secret settings.

The pipeline step runs:

```bash
go test ./test/e2e -v -run TestEndpointServesTraffic
```

The pipeline installs nginx, then cross-builds `cmd` for **linux/amd64**, copies the binary plus `deploy/*` to the VM, runs a **systemd** unit on port **8080**, and configures nginx to **reverse-proxy** port **80** to the app (so the e2e check for `recipe rotation` hits the Go server). If the VM is **ARM** (e.g. T2A), change `GOARCH` in `ci/.woodpecker.yml` to `arm64`.

## Useful commands

```bash
./ci/deploy_local_woodpecker.sh status
./ci/deploy_local_woodpecker.sh logs
./ci/deploy_local_woodpecker.sh down
```
