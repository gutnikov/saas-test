# release-confirmation — changes

- 2026-11-10: proposed.
- 2026-11-10: built: run judges a snapshot of the window, snapshot waits for it and saves it; wired into ops/prod deploy and confirm, with a rollback on not-confirmed.
- 2026-11-10: validated on 10 recorded windows; triggered for real on two scratch releases: confirmed the good one, caught the broken one and rolled it back.
- 2026-11-10: active: committed to main, wired into ops/prod.
- 2026-11-10: confirmed TEAMS-1 in production with its spec signals: 185 TeamCreated, 167 views of the list, 0 errors in 20 min.
