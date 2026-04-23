# Copilot Instructions — HomeLog

HomeLog is a self-hosted family expense & utilities manager. Stack: **Go 1.25 / Gin / GORM / SQLite** backend, **Vue 3 (Composition API) / Vite / Tailwind / Pinia** frontend. Target: Raspberry Pi 3B+ via Docker (256 MB RAM limit).

---

## Go — avoid false positives

- **Go version is 1.25.** `for i := range n` (integer range) is valid Go 1.22+ syntax — do not flag it.
- **`go 1.25.0` in `go.mod` is valid.** Patch-version `go` directives are intentional since Go 1.21 (toolchain selection). Do not suggest removing the patch version.
- **`gorm:"-"` fields may be populated in handlers**, not in the model itself. Check the handler before flagging a `gorm:"-"` field as always-zero (e.g. `IsCurrencyLocked` is set in list/get/update handlers via `isUtilityCurrencyLocked()`).

## Go — security-critical patterns to enforce

Every DB query touching user data **must** be scoped. Flag any handler that:
- fetches rows without filtering by `userID` (from `middleware.GetUserID(c)`)
- fetches property-scoped data without verifying the user owns or is a member of that property
- creates `ExpenseSplit` records where the payer's own split is not `IsSettled=true` + `SettledAt=now` (an unsettled self-split is orphaned forever and breaks the "fully settled" check)

REST route contract: when a handler has both a parent `:id` and a child `:childId` in the path (e.g. `/utilities/:id/readings/:readingId`), it must verify the child's foreign key matches the parent `:id` before proceeding — not just check access to the child in isolation.

## Go — FTS5 search scoping

`GET /api/v1/search` scopes results to properties the user owns or is a member of, plus `property_id=0` rows scoped by `user_id` (legacy unscoped data). Users with no property memberships still see their own `property_id=0` records — they do not get a completely empty result.

## Vue — composition API rules

- **`<script setup>` only** — no Options API, no `export` statements inside SFC scripts.
- **`onMounted` alone is not enough for keep-alive views.** `App.vue` wraps Dashboard, Expenses, Utilities, Projects, and Settings in `<keep-alive>`. Query-only navigations (e.g. `?highlight=3` → `?highlight=5`) do not remount — use `watch(() => route.query.foo, ...)` for anything that must react to query changes. Flag any code in these views that reads `route.query` only in `onMounted` or `setup()` top-level without a corresponding watcher.
- **`onActivated`** must re-apply `route.query.property` overrides (global-search deep-links) before falling back to the previously selected property.
- **`watch(src, cb, { immediate: true })` + `const stop = …`** is a TDZ trap — Vue fires the callback synchronously before the `const` is assigned. Declare `stop` as `let` and guard with `stop?.()`.

## Vue — formatting

**Never hardcode `'it-IT'` or `'EUR'`** anywhere in the frontend. All user-facing numbers, currencies, and dates must go through `dateFormatter.js` (`formatNumber`, `formatCurrency`, `formatDate`, `formatDiff`, `formatPeriodCompact`) with `settingsStore.formatSettings` / `settingsStore.dateSettings`.

## Vue — highlight / deep-link composable

`useHighlight.js` uses a `handledId` guard to prevent re-triggering on unrelated source mutations. The route watcher resets `handledId` when the highlight changes — this is intentional (unblocks re-triggering for the same id after navigation). Do not flag it as unnecessary.

## SQLite / GORM

- Complex data (extraction rules, split member IDs) is stored as JSON strings in TEXT columns. Marshal/unmarshal explicitly — GORM does not do this automatically.
- FTS5 virtual tables do not support `ALTER TABLE ADD COLUMN`. Schema changes require DROP + recreate, which is handled by bumping `searchIndexVersion` in `database/search_index.go`.

## Bill / utility domain rules

- `Utility.IsDomiciled` (direct debit) and `Utility.IsInstallmentBased` (installment billing) are independent flags.
- `PATCH /bills/:id` with `is_paid:true` auto-creates an expense via `autoCreateExpenseFromBill()`. The payer is `utility.PaidByMemberID`, not the logged-in user.
- Per-service split override (`utility.split_override`): `""` = global household split, `"no_split"` = never split, `"custom"` = use `split_member_ids`.
