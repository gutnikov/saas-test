---
id: behaviour-invariance                   # the folder name under loops/
title: "Behaviour invariance: maintenance changes no expected value, and improves its measure"
type: behaviour-invariance            # from the plugin's loop-types/index.yaml, or custom
status: validated
trigger: [ci]                     # edit | local | commit | ci | preview | staging | production | scheduled | task | sprint | continuous
places: [maintenance/oracle, maintenance/gate]                      # where it stands in the super-loops, as <super-loop>/<phase>: delivery/confirmation
run: ./run                      # one command; exit 0 = passes, non-zero = fires; machine-readable output
budget_seconds: 10              # one run must finish within this time
validation: cases               # cases (cases/must-fire, cases/must-pass) or manual (say why below)
agents: []                      # agents whose execution is a step of this loop, e.g. .claude/agents/reviewer.md
parts: []                       # composite loop only: ids of the loops under loops/ it is made of
hitl: []                        # ids from loops/policy.yaml this loop may call a person under
wiring: [.github/workflows/ci.yml job maintenance]                      # where the loop is hooked in, e.g. .claude/settings.json hooks.PostToolUse
---

# Behaviour invariance: maintenance changes no expected value, and improves its measure

## Requirement

Maintenance runs without a person approving any behaviour, so its gate has to check the behaviour itself: no test,
acceptance test, spec or golden file may change, and the measure the change is for has to improve. Built with the
first maintenance of saas-test, moving the build off Node 20 (2026-11-10). Its places are `maintenance/oracle` and
`maintenance/gate`.

## Loop

```
pull request from a maint/ branch ──► CI job maintenance ──► collect . origin/main OUT:
    the files BASE...HEAD touches, and the days of support left of the runtimes at BASE and at HEAD (runtime-eol)
──► run OUT:
    a test, an acceptance test, the spec or a golden file changed?  ──► fires: not maintenance, back to triage
    the measure did not improve?                                     ──► fires
    otherwise                                                        ──► invariant, measure improved
```

## Tools

- Python 3 (standard library), git (the history down to the base: `fetch-depth: 0`), `loops/runtime-eol/collect`.

## Signal

```
expected value changed: internal/store/store_test.go — this is not maintenance; back to triage
measure improved — days of support left of the runtimes: node 20: -194 days; go 1.26: no end of life announced → node 26: 902 days; go 1.26: no end of life announced
```

## Validation

| Case | Expected | What it stands for |
|---|---|---|
| must-fire/a-test-changed | fires | A Go test changed along with the toolchain |
| must-fire/an-acceptance-test-changed | fires | An acceptance test changed |
| must-fire/the-spec-changed | fires | The spec changed |
| must-fire/a-web-test-changed | fires | A frontend test changed |
| must-fire/measure-not-improved | fires | The runtimes' support stayed where it was |
| must-pass/runtime-moved | passes | Only the toolchain files changed, and Node moved from 20 to 26 |

The measure is the runtimes' support only: a maintenance of another kind (a refactoring, a dependency bump) needs its
own measure here first.

## Wiring

- `.github/workflows/ci.yml`, job `maintenance`: on a pull request whose branch starts with `maint/`, against its base.
