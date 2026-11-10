# Migrations

Each file `NNNN_<what>.sql` changes the schema once. `store.Open` applies the files in name order, each in its
own transaction, and records it in `schema_migrations`; a failed migration is rolled back whole and the server does
not start.

- Never edit a migration that has been deployed: add a new one.
- One change per file, numbered after the last one: `0001_teams.sql`, `0002_plans.sql`.
