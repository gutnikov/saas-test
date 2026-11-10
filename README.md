# saas-test

A small SaaS for team subscriptions (teams, plans, trials, seats), made to try agent super-loops on: delivery,
remediation, maintenance and discovery, built by the `harness` Claude Code plugin. The skeleton has no product
features yet; they come through the delivery super-loop.

## What is here

| Path | What |
|---|---|
| `cmd/server` | The service: one Go binary with the HTTP API and the built frontend |
| `cmd/traffic` | Synthetic users that send requests to a running server |
| `internal/httpapi` | Routes, the frontend, the JSON request log |
| `internal/store` | The SQLite database and its migrations (`internal/store/migrations/`) |
| `web/` | The React app (Vite, TypeScript), embedded into the server from `web/dist` |
| `ops/prod` | The local production: deploy, rollback, start, stop, logs, traffic |
| `.github/workflows/ci.yml` | CI: `make ci` on every push to `main` and every pull request |

## Develop

Go 1.26 and Node 26 (`.nvmrc`; with nvm, `nvm use`).

```
make ci          # what CI runs: the frontend build, gofmt, go vet, go test, the binaries
make dev-server  # the server on :8080
make dev-web     # the Vite dev server on :5173, proxying /api and /healthz to :8080
```

## Production

Production runs on this machine, outside the repository, in `~/.saas-test/prod`, and listens on `127.0.0.1:18080`.

```
ops/prod deploy            # build origin/main and make it the running release
ops/prod deploy <ref>      # the same for any git ref
ops/prod rollback          # back to the previous release
ops/prod status            # the server, its release and /healthz; the traffic
ops/prod logs 50           # the last lines of the server's log
ops/prod traffic start     # synthetic users; ops/prod traffic stop
```

- A deploy builds the ref in a temporary worktree, starts it, and waits for `/healthz` to answer with the new
  version. If it does not, the previous release goes back and the deploy fails.
- Every log line is JSON and carries `version`; a response with a 5xx status and a panic are `"level":"ERROR"`.
- `deploys.jsonl` records each deploy and rollback with its result.
