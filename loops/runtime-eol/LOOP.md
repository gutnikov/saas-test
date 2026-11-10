---
id: runtime-eol                   # the folder name under loops/
title: "Runtime end of life: the runtimes stay supported, or someone hears in time"
type: dependency-updates            # from the plugin's loop-types/index.yaml, or custom
status: validated
trigger: [scheduled]                     # edit | local | commit | ci | preview | staging | production | scheduled | task | sprint | continuous
places: [maintenance/entry]                      # where it stands in the super-loops, as <super-loop>/<phase>: delivery/confirmation
run: ./run                      # one command; exit 0 = passes, non-zero = fires; machine-readable output
budget_seconds: 10              # one run must finish within this time
validation: cases               # cases (cases/must-fire, cases/must-pass) or manual (say why below)
agents: []                      # agents whose execution is a step of this loop, e.g. .claude/agents/reviewer.md
parts: []                       # composite loop only: ids of the loops under loops/ it is made of
hitl: []                        # ids from loops/policy.yaml this loop may call a person under
wiring: [.github/workflows/runtime-eol.yml]                      # where the loop is hooked in, e.g. .claude/settings.json hooks.PostToolUse
---

# Runtime end of life: the runtimes stay supported, or someone hears in time

## Requirement

Node 20, which builds the frontend, reached its end of life on 2026-04-30, and nobody noticed for six months:
nothing watched the runtimes' support (an escape at `maintenance/entry`, in `loops/trace.jsonl`). Found while the
user chose maintenance as the next super-loop, 2026-11-10. Its place is `maintenance/entry`.

## Loop

```
every Monday ──► .github/workflows/runtime-eol.yml ──► collect . OUT: Node from .nvmrc (or CI), Go from go.mod,
                                                        their end of life from endoflife.date
                                                   ──► run OUT: past its end of life, or less than 90 days left?
   ──► exit 1: the scheduled run fails, GitHub tells the owner, and `gh run list --workflow runtime-eol` shows it
       to the agent: a trigger for maintenance
```

The days of support left are also the measure that `behaviour-invariance` reads.

## Tools

- Python 3 (standard library), the endoflife.date API, GitHub Actions on a schedule.

## Signal

```
node 20 (.github/workflows/ci.yml): past its end of life on 2026-04-30, 194 days ago — move to a supported version (latest of this line: 20.20.2)
go 1.26 (go.mod): supported, no end of life announced; latest 1.26.10
```

## Validation

| Case | Expected | What it stands for |
|---|---|---|
| must-fire/node-past-its-end-of-life | fires | Node 20 in November 2026 |
| must-fire/node-near-its-end-of-life | fires | 60 days before the end of Node 22 |
| must-fire/unknown-version | fires | A version endoflife.date does not know |
| must-pass/supported | passes | Node 26 and Go 1.26 in November 2026 |

## Wiring

- `.github/workflows/runtime-eol.yml`: on a schedule (Mondays, 06:17 UTC) and by hand (`workflow_dispatch`).
