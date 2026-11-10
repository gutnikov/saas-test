---
id: release-confirmation
title: "Release confirmation: the change is there, it happened, nothing broke"
type: release-confirmation
status: active
trigger: [production]
run: ./run
budget_seconds: 10
validation: cases
agents: []
parts: []
hitl: []
wiring: [ops/prod deploy, ops/prod confirm]
places: [delivery/confirmation, remediation/confirmation, maintenance/confirmation]
---

# Release confirmation: the change is there, it happened, nothing broke

## Requirement

A release counts as done once `/healthz` answers with its version, and nothing looks at production after that. The
user wants delivery to end on a fact confirmed in production instead: the release running, and the log clean for a
window such as 20 minutes (asked on 2026-11-10). Its place is `delivery/confirmation`, and the same phase of
remediation and maintenance.

## Loop

```
ops/prod deploy ──► /healthz answers the new version ──► ops/prod confirm, in the background or with --wait
  snapshot: waits for at least 20 min and 50 requests of the release, at most 60 min; saves the release's log
            and the previous release's log over as long a window before the deploy
  ./run judges the snapshot ──► confirmed: done
                            ├─► not-confirmed: ops/prod rollback; the snapshot is the evidence
                            └─► inconclusive: too few requests; a longer window, or a person decides
```

Every verdict goes into `~/.saas-test/prod/deploys.jsonl` as `"action":"confirm"`, with the path of its snapshot. It
is the trace of the super-loops that the harness level reads.

## Tools

- Python 3.9.6, the system `python3` of macOS, standard library only: `run` and `snapshot`.
- `ops/prod` (bash, compatible with 3.2) and `curl`. The server's JSON log carries `version` on every line.

## Signal

`./run <snapshot>` prints one line per check and the result, and exits 0 when confirmed, 1 when not confirmed and 3
when inconclusive:

```
there: ok — /healthz answers bbbbbbb
window: ok — 21.0 min, 123 requests
happened: ok — serves requests, 123 (min 50)
broke: new error — http GET /api/version 500 (3)
broke: new error — panic: version lookup failed (3)
broke: error rate — 2.44% of requests got a 5xx, over the limit 0.50% (previous 0.00%)
broke: latency ok — p95 3 ms vs 3 ms before
result: not-confirmed bbbbbbb
```

A release names the signals it must show in `ops/release-signals.json`, such as
`[{"name": "team created", "match": {"msg": "team created"}, "min": 10}]`. Every release must also serve the
window's requests: the signal `serves requests`.

## Validation

Recorded windows in `cases/`:

| Case | Expected | What it stands for |
|---|---|---|
| must-fire/new-error | not confirmed | An error the previous release never had |
| must-fire/crash-outside-handlers | not confirmed | A crash that prints plain text, not JSON |
| must-fire/error-rate-up | not confirmed | A known error, six times as often |
| must-fire/latency-up | not confirmed | p95 of 400 ms against 3 ms |
| must-fire/old-release-running | not confirmed | `/healthz` still answers the previous version |
| must-fire/release-signal-missing | not confirmed | The release's own event appeared 3 times of the 10 it needs |
| must-fire/too-few-events | inconclusive | 12 requests in an hour: nothing to judge on |
| must-pass/clean | confirmed | Enough requests, no errors |
| must-pass/background-errors | confirmed | The same errors at the same rate as before |
| must-pass/release-signal-appeared | confirmed | The release's own event appeared 15 times of the 10 it needs |

`loops.py validate`: all 10 cases as expected, the same verdict on every run, 0.04–0.06 s each.

Triggered for real on 2026-11-10, with a window of 2 minutes and 30 requests, on two scratch releases made from main
and deleted afterwards:

- `8598a5c`, the same code as main: confirmed, 65 requests and no errors.
- `2432236`, where `/api/version` panics on one request in four and the unit tests stay green: not confirmed, with
  two new errors and a 5xx rate of 6.25% against 0%. `ops/prod` rolled it back to `8598a5c` by itself.
- `6e28de3`, main again: confirmed on a weak baseline, since the release before it had served only 7 requests, so
  the check fell back to the 1% limit.

The snapshots and verdicts are in `~/.saas-test/prod/confirmations/`, and the five entries in `deploys.jsonl`.

## Wiring

- `ops/prod deploy [REF] [--wait]`: once `/healthz` answers the new version, it runs `ops/prod confirm`, in the
  background (`ops/prod status` and `logs/confirm.log` show how far it got) or in the foreground with `--wait`, which
  then exits with the verdict's code.
- `ops/prod confirm`: `snapshot`, then `run`; records the verdict in `deploys.jsonl`, and rolls back on
  `not-confirmed`. `SAAS_CONFIRM_MINUTES`, `SAAS_CONFIRM_EVENTS` and `SAAS_CONFIRM_CAP` set the window.
- It runs in the deploy, not in the agent's session. In this local production the agent can still reach the log and
  `ops/prod`: a known limit of the test setup the user chose.
