# gh-storage

GitHub-backed file storage API, written in Go and deployed on Vercel Fluid compute.

Production: https://gh-storage.vercel.app

## Development and deployment

Use Go 1.27.1. Set `ME`, `GITHUB_TOKEN`, `GITHUB_ACCOUNT`, and `GITHUB_REPO`
in `.env`. The GitHub token needs read/write access to the storage repository's
contents. `PORT` defaults to `8000`.

- `just dev` loads `.env` and runs the server locally.
- `just test` runs tests with the race detector.
- `just build` builds `exe`.
- `just clean` removes `exe` and the temporary build directory `tmp/`.
- `just deploy` deploys the linked Vercel project to production.

Set the four required environment variables in Vercel before deployment.
Deployments use the Vercel CLI; automatic GitHub deployments are not connected.
Local `.env` files are excluded from both Git and deployment uploads.
Vercel runs the root `main.go` using the Go framework preset with Fluid compute
in London (`lhr1`). Application packages and tests live under `internal/`.

## API

All `/data/` and `/raw/` requests require the `ME` header matching the configured
secret. File paths may include subdirectories.

| Method | Route | Response |
| --- | --- | --- |
| GET | `/` | `ha?` |
| GET | `/health` | JSON version |
| GET | `/status` | README read timings in milliseconds, comparison result, first line |
| GET | `/data/:path` | File text; any nonempty `binary` query value returns bytes |
| GET | `/raw/:path` | Text from the repository's `main` branch via raw GitHub |
| HEAD | `/data/:path` | 200 if the file exists; otherwise 404 |
| OPTIONS | `/data/:path` | GitHub file metadata without content |
| POST | `/data/:path` | Create a file from the UTF-8 request body; `created` |
| PUT | `/data/:path` | Update an existing file; `updated` |
| DELETE | `/data/:path` | Delete an existing file; `deleted` |

Contents API operations use the repository's default branch. Mutations create
GitHub commits. Failed reads return 404; failed mutations return 500. Incorrect
or missing authentication returns 403. The public `/status` endpoint reads
`README.md` and exposes its first line, matching the original service.

The original Deno deployment was at `https://ghdb.ea.deno.net`. Existing clients
should switch their base URL to `https://gh-storage.vercel.app`, keeping their
paths and authentication header. The Deno configuration and TypeScript sources
are retained unchanged for reference and excluded from Vercel uploads.
