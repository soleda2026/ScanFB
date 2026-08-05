# Persistence Schema Design

Phase 5F defined the durable local SQLite storage model for completed ScanFB batch snapshots. Phase 5G1 implements only the SQLite foundation for schema version 1: explicit-path open/create, foreign-key enable/verify, transactional empty-schema creation, schema metadata validation, and close. Durable `SaveBatch`, load/list APIs, migrations, UI/CLI wiring, and production scan persistence remain deferred.

## 1. Purpose And Non-Goals

Purpose:

- Store one completed `persistence.BatchRecord` as the root aggregate.
- Preserve every historical decision, reason code, source post, lead, outcome, conflict, and summary value.
- Allow a future save/load adapter to reconstruct the same `BatchRecord` deterministically without rerunning buyer-intent classification, geography classification, deduplication, blocklist evaluation, aggregation, or summary calculation.
- Keep storage local-only and inside `internal/persistence`.

Non-goals:

- No durable `SaveBatch`, `LoadBatch`, `ListBatches`, update, delete, search, or paging API.
- No migration files, migration execution, production database location policy, UI/CLI wiring, or production scan persistence.
- No JSON blobs, generic metadata maps for business data, polymorphic `entity_type/entity_id` references, ORM naming assumptions, cloud sync, cross-device sync, multi-user tenancy, soft deletion, or audit logging.

## 2. Approved Dependency Boundary

- `internal/persistence` owns the SQLite schema-bootstrap implementation and remains the future home for durable SQLite save/load implementation.
- `SQLiteBatchRepository` opens and validates schema version 1 but intentionally does not implement `SaveBatch`.
- Future durable SQLite save adapters consume validated `persistence.BatchRecord` snapshots and implement the existing save-only `BatchRepository.SaveBatch(record)` contract.
- Storage must not own or recompute business decisions.
- `internal/application` does not import persistence.
- `internal/orchestration` may import application and persistence, and may pass completed snapshots to the repository.
- Domain, rules, dedup, blocklist, application, and orchestration contracts must not receive database-local IDs or SQLite-specific values.
- Durable storage remains local-only.

## 3. Root Aggregate Model

One row in `scan_batches` is the root aggregate for one completed batch snapshot. `batch_record_id` is caller-supplied, opaque, unique, and authoritative at the contract boundary.

Schema version 1 uses an internal integer surrogate primary key, `batch_pk`, to make child foreign keys compact and stable inside SQLite. `batch_pk` is database-local only, must never be exposed through `BatchRecordID`, and must never be required for public reconstruction. Reconstruction starts from `batch_record_id`, then loads children through the internal key.

The root stores:

- `batch_record_id`
- `schema_version`
- scan window start: `scan_start_of_day`
- scan window end: `scan_started_at`
- `scan_date`
- `scan_timezone`
- SearchProfile identity and enabled state
- `geographic_mode`
- all `BatchSummaryRecord` count fields

No random UUID generation, current-time generation, or SQLite-generated public ID is part of the design.

## 4. Table Inventory

| Table | Responsibility |
| --- | --- |
| `schema_metadata` | Single database metadata row with current schema version. |
| `scan_batches` | Root completed batch snapshot and batch summary counts. |
| `batch_search_profile_terms` | Ordered SearchProfile product, buyer-intent, and noise term snapshots. |
| `batch_groups` | Ordered group snapshots owned by one batch. |
| `raw_post_occurrences` | Exact raw post occurrence values and batch/group input positions. |
| `evaluated_posts` | Rule decision metadata for evaluated post records. |
| `evaluated_post_reasons` | Ordered rule reason codes for evaluated posts. |
| `bucketed_posts` | Ordered include/review/excluded collection membership. |
| `bucketed_post_reasons` | Ordered reason codes for bucket-specific evaluated post records. |
| `leads` | Aggregated lead identity, author key, and need identity. |
| `lead_key_need_product_evidence` | Ordered product evidence for a lead key need. |
| `lead_key_need_buyer_intent_evidence` | Ordered buyer-intent evidence for a lead key need. |
| `lead_need_product_evidence` | Ordered product evidence for a lead need. |
| `lead_need_buyer_intent_evidence` | Ordered buyer-intent evidence for a lead need. |
| `lead_sources` | Ordered source-post references for each lead. |
| `lead_outcomes` | Ordered allowed, blocked, and unresolved lead outcome records. |
| `lead_outcome_blocklist_reasons` | Ordered blocklist reason codes for lead outcomes. |
| `lead_outcome_application_reasons` | Ordered application reason codes for unresolved outcomes. |
| `unaggregated_posts` | Ordered unaggregated post records with candidate/source identity. |
| `unaggregated_candidate_product_evidence` | Ordered product evidence for unaggregated candidate need. |
| `unaggregated_candidate_buyer_intent_evidence` | Ordered buyer-intent evidence for unaggregated candidate need. |
| `unaggregated_post_reasons` | Ordered dedup reason codes for unaggregated posts. |
| `source_conflicts` | Ordered source conflict records and involved source identities. |
| `source_conflict_candidate_product_evidence` | Ordered product evidence for conflict candidate need. |
| `source_conflict_candidate_buyer_intent_evidence` | Ordered buyer-intent evidence for conflict candidate need. |
| `source_conflict_reasons` | Ordered dedup reason codes for source conflicts. |
| `group_summaries` | Ordered per-group summary snapshot values. |

## 5. Columns For Each Table

Column lists are schema contracts. Executable DDL lives in `internal/persistence` and must stay aligned with this document.

### `schema_metadata`

| Column | Affinity | Notes |
| --- | --- | --- |
| `metadata_pk` | INTEGER | Internal fixed single-row key. |
| `current_schema_version` | INTEGER | Current design version is `1`. |

### `scan_batches`

| Column | Affinity | Notes |
| --- | --- | --- |
| `batch_pk` | INTEGER | Internal surrogate primary key. |
| `batch_record_id` | TEXT | Caller-supplied authoritative `BatchRecordID`. |
| `schema_version` | INTEGER | Snapshot interpretation version, initially `1`. |
| `scan_date` | TEXT | `ScanWindowRecord.ScanDate`, RFC3339Nano UTC. |
| `scan_start_of_day` | TEXT | `ScanWindowRecord.StartOfDay`, RFC3339Nano UTC. |
| `scan_started_at` | TEXT | `ScanWindowRecord.ScanStarted`, RFC3339Nano UTC. |
| `scan_timezone` | TEXT | `ScanWindowRecord.Timezone`. |
| `search_profile_id` | TEXT | `SearchProfileRecord.ID`. |
| `search_profile_display_name` | TEXT | `SearchProfileRecord.DisplayName`. |
| `search_profile_is_enabled` | INTEGER | `1` for true, `0` for false. |
| `geographic_mode` | TEXT | Exact `BatchRecord.GeographicMode()` string. |
| `summary_group_count` | INTEGER | `BatchSummaryRecord.GroupCount`. |
| `summary_input_post_count` | INTEGER | `InputPostCount`. |
| `summary_evaluated_post_count` | INTEGER | `EvaluatedPostCount`. |
| `summary_include_post_count` | INTEGER | `IncludePostCount`. |
| `summary_review_post_count` | INTEGER | `ReviewPostCount`. |
| `summary_excluded_post_count` | INTEGER | `ExcludedPostCount`. |
| `summary_aggregated_lead_count` | INTEGER | `AggregatedLeadCount`. |
| `summary_allowed_lead_count` | INTEGER | `AllowedLeadCount`. |
| `summary_blocked_lead_count` | INTEGER | `BlockedLeadCount`. |
| `summary_unresolved_lead_count` | INTEGER | `UnresolvedLeadCount`. |
| `summary_unaggregated_post_count` | INTEGER | `UnaggregatedPostCount`. |
| `summary_source_conflict_count` | INTEGER | `SourceConflictCount`. |
| `summary_allowed_lead_source_post_count` | INTEGER | `AllowedLeadSourcePostCount`. |
| `summary_blocked_lead_source_post_count` | INTEGER | `BlockedLeadSourcePostCount`. |

### `batch_search_profile_terms`

| Column | Affinity | Notes |
| --- | --- | --- |
| `term_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owns term snapshot. |
| `term_kind` | TEXT | `product`, `buyer_intent`, or `noise`. |
| `term_position` | INTEGER | Zero-based position within kind. |
| `term_value` | TEXT | Exact term string. |

### `batch_groups`

| Column | Affinity | Notes |
| --- | --- | --- |
| `group_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `group_position` | INTEGER | Zero-based batch group order. |
| `group_id` | TEXT | Normalized `GroupRecord.GroupID`. |
| `group_name` | TEXT | `GroupRecord.GroupName`. |

### `raw_post_occurrences`

| Column | Affinity | Notes |
| --- | --- | --- |
| `post_occurrence_pk` | INTEGER | Internal primary key for references. |
| `batch_pk` | INTEGER | Owning batch. |
| `group_pk` | INTEGER | Owning group row. |
| `group_position` | INTEGER | Copied from group for validation. |
| `group_post_position` | INTEGER | Zero-based position inside group input. |
| `flattened_position` | INTEGER | Zero-based position in `BatchRecord.FlattenedPosts()`. |
| `post_id` | TEXT | Exact `RawPost.PostID`, may be blank. |
| `group_id` | TEXT | Exact `RawPost.GroupID`, may be blank if source was blank. |
| `group_name` | TEXT | Exact `RawPost.GroupName`. |
| `post_url` | TEXT | Exact `RawPost.PostURL`, may be blank. |
| `author_facebook_user_id` | TEXT | Exact author field. |
| `author_canonical_profile_url` | TEXT | Exact author field. |
| `author_username` | TEXT | Exact author field. |
| `author_display_name` | TEXT | Exact author field. |
| `body` | TEXT | Exact `RawPost.Body`; never overwritten by normalized text. |
| `created_at` | TEXT | Exact timestamp encoded RFC3339Nano UTC. |
| `captured_at` | TEXT | Exact timestamp encoded RFC3339Nano UTC. |

### `evaluated_posts`

| Column | Affinity | Notes |
| --- | --- | --- |
| `evaluated_post_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `post_occurrence_pk` | INTEGER | Original post occurrence. |
| `evaluated_position` | INTEGER | Zero-based order in `EvaluatedPosts()`. |
| `decision` | TEXT | Exact decision string. |
| `geographic_class` | TEXT | Current field is usually blank; preserve exact value. |
| `geographic_reason_set_present` | INTEGER | Optional presence marker for future distinction. |

`EvaluatedPostRecord.Reasons` are in `evaluated_post_reasons`. `GeographicReasons` use the same table with `reason_category = geographic` if non-empty.

### `evaluated_post_reasons`

| Column | Affinity | Notes |
| --- | --- | --- |
| `reason_pk` | INTEGER | Internal primary key. |
| `evaluated_post_pk` | INTEGER | Parent evaluated record. |
| `reason_category` | TEXT | `rule` or `geographic`. |
| `reason_position` | INTEGER | Zero-based order within category. |
| `reason_code` | TEXT | Exact machine-readable code. |

### `bucketed_posts`

| Column | Affinity | Notes |
| --- | --- | --- |
| `bucketed_post_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `bucket` | TEXT | `include`, `review`, or `exclude`. |
| `bucket_position` | INTEGER | Zero-based order within bucket. |
| `post_occurrence_pk` | INTEGER | Original post occurrence. |
| `decision` | TEXT | Exact bucket record decision. |
| `geographic_class` | TEXT | Preserve current field if non-empty. |

### `bucketed_post_reasons`

| Column | Affinity | Notes |
| --- | --- | --- |
| `reason_pk` | INTEGER | Internal primary key. |
| `bucketed_post_pk` | INTEGER | Parent bucketed record. |
| `reason_category` | TEXT | `rule` or `geographic`. |
| `reason_position` | INTEGER | Zero-based order within category. |
| `reason_code` | TEXT | Exact machine-readable code. |

### `leads`

| Column | Affinity | Notes |
| --- | --- | --- |
| `lead_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `lead_position` | INTEGER | Zero-based order in `BatchRecord.Leads()`. |
| `lead_key_value` | TEXT | `LeadKeyRecord.Value`. |
| `lead_key_author_kind` | TEXT | `LeadKeyRecord.Author.Kind`. |
| `lead_key_author_value` | TEXT | `LeadKeyRecord.Author.Value`. |
| `lead_key_need_search_profile_id` | TEXT | `LeadKeyRecord.Need.SearchProfileID`. |
| `lead_key_need_normalized_body` | TEXT | `LeadKeyRecord.Need.NormalizedBody`. |
| `lead_key_need_body_fingerprint` | TEXT | `LeadKeyRecord.Need.BodyFingerprint`. |
| `author_kind` | TEXT | `LeadRecord.Author.Kind`. |
| `author_value` | TEXT | `LeadRecord.Author.Value`. |
| `need_search_profile_id` | TEXT | `LeadRecord.Need.SearchProfileID`. |
| `need_normalized_body` | TEXT | `LeadRecord.Need.NormalizedBody`. |
| `need_body_fingerprint` | TEXT | `LeadRecord.Need.BodyFingerprint`. |

### Lead evidence tables

`lead_key_need_product_evidence`, `lead_key_need_buyer_intent_evidence`, `lead_need_product_evidence`, and `lead_need_buyer_intent_evidence` all use:

| Column | Affinity | Notes |
| --- | --- | --- |
| `evidence_pk` | INTEGER | Internal primary key. |
| `lead_pk` | INTEGER | Parent lead. |
| `evidence_position` | INTEGER | Zero-based evidence order. |
| `evidence_value` | TEXT | Exact evidence string. |

### `lead_sources`

| Column | Affinity | Notes |
| --- | --- | --- |
| `lead_source_pk` | INTEGER | Internal primary key. |
| `lead_pk` | INTEGER | Parent lead. |
| `source_position` | INTEGER | Zero-based order in `LeadRecord.Sources`. |
| `post_occurrence_pk` | INTEGER | Original source post occurrence. |
| `source_key_kind` | TEXT | `SourcePostRecord.Key.Kind`. |
| `source_key_value` | TEXT | `SourcePostRecord.Key.Value`. |

### `lead_outcomes`

| Column | Affinity | Notes |
| --- | --- | --- |
| `lead_outcome_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `lead_pk` | INTEGER | Referenced lead. |
| `outcome_bucket` | TEXT | `allowed`, `blocked`, or `unresolved`. |
| `bucket_position` | INTEGER | Zero-based order within outcome bucket. |
| `match_outcome` | TEXT | Exact `BlocklistMatchRecord.Outcome`. |
| `match_author_key_kind` | TEXT | `BlocklistMatchRecord.AuthorKey.Kind`. |
| `match_author_key_value` | TEXT | `BlocklistMatchRecord.AuthorKey.Value`. |
| `matched_entry_key_kind` | TEXT | `MatchedEntry.Key.Kind`. |
| `matched_entry_key_value` | TEXT | `MatchedEntry.Key.Value`. |
| `matched_entry_display_name` | TEXT | `MatchedEntry.DisplayName`. |

### Outcome reason tables

`lead_outcome_blocklist_reasons` and `lead_outcome_application_reasons` both use:

| Column | Affinity | Notes |
| --- | --- | --- |
| `reason_pk` | INTEGER | Internal primary key. |
| `lead_outcome_pk` | INTEGER | Parent outcome. |
| `reason_position` | INTEGER | Zero-based reason order. |
| `reason_code` | TEXT | Exact reason code string. |

Application reasons are expected mainly for unresolved outcomes.

### `unaggregated_posts`

| Column | Affinity | Notes |
| --- | --- | --- |
| `unaggregated_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `unaggregated_position` | INTEGER | Zero-based order in `BatchRecord.Unaggregated()`. |
| `post_occurrence_pk` | INTEGER | Original post occurrence. |
| `candidate_author_kind` | TEXT | `CandidateRecord.Author.Kind`. |
| `candidate_author_value` | TEXT | `CandidateRecord.Author.Value`. |
| `candidate_need_search_profile_id` | TEXT | `CandidateRecord.Need.SearchProfileID`. |
| `candidate_need_normalized_body` | TEXT | `CandidateRecord.Need.NormalizedBody`. |
| `candidate_need_body_fingerprint` | TEXT | `CandidateRecord.Need.BodyFingerprint`. |
| `source_key_kind` | TEXT | `SourceIdentityRecord.Kind`. |
| `source_key_value` | TEXT | `SourceIdentityRecord.Value`. |

### Unaggregated evidence and reason tables

`unaggregated_candidate_product_evidence` and `unaggregated_candidate_buyer_intent_evidence` both use:

| Column | Affinity | Notes |
| --- | --- | --- |
| `evidence_pk` | INTEGER | Internal primary key. |
| `unaggregated_pk` | INTEGER | Parent unaggregated record. |
| `evidence_position` | INTEGER | Zero-based evidence order. |
| `evidence_value` | TEXT | Exact evidence string. |

`unaggregated_post_reasons` uses:

| Column | Affinity | Notes |
| --- | --- | --- |
| `reason_pk` | INTEGER | Internal primary key. |
| `unaggregated_pk` | INTEGER | Parent unaggregated record. |
| `reason_position` | INTEGER | Zero-based reason order. |
| `reason_code` | TEXT | Exact dedup reason code. |

### `source_conflicts`

| Column | Affinity | Notes |
| --- | --- | --- |
| `source_conflict_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `conflict_position` | INTEGER | Zero-based order in `BatchRecord.Conflicts()`. |
| `post_occurrence_pk` | INTEGER | Current conflict post. |
| `existing_post_occurrence_pk` | INTEGER | Existing source post occurrence. |
| `existing_source_key_kind` | TEXT | `ExistingSource.Key.Kind`. |
| `existing_source_key_value` | TEXT | `ExistingSource.Key.Value`. |
| `candidate_author_kind` | TEXT | `CandidateRecord.Author.Kind`. |
| `candidate_author_value` | TEXT | `CandidateRecord.Author.Value`. |
| `candidate_need_search_profile_id` | TEXT | `CandidateRecord.Need.SearchProfileID`. |
| `candidate_need_normalized_body` | TEXT | `CandidateRecord.Need.NormalizedBody`. |
| `candidate_need_body_fingerprint` | TEXT | `CandidateRecord.Need.BodyFingerprint`. |
| `source_key_kind` | TEXT | `SourceIdentityRecord.Kind`. |
| `source_key_value` | TEXT | `SourceIdentityRecord.Value`. |

### Conflict evidence and reason tables

`source_conflict_candidate_product_evidence` and `source_conflict_candidate_buyer_intent_evidence` both use:

| Column | Affinity | Notes |
| --- | --- | --- |
| `evidence_pk` | INTEGER | Internal primary key. |
| `source_conflict_pk` | INTEGER | Parent conflict. |
| `evidence_position` | INTEGER | Zero-based evidence order. |
| `evidence_value` | TEXT | Exact evidence string. |

`source_conflict_reasons` uses:

| Column | Affinity | Notes |
| --- | --- | --- |
| `reason_pk` | INTEGER | Internal primary key. |
| `source_conflict_pk` | INTEGER | Parent conflict. |
| `reason_position` | INTEGER | Zero-based reason order. |
| `reason_code` | TEXT | Exact dedup reason code. |

### `group_summaries`

| Column | Affinity | Notes |
| --- | --- | --- |
| `group_summary_pk` | INTEGER | Internal primary key. |
| `batch_pk` | INTEGER | Owning batch. |
| `group_pk` | INTEGER | Corresponding ordered group. |
| `group_position` | INTEGER | Zero-based order, matching group order. |
| `group_id` | TEXT | `GroupSummaryRecord.GroupID`. |
| `input_post_count` | INTEGER | `InputPostCount`. |
| `evaluated_post_count` | INTEGER | `EvaluatedPostCount`. |
| `include_post_count` | INTEGER | `IncludePostCount`. |
| `review_post_count` | INTEGER | `ReviewPostCount`. |
| `excluded_post_count` | INTEGER | `ExcludedPostCount`. |

## 6. Primary Keys

- Every table has an internal primary key ending in `_pk`.
- `scan_batches.batch_pk` is only an internal SQLite surrogate.
- `scan_batches.batch_record_id` is unique and authoritative for public save/load identity.
- Child primary keys are implementation details used for compact foreign keys and deterministic reconstruction.

## 7. Foreign Keys

- Every child table belongs directly or indirectly to one `scan_batches` row.
- `batch_search_profile_terms`, `batch_groups`, `raw_post_occurrences`, `evaluated_posts`, `bucketed_posts`, `leads`, `lead_outcomes`, `unaggregated_posts`, `source_conflicts`, and `group_summaries` reference `scan_batches`.
- Reason, evidence, and source tables reference their narrow parent records.
- Post references use `post_occurrence_pk`, not `PostID`.
- Lead outcome references use `lead_pk`, not recomputed lead keys.
- SQLite connections must enable and verify foreign-key enforcement before schema initialization or validation.
- Public delete is not part of `BatchRepository`; delete behavior is deferred. If internal cascading is chosen later, it must be documented as schema integrity behavior, not product-level deletion semantics.

## 8. Unique Constraints

Schema constraints include:

- `scan_batches.batch_record_id` unique.
- `batch_search_profile_terms(batch_pk, term_kind, term_position)` unique.
- `batch_groups(batch_pk, group_position)` unique.
- `batch_groups(batch_pk, group_id)` unique.
- `raw_post_occurrences(batch_pk, flattened_position)` unique.
- `raw_post_occurrences(batch_pk, group_pk, group_post_position)` unique.
- `evaluated_posts(batch_pk, evaluated_position)` unique.
- Reason tables: parent key plus `reason_position` unique, scoped by `reason_category` where present.
- `bucketed_posts(batch_pk, bucket, bucket_position)` unique.
- `leads(batch_pk, lead_position)` unique.
- `lead_sources(lead_pk, source_position)` unique.
- `lead_outcomes(batch_pk, outcome_bucket, bucket_position)` unique.
- `unaggregated_posts(batch_pk, unaggregated_position)` unique.
- `source_conflicts(batch_pk, conflict_position)` unique.
- `group_summaries(batch_pk, group_position)` unique.

## 9. Ordering Fields

No reconstruction may depend on `rowid`, insertion order, lexical sorting, `PostID`, or surrogate key order.

Every ordered collection has an explicit zero-based position:

- SearchProfile term order: `term_position`
- Groups: `group_position`
- Group input posts: `group_post_position`
- Flattened posts: `flattened_position`
- Evaluated posts: `evaluated_position`
- Included/review/excluded buckets: `bucket_position`
- Leads: `lead_position`
- Lead sources: `source_position`
- Outcome buckets: `bucket_position`
- Unaggregated posts: `unaggregated_position`
- Source conflicts: `conflict_position`
- Reasons and evidence: `reason_position` or `evidence_position`
- Group summaries: `group_position`

Future loads must reject duplicate positions and gaps in required ordered positions.

## 10. Enum And Reason Storage Policy

- Enum-like values are stored as exact TEXT strings from current contracts.
- Decisions remain `include`, `review`, or `exclude`.
- Geographic mode remains the exact `domain.GeographicMode` string.
- Blocklist outcomes remain exact strings such as `blocked`, `not_blocked`, and `insufficient_identity`.
- Reason codes remain exact strings in narrow ordered child tables.
- No comma-separated reason text, JSON arrays, global mutable reason-code registry, or prose conversion.
- Unknown enum-like values or unsupported reason categories fail closed during future load where current validation can detect them.

## Data Types And Time Policy

- TEXT stores stable IDs, URLs, post bodies, display names, enum-like values, reason codes, and timestamps.
- INTEGER stores internal surrogate keys, positions, booleans, schema versions, and count fields.
- Timestamps are encoded as RFC3339Nano UTC strings and must preserve the exact instant and nanosecond precision supplied by the snapshot.
- Historical timestamps come only from the supplied `BatchRecord`; future persistence code must not call SQLite current-time functions or Go current time.
- No floating-point timestamps, Go-specific binary encodings, gob, serialized structs, arbitrary BLOB payloads, or arbitrary JSON payloads.

## 11. Source-Post Occurrence Identity

`RawPost.PostID` is not guaranteed to be present or globally unique. The schema therefore identifies source posts by `raw_post_occurrences.post_occurrence_pk`, with unique positional constraints:

- one batch-local `flattened_position`
- one group-local `group_post_position`
- an owning `group_pk`

Evaluated records, bucket records, lead sources, unaggregated records, and conflicts reference the preserved post occurrence. The original raw body and author fields are stored once in `raw_post_occurrences` and are not duplicated into every derived record.

## 12. Transaction Policy

A future save transaction must:

1. validate `BatchRecord` before opening or before mutating the database;
2. begin one transaction;
3. insert the root batch row;
4. insert every child collection in deterministic accessor order;
5. verify affected rows where practical;
6. commit only after the full snapshot is inserted;
7. roll back on any failure.

Required behavior:

- no partial batch visibility;
- duplicate `BatchRecordID` fails and leaves database unchanged;
- no update, upsert, replace, or retry loop;
- no cross-batch mutation;
- transactions are internal to the adapter and are not exposed through `BatchRepository`.

## 13. Future Load And Reconstruction Policy

`BatchRepository` remains save-only. If a future milestone adds loading, it must fail closed:

- fetch one root by `BatchRecordID`;
- reject missing or unsupported database/schema version;
- load every child collection ordered only by explicit position columns;
- rebuild `persistence.BatchRecord` without rerunning rules, geography, dedup, blocklist, aggregation, or summary calculation;
- reject missing required child rows;
- reject duplicate positions;
- reject gaps in required ordered positions;
- reject unknown decisions, outcomes, and reason categories where current validation can detect them;
- reject inconsistent group/post relationships;
- run `BatchRecord.Validate` after reconstruction;
- return no partially valid snapshot on any failure.

## 14. Validation Layers

Validation is layered:

1. `BatchRecord.Validate` before save.
2. Database foreign-key, uniqueness, and non-null constraints during insertion.
3. `BatchRecord.Validate` after future load reconstruction.

Database constraints do not replace application validation. Application validation does not replace ownership, foreign-key, and uniqueness constraints. Persistence validation never reruns business rules.

## 15. Schema-Version Policy

- Current design schema version is `1`.
- `schema_metadata.current_schema_version` stores the database version.
- `scan_batches.schema_version` stores the snapshot interpretation version for each root batch.
- Missing version fails closed.
- Unsupported newer versions fail closed.
- Duplicate or malformed schema metadata fails closed.
- No best-effort loading of unknown schemas.
- No automatic destructive downgrade.
- Migration execution remains deferred.

## 16. Migration Policy

Future migrations must be:

- explicit, ordered, and one-way;
- sequential by integer version;
- fully transactional;
- rolled back completely on failure;
- tested with temporary databases and deterministic fixtures.

Application startup must not silently delete or recreate an incompatible database. Unsupported versions produce a stable storage-layer error. Backup policy remains deferred.

## 17. Index Policy

Design only justified indexes:

- unique index for `scan_batches.batch_record_id`;
- indexes for child foreign keys used during save/load;
- unique indexes for group, post, lead, outcome, reason, evidence, and summary positions;
- index for `raw_post_occurrences(batch_pk, flattened_position)`;
- index for `lead_sources(lead_pk, source_position)`;
- index for reason parent/position lookups.

No speculative full-text search indexes, broad filtering indexes, cross-batch reporting indexes, or indexes for hypothetical APIs.

## 18. Local-Only And Security Boundary

Future durable storage is a local SQLite database only.

The database must not store:

- Facebook credentials;
- browser cookies;
- Facebook session data;
- raw browser profile data;
- access tokens;
- cloud sync state.

No network access is part of persistence. Encryption at rest and key management are deferred; SQLite alone must not be described as encryption.

## 19. Deferred Work

Deferred:

- durable SQLite `SaveBatch` implementation;
- migrations and migration tests;
- storage-layer error taxonomy;
- durable load/list APIs;
- temporary-database integration tests;
- backup policy;
- encryption-at-rest policy;
- deletion semantics;
- search or reporting APIs;
- UI/CLI wiring;
- generated IDs;
- import/export.

## 20. BatchRecord Field Mapping

| `BatchRecord` field or accessor | Storage |
| --- | --- |
| `ID()` / `id` | `scan_batches.batch_record_id` |
| `ScanWindow().ScanDate` | `scan_batches.scan_date` |
| `ScanWindow().StartOfDay` | `scan_batches.scan_start_of_day` |
| `ScanWindow().ScanStarted` | `scan_batches.scan_started_at` |
| `ScanWindow().Timezone` | `scan_batches.scan_timezone` |
| `SearchProfile().ID` | `scan_batches.search_profile_id` |
| `SearchProfile().DisplayName` | `scan_batches.search_profile_display_name` |
| `SearchProfile().ProductTerms` | `batch_search_profile_terms` with `term_kind = product` |
| `SearchProfile().BuyerIntentTerms` | `batch_search_profile_terms` with `term_kind = buyer_intent` |
| `SearchProfile().NoiseTerms` | `batch_search_profile_terms` with `term_kind = noise` |
| `SearchProfile().IsEnabled` | `scan_batches.search_profile_is_enabled` |
| `GeographicMode()` | `scan_batches.geographic_mode` |
| `Groups()` | `batch_groups`; group posts reference `raw_post_occurrences` by group position |
| `FlattenedPosts()` | `raw_post_occurrences.flattened_position` and exact raw post columns |
| `EvaluatedPosts()` | `evaluated_posts` plus `evaluated_post_reasons` |
| `IncludedPosts()` | `bucketed_posts(bucket = include)` plus `bucketed_post_reasons` |
| `ReviewPosts()` | `bucketed_posts(bucket = review)` plus `bucketed_post_reasons` |
| `ExcludedPosts()` | `bucketed_posts(bucket = exclude)` plus `bucketed_post_reasons` |
| `Leads()` | `leads`, lead evidence tables, and `lead_sources` |
| `AllowedLeads()` | `lead_outcomes(outcome_bucket = allowed)` plus outcome reason tables |
| `BlockedLeads()` | `lead_outcomes(outcome_bucket = blocked)` plus outcome reason tables |
| `UnresolvedLeads()` | `lead_outcomes(outcome_bucket = unresolved)` plus blocklist and application reason tables |
| `Unaggregated()` | `unaggregated_posts`, evidence tables, and `unaggregated_post_reasons` |
| `Conflicts()` | `source_conflicts`, evidence tables, and `source_conflict_reasons` |
| `Summary()` | summary columns on `scan_batches` |
| `GroupSummaries()` | `group_summaries` |

## 21. Snapshot Reconstruction Checklist

Before a future implementation is accepted:

- every `BatchRecord` accessor must have a storage source;
- every nested collection must have a parent/child table;
- every ordered slice must have a position column;
- every source post must reference `raw_post_occurrences`;
- every reason slice must be stored as ordered child rows;
- summaries must be stored and then structurally verified against reconstructed collections;
- application and orchestration boundaries must remain unchanged.
