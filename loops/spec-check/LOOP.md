---
id: spec-check                   # the folder name under loops/
title: "Spec check: the spec stays whole, and every test obligation has an example"
type: spec-check            # from the plugin's loop-types/index.yaml, or custom
status: active
trigger: [ci]                     # edit | local | commit | ci | preview | staging | production | scheduled | task | sprint | continuous
places: [delivery/oracle]                      # where it stands in the super-loops, as <super-loop>/<phase>: delivery/confirmation
run: ./run                      # one command; exit 0 = passes, non-zero = fires; machine-readable output
budget_seconds: 10              # one run must finish within this time
validation: cases               # cases (cases/must-fire, cases/must-pass) or manual (say why below)
agents: []                      # agents whose execution is a step of this loop, e.g. .claude/agents/reviewer.md
parts: []                       # composite loop only: ids of the loops under loops/ it is made of
hitl: []                        # ids from loops/policy.yaml this loop may call a person under
wiring: [.github/workflows/ci.yml job acceptance]                      # where the loop is hooked in, e.g. .claude/settings.json hooks.PostToolUse
---

# Spec check: the spec stays whole, and every test obligation has an example

## Requirement

The acceptance tests of TEAMS-1 are derived from the spec, so the spec has to stay whole: no field from nowhere, no
error or screen field without an example, no element that names a requirement nobody wrote, no approved
requirement the spec does not cover. Asked for with the delivery of TEAMS-1 (2026-11-10). Its place is
`delivery/oracle`, and it is a part of the acceptance loop.

## Loop

```
push or pull request ──► CI job acceptance ──► ./run .
   ops/spec specs check            requirements, layers, the trace from requirements to the spec
   ops/spec em check --strict      the event model, and every test obligation pinned down by an example
   ──► exit 1, one line per problem: "specs: …" or "model: …"
```

## Tools

- The spec plugin 0.3.1 (github.com/gutnikov/my-claude-plugins at 0e3c71d), through `ops/spec`: `specs.py check` and
  `em.py check --strict`. Python 3 with PyYAML 6.0.3.

## Signal

```
model: [S01] event TeamCreated.owner: source 'cmd' but command CreateTeam has no field 'owner'
model: no example pins down the error S01 TeamNameReserved
specs: specs/model/teams.em.yaml: slice S01 refers to unknown requirement TEAMS-1#9
```

## Validation

| Case | Expected | What it stands for |
|---|---|---|
| must-fire/field-from-nowhere | fires | An event field whose source the command does not have |
| must-fire/error-without-example | fires | An error of a command that no example pins down |
| must-fire/unknown-requirement | fires | A slice that names a requirement nobody wrote |
| must-pass/approved-spec | passes | The spec of TEAMS-1 as it was approved |

## Wiring

- `.github/workflows/ci.yml`, job `acceptance`, step `spec-check`: on every push to `main` and every pull request,
  with the spec plugin checked out at the pinned commit (`SPEC_PLUGIN`). It runs as the base branch has it.
- Locally: `loops/spec-check/run .`, with the plugin as the sibling checkout `../my-claude-plugins`.
