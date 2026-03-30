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

1. Open your repo in Woodpecker -> `Settings` -> `Secrets`
2. Add secret `gcp_adc_json` with the full JSON contents of:
   `~/.config/gcloud/application_default_credentials.json`
3. Add secret `gcp_project_id` with your GCP project ID

The pipeline uses these secrets in a deploy step to install/start Nginx on your VM.

- instance: `recipe-rotation`
- zone: `us-central1-b`
- external IP: `34.60.141.247`

Required IAM permissions depend on what credentials you use:

- If using ADC from a logged-in user, ensure that user is allowed to:
  - connect to the VM over SSH (OS Login / project-wide SSH permissions)
  - start/stop/update instances as required (at minimum, the permissions needed to run your `--command`)

If you tell me how you authenticated (ADC from user login vs different flow), I can suggest the least-privilege IAM roles.

Note: Woodpecker pipeline containers cannot directly read `~/.config/...` from your workstation. You must provide that file contents via a Woodpecker secret (as above), or mount it into the agent container (advanced/less ideal).

For user ADC (`gcloud auth application-default login`), this pipeline uses:

1. `GOOGLE_APPLICATION_CREDENTIALS` -> ADC JSON file
2. `gcloud auth application-default print-access-token`
3. `gcloud --access-token-file=... compute ssh ...`

This avoids interactive `gcloud auth login` in CI.

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
GO111MODULE=off go test ./test/e2e -v -run TestEndpointServesTraffic
```

Because nothing is deployed at `34.60.141.247:80` yet, this test is currently expected to fail.

## Useful commands

```bash
./ci/deploy_local_woodpecker.sh status
./ci/deploy_local_woodpecker.sh logs
./ci/deploy_local_woodpecker.sh down
```
