# Acceptance tests

The acceptance tests check the approved spec in `specs/` through the HTTP API: one test per example of the event
model, tagged with the example's id. An agent that does not implement the spec writes them from it, and they fail
before the work.

## The seam

How a test reaches the app. It was decided before the tests and the code, and both keep to it.

- **The app**: `httpapi.New(httpapi.Config{…})` on a database from `store.Open` in a temporary directory, served with
  `net/http/httptest`. Every test gets a fresh app. Give the config a JSON logger (`slog.NewJSONHandler`) that writes
  to a buffer the test can read.
- **The states**: `noTeams` is a fresh app. `teamAcme` is a fresh app in which `POST /api/teams` created a team named
  `Acme`.
- **The command CreateTeam**: `POST /api/teams` with the JSON body `{"name": "<name>"}`.
  - Created: status 201 and `{"id": "<id>", "name": "<the name, trimmed>"}`.
  - Refused: status 400 for `TeamNameRequired` and `TeamNameTooLong`, status 409 for `TeamNameTaken`, and the body
    `{"error": "<the error>", "message": "<the text>"}` with the texts of TEAMS-1#4.
- **The read model Teams on TeamsPage**: `GET /api/teams` answers 200 and `[{"id": "<id>", "name": "<name>"}, …]`,
  newest first. A `see` of `Teams.name` compares the list of names, in that order.
- **The event TeamCreated**: the app logs it as a JSON line with `"msg":"domain event"`, `"event":"TeamCreated"`,
  `"teamId"` and `"name"`. A test that expects the event finds that line in the logger's buffer. The same line is
  the production signal of TEAMS-1#1.
- **Tags**: the line before each test function is `// example: <slice id>/<example name>`, with the example's name
  exactly as the spec has it. The acceptance loop maps the tests to the requirements through these tags.
