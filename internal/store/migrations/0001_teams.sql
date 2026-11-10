-- Teams (TEAMS-1). name_key is the name in lower case: two names that differ only in case are the same name.
-- seq orders the teams newest first.
CREATE TABLE teams (
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    id         TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    name_key   TEXT    NOT NULL UNIQUE,
    created_at TEXT    NOT NULL
);
