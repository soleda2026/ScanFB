# Watched Group Persistence Decision

## Status

APPROVE, implemented by Phase 9E2.

Phase 9E2a was documentation-only. Phase 9E2 now implements the approved
database, runtime path resolver, bridge authority and Swift presentation flow
without adding migration execution or Swift persistence.

## Blocker Context

Phase 9E2 stopped because ScanFB had no approved production storage location
and no implemented migration runner. The existing SQLite schema version 1 is
owned exclusively by completed `BatchRecord` snapshots, while Phase 9E1 keeps
WatchedGroups and the Phase 9C cursor only in Swift session memory.

The authoritative macOS app bundle identifier is `com.soleda.ScanFB`.

## Location Candidates

| Criterion | A. One SQLite DB | B. Separate SQLite DBs | C. SQLite + state file | D. Existing alternative |
| --- | --- | --- | --- | --- |
| Ownership | Go | Go | Go, plus a second codec | None exists |
| Path derivation | One Application Support file | One Application Support directory, two fixed files | One database and one file under Application Support | Not applicable |
| Sandbox compatibility | Compatible logical root | Compatible logical root and isolated files | Compatible root, but two file behaviors | Not applicable |
| Backup/user data | One user-data file | Two ordinary Application Support user-data files | Mixed SQLite/file backup behavior | Not applicable |
| Atomicity | Per-file only; unrelated aggregates gain no useful shared transaction | Groups and cursor share one transaction; no cross-store invariant | Requires atomic file replacement plus SQLite transactions | Not applicable |
| Versioning complexity | Requires completed-batch v2 now | Independent schema v1 stores | Separate SQLite and file format versions | Not applicable |
| Bridge exposure | Helper can keep path internal | Helper keeps both paths internal | Helper keeps paths internal but owns two mechanisms | Not applicable |
| Testability | Requires migration fixtures | Explicit temporary database paths | Temporary DB plus file/replacement tests | Not applicable |
| Recovery/fail-closed | One corruption blast radius | Store-specific validation and failure | Custom malformed/truncated-file recovery | Not applicable |
| Future Phase 11 | One file, but migration precedes use | Completed snapshots and mutable group state coexist without redesign | Phase 11 spans mixed storage mechanisms | Not applicable |
| Swift path visibility | Not required | Not required | Not required | Not applicable |
| Consistency risk | Encourages accidental coupling | Two aggregates remain intentionally separate | Dual-format behavior can diverge | Not applicable |

### A. One application SQLite database

- Ownership and atomicity are simple inside one file, and one Application
  Support path is sandbox-compatible.
- Adding WatchedGroup state would require completed-batch schema v1 to migrate
  to v2 before migration execution exists.
- A malformed state table could increase the corruption blast radius for
  completed batch history.
- A unified file would not provide useful cross-domain atomicity because
  WatchedGroup configuration and completed snapshots are separate aggregates.

Decision: rejected for Phase 9E2 because it forces migration work only to keep
one file.

### B. Separate application SQLite databases

- Go owns both stores and derives both paths from the same macOS Application
  Support root.
- The completed-batch database remains schema v1 unchanged. WatchedGroup state
  starts with an independent schema v1.
- Groups and cursor share one state database and transaction. There is no
  cross-database invariant requiring a distributed transaction.
- Corruption and schema evolution have a smaller blast radius, while both
  stores remain compatible with future Phase 11 orchestration.
- Tests can open each repository at an explicit temporary path.

Decision: selected.

### C. SQLite plus a Go-owned local state file

- This avoids changing the completed-batch schema but introduces a second
  serialization, validation, atomic-replacement and versioning mechanism.
- A plain state file provides weaker relational constraints and transaction
  semantics for ordered groups plus a cursor.
- Recovery and permission behavior would need separate implementation and
  tests despite SQLite already being an approved dependency.

Decision: rejected as unnecessary storage-format duplication.

### D. Existing repository alternative

No other production location or local-state mechanism exists in the
repository. `UserDefaults`, `@AppStorage`, Swift JSON storage, cloud storage and
network storage are outside the approved architecture.

## Schema Evolution Candidates

| Criterion | A. Existing DB v2 | B. Separate state DB v1 | C. Independent logical schemas in one DB | D. Other |
| --- | --- | --- | --- | --- |
| Migration complexity | Requires a new runner before Phase 9E2 | No migration of existing data | Still requires coordinated bootstrap/version logic | None exists |
| Batch snapshot risk | Existing user history is in migration blast radius | Existing completed-batch v1 is untouched | Shared-file corruption/version risk remains | Not applicable |
| Rollback/fail-closed | Needs transactional v1-to-v2 rollback | Empty v1 bootstrap; strict open validation | Multiple logical versions must fail together coherently | Not applicable |
| Transactionality | One database transaction | Groups and cursor transact together in state DB | One file, but version domains are coupled | Not applicable |
| Repository compatibility | Broadens current batch repository bootstrap | Adds a narrow independent repository | Requires redesign of current strict inventory | Not applicable |
| Future evolution | Sequential migrations for all data | Each database evolves independently | Coordinated logical versions add policy surface | Not applicable |
| Corruption blast radius | Broad | State-specific | Broad | Not applicable |
| Testability | Requires populated v1 migration fixtures | Temporary empty/state database fixtures | Requires mixed-version matrix | Not applicable |
| Redesign burden | High for current milestone | Low | High | Not applicable |

### A. Existing database v1 to v2

Rejected. It requires a migration runner and risks completed batch snapshots.
The current repository intentionally supports schema v1 bootstrap and strict
validation only.

### B. Separate WatchedGroup database v1

Selected. The completed-batch schema and repository remain unchanged. Phase
9E2 introduces only a dedicated WatchedGroup-state repository and its empty
schema version 1.

### C. One database with independently versioned logical schemas

Rejected. SQLite still has one physical file and one bootstrap/validation
boundary. Multiple logical version rows would add coordination complexity
without removing the need to evolve the existing database safely.

### D. Other schema mechanism

No repository-consistent alternative was found. No ORM, generic metadata store
or migration framework will be introduced by this decision.

## Selected Architecture

ScanFB uses separate local SQLite databases under one application-owned
directory:

```text
<user-application-support>/com.soleda.ScanFB/completed-batches.sqlite3
<user-application-support>/com.soleda.ScanFB/watched-groups.sqlite3
```

`<user-application-support>` means the current user's standard macOS
Application Support location returned by the operating system. It is not a
literal path, repository path, DerivedData path or hardcoded home directory.
Under App Sandbox, the OS-provided user Application Support root may be
containerized; the relative `com.soleda.ScanFB` directory and filenames remain
the logical policy.

The existing bundle identifier `com.soleda.ScanFB` is the stable application
directory component. A future bundle-identifier change requires an explicit
storage-location migration decision.

## Path And Store Ownership

- Go owns production path derivation, directory creation, database opening,
  schema validation and persistence behavior.
- The packaged bridge helper resolves the production path internally. No raw
  path is accepted from or returned to Swift.
- Swift never opens SQLite and never receives a database-local identifier.
- The CLI remains unchanged and does not open either production database.
- Repository constructors retain explicit-path support for deterministic tests.
  Tests use temporary directories and never use production Application Support.
- The Go production resolver creates the application directory before
  opening the database. Failure to resolve or create it is an explicit storage
  error and must not be presented as an empty group list.
- The directory and database are owned by the current user. Creation must use
  owner-only permissions: directory `0700`, database `0600`; any SQLite sidecar
  files must remain inside the same owner-only directory.
- No cloud container, network volume, sync service or backup service is part of
  the implementation. As ordinary Application Support data, files may be
  included in user/system backup policy, but ScanFB adds no backup behavior.

## WatchedGroup-State Schema Version 1

The dedicated database persists only:

- ordered WatchedGroups;
- one Phase 9C collection-position cursor;
- its own singleton schema-version metadata.

Each WatchedGroup preserves exactly:

- local ID;
- optional Facebook group ID;
- optional canonical URL;
- name;
- createdAt;
- active state;
- notes;
- optional lastSuccessfulScanAt;
- lastError;
- displayOrder;
- explicit insertion position.

The row is source-neutral: persistence does not imply provenance. Phase 9E3b
one-time enrollment populates these same identity/state fields without a schema
change. Optional future discovery/synchronization may also use them only in a
separately approved milestone, without adding account, session, browser or
membership metadata to schema v1.

At least one authoritative source identity remains required by the Go domain.
Persistence does not generate IDs, infer identity, normalize stored values,
repair duplicates or recompute metadata.

The schema enforces one row per local ID, the representable authoritative
identity constraints, unique insertion positions, supported boolean values,
one schema metadata row and one cursor state row. Go restore must still rebuild
`WatchedGroupCollection` in insertion order so Phase 9B validation remains the
final authority, including cross-kind canonical conflicts.

Phase 9E2 implements tables `watched_group_schema_metadata`, `watched_groups`
and `watched_group_selection_state`, with explicit unique indexes for local ID,
Facebook ID and URL-only canonical identity. It uses finite SQLite DELETE
journal mode. The application directory is created/chmoded `0700`, the database
is chmoded `0600`, and transient sidecars remain inside that owner-only directory.

## Cursor Semantics

- Store the exact non-negative Phase 9C collection position as an integer in
  the same database as WatchedGroups.
- A new empty database stores the initial cursor `0`.
- Zero restored groups require cursor `0`.
- With groups present, the restored cursor must be less than the restored
  collection count.
- Negative, malformed or out-of-range values fail closed. They are never
  silently reduced modulo the collection size.
- An explicit selection advance persists the exact `NextCursor()` returned by
  Phase 9C in one transaction before success is returned.
- Preview/list reads never advance the cursor. In the intended product flow,
  future real batch execution owns the explicit advance; the cursor is not a
  user-managed queue control.
- An insufficient-active-groups result does not advance the cursor.
- Add and active-state changes preserve the current cursor and persist their
  mutations atomically. The Phase 9E2 scope has no delete operation, so those
  mutations cannot make an already-valid collection-position cursor invalid.

## Transaction And Failure Policy

- Empty-schema creation is fully transactional.
- Add, metadata update and active-state update each validate before mutation
  and commit in one transaction.
- Cursor advance commits in one transaction with the authoritative cursor
  state returned to the caller.
- Restore uses a consistent read transaction and returns no partial snapshot.
- Missing storage is initialized as an empty valid state. Inaccessible storage,
  malformed rows, duplicate identity, invalid ordering, bad timestamps,
  unsupported schema version or invalid cursor fails closed.
- ScanFB never silently deletes, recreates, repairs, truncates or downgrades an
  incompatible database. A storage-load failure must remain visible in the UI
  and must not be converted to a fabricated empty state.

The two databases intentionally do not share a transaction. Completed batch
snapshots and WatchedGroup configuration are separate aggregates; future work
must not create an invariant requiring atomic writes across both stores without
a separate architecture decision.

## Bridge Implications

Phase 9E2 keeps the existing narrow watched-group operation family. The
helper opens the dedicated state store internally for each one-request bridge
call. Bridge payloads contain typed group/action data only, never SQL, database
handles, database-local IDs or filesystem paths.

The Phase 9E2 bridge schema removes the Phase 9E1 Swift snapshot/cursor as
persistent authority. Restore and mutation responses return authoritative Go
state. `watched_groups_next_five` is the finite typed advance intent that
atomically persists Phase 9C `NextCursor()` without adding a generic command bus.

Storage errors use bounded typed bridge failures. Diagnostic output remains
bounded, redacted and stderr-only; no path or private group value is written to
diagnostics.

## Future Migration Policy

The completed-batch database and WatchedGroup-state database evolve
independently. Each has its own integer schema version and strict schema
inventory.

The initial Phase 9E2 implementation creates and validates only WatchedGroup
schema v1. It does not add a migration runner. Before either non-empty database
increments version, a separate migration milestone must define explicit,
ordered, sequential, one-way, fully transactional migrations and temporary
database tests. Migration failure rolls back completely; unsupported versions
fail closed; no database is silently recreated. Backup policy remains deferred.

## Implemented Boundary

Phase 9E2 implements the approved boundary:

1. Go production path resolver for the approved Application Support
   location, with explicit temporary-path injection for tests.
2. Dedicated WatchedGroup-state repository and schema v1 without changing
   completed-batch schema/runtime code.
3. Full Phase 9B value plus exact Phase 9C cursor persistence/restore with
   the validation and transactions above.
4. Existing watched-group bridge schema/handlers upgraded to load and
   mutate authoritative persistent state.
5. `WatchedGroupsStore` displays loading/storage failures and refreshes
   only from authoritative bridge responses.

## Non-Goals

- No completed-batch schema change or migration.
- No scan result, attempt, post, lead or lifecycle persistence in this store.
- No Phase 11 execution, scheduling, retry, worker or concurrency system.
- No Facebook acquisition, selector, cookie, credential, session or browser
  profile data.
- No Swift persistence, `UserDefaults`, `@AppStorage` or arbitrary JSON file.
- No cloud, sync, networking, encryption/key management, backup implementation,
  import/export, delete API or generic CRUD/SQL bridge.
