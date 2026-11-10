---
id: acceptance
title: "Acceptance loop: every approved requirement goes from red to green, its spec and tests untouched"
type: acceptance-loop
status: active
trigger: [ci]
places: [delivery/oracle, delivery/gate]
run: ./run
budget_seconds: 10
validation: cases
agents: []
parts: [spec-check]
hitl: []
wiring: [.github/workflows/ci.yml job acceptance]
---

# Acceptance loop: every approved requirement goes from red to green, its spec and tests untouched

## Requirement

The user wants a change delivered against a spec they approved, with acceptance tests that a separate agent derives
from it (asked with TEAMS-1, 2026-11-10). This loop is the gate of the delivery super-loop: its places are
`delivery/oracle` and `delivery/gate`. Delivery itself ends later, at the release's confirmation
(`release-confirmation`).

## Loop

```
approved spec (specs/) ──► a test writer, not the implementing agent, writes acceptance/*_test.go, one per example
   ──► the tests are red ──► collect REPO --record: red-before.json and oracle.json (the fingerprints of the spec
       and the tests)
   ──► the work ──► CI job acceptance: spec-check, then collect + run:
       oracle: changed without approval?   every approved requirement: no test | green before the work | red | green
   ──► exit 0 only when the oracle is untouched and every approved requirement is green
```

A change to an example or to a test goes through the spec and a new approval, then `collect REPO --record` again.

## Tools

- Python 3 (stdlib for `run`; PyYAML for `collect`), Go 1.26 (`go test -json`), the spec plugin through `ops/spec`
  (`em.py export`).

## Signal

```
oracle: changed without approval — acceptance/teams_test.go
TEAMS-1#1: red — TestS01ANewNameCreatesATeam
TEAMS-1#2: green (2 tests)
TEAMS-1#3: no test — S01/a name of 61 characters is refused
```

## Validation

| Case | Expected | What it stands for |
|---|---|---|
| must-fire/example-without-test | fires | An example of an approved requirement has no test |
| must-fire/green-before-the-work | fires | A test passed before the work, so it tests nothing |
| must-fire/red-now | fires | A test fails after the work |
| must-fire/oracle-changed | fires | A test file differs from the one recorded red |
| must-fire/no-record-before-the-work | fires | Nobody recorded the run before the work |
| must-fire/tag-not-in-the-spec | fires | A test is tagged with an example the spec does not have |
| must-fire/tests-do-not-build | fires | The acceptance tests do not compile |
| must-pass/all-green | passes | Red before, green now, nothing changed; a draft requirement is ignored |

## Wiring

- `.github/workflows/ci.yml`, job `acceptance`, step `acceptance gate`: `collect . $RUNNER_TEMP/acceptance` and `run`
  on it, on every push to `main` and every pull request, after `spec-check`.
- `red-before.json` and `oracle.json` are recorded locally when the tests are written and red, and committed with
  them. The implementing agent may not change `specs/` or `acceptance/`; the gate in CI tells when it did.
