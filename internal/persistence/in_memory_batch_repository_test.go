package persistence

import (
	"errors"
	"reflect"
	"testing"
)

func TestInMemoryBatchRepositoryConstructionAndInterface(t *testing.T) {
	var _ BatchRepository = (*InMemoryBatchRepository)(nil)

	repo := NewInMemoryBatchRepository()
	if repo == nil {
		t.Fatal("NewInMemoryBatchRepository returned nil")
	}
	if repo.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", repo.Count())
	}
	if got := repo.Records(); len(got) != 0 {
		t.Fatalf("Records() = %#v, want empty", got)
	}
	if got, ok := repo.RecordByID(""); ok || !reflect.DeepEqual(got, BatchRecord{}) {
		t.Fatalf("RecordByID(empty) = %#v, %v; want zero, false", got, ok)
	}

	var zero InMemoryBatchRepository
	if zero.Count() != 0 {
		t.Fatalf("zero Count() = %d, want 0", zero.Count())
	}
	if got := zero.Records(); len(got) != 0 {
		t.Fatalf("zero Records() = %#v, want empty", got)
	}
	if got, ok := zero.RecordByID(BatchRecordID("missing")); ok || !reflect.DeepEqual(got, BatchRecord{}) {
		t.Fatalf("zero RecordByID(missing) = %#v, %v; want zero, false", got, ok)
	}
}

func TestInMemoryBatchRepositorySavesValidSnapshots(t *testing.T) {
	var repo InMemoryBatchRepository
	record := buildTestBatchRecord(t)

	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if got, want := repo.Count(), 1; got != want {
		t.Fatalf("Count() = %d, want %d", got, want)
	}
	if got := repo.Records(); len(got) != 1 || !reflect.DeepEqual(got[0], record) {
		t.Fatalf("Records() = %#v, want original snapshot", got)
	}
	stored, ok := repo.RecordByID(record.ID())
	if !ok {
		t.Fatalf("RecordByID(%q) not found", record.ID())
	}
	if !reflect.DeepEqual(stored, record) {
		t.Fatalf("RecordByID() = %#v, want %#v", stored, record)
	}
	if got, want := stored.ID(), record.ID(); got != want {
		t.Fatalf("stored ID = %q, want %q", got, want)
	}
}

func TestInMemoryBatchRepositorySavesFiveDistinctSnapshots(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	var want []BatchRecord
	for i := 1; i <= 5; i++ {
		record := buildTestBatchRecordWithID(t, BatchRecordID("batch-00"+string(rune('0'+i))))
		if err := repo.SaveBatch(record); err != nil {
			t.Fatalf("SaveBatch(%q) error = %v", record.ID(), err)
		}
		want = append(want, record)
	}

	if got := repo.Count(); got != len(want) {
		t.Fatalf("Count() = %d, want %d", got, len(want))
	}
	if got := repo.Records(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Records() order = %#v, want %#v", batchRecordIDs(got), batchRecordIDs(want))
	}
}

func TestInMemoryBatchRepositoryOrderingIsStable(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	first := buildTestBatchRecordWithID(t, "batch-001")
	second := buildTestBatchRecordWithID(t, "batch-002")
	invalid := BatchRecord{}

	if err := repo.SaveBatch(first); err != nil {
		t.Fatalf("SaveBatch(first) error = %v", err)
	}
	if err := repo.SaveBatch(second); err != nil {
		t.Fatalf("SaveBatch(second) error = %v", err)
	}
	if err := repo.SaveBatch(first); !errors.Is(err, ErrBatchRecordAlreadyExists) {
		t.Fatalf("duplicate SaveBatch error = %v, want %v", err, ErrBatchRecordAlreadyExists)
	}
	if err := repo.SaveBatch(invalid); !errors.Is(err, ErrEmptyBatchRecordID) {
		t.Fatalf("invalid SaveBatch error = %v, want %v", err, ErrEmptyBatchRecordID)
	}
	if _, ok := repo.RecordByID(first.ID()); !ok {
		t.Fatalf("RecordByID(%q) not found", first.ID())
	}

	wantIDs := []BatchRecordID{first.ID(), second.ID()}
	if got := batchRecordIDs(repo.Records()); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("Records() IDs = %#v, want %#v", got, wantIDs)
	}
	if got := batchRecordIDs(repo.Records()); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("repeated Records() IDs = %#v, want %#v", got, wantIDs)
	}
}

func TestInMemoryBatchRepositoryDuplicateIdentityFailsWithoutOverwrite(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	original := buildTestBatchRecordWithID(t, "batch-001")
	duplicate := buildTestBatchRecordWithID(t, "batch-001")
	duplicate.groups[0].GroupName = "mutated duplicate"
	differentID := buildTestBatchRecordWithID(t, "batch-002")

	if err := repo.SaveBatch(original); err != nil {
		t.Fatalf("SaveBatch(original) error = %v", err)
	}
	if err := repo.SaveBatch(duplicate); !errors.Is(err, ErrBatchRecordAlreadyExists) {
		t.Fatalf("SaveBatch(duplicate) error = %v, want %v", err, ErrBatchRecordAlreadyExists)
	}
	if got := repo.Count(); got != 1 {
		t.Fatalf("Count() after duplicate = %d, want 1", got)
	}
	stored, ok := repo.RecordByID(original.ID())
	if !ok {
		t.Fatalf("original ID not found after duplicate failure")
	}
	if !reflect.DeepEqual(stored, original) {
		t.Fatalf("duplicate replaced original:\ngot  %#v\nwant %#v", stored, original)
	}
	if err := repo.SaveBatch(differentID); err != nil {
		t.Fatalf("SaveBatch(differentID duplicate content) error = %v", err)
	}
	if got := repo.Count(); got != 2 {
		t.Fatalf("Count() after distinct ID = %d, want 2", got)
	}
}

func TestInMemoryBatchRepositoryValidationFailuresDoNotStore(t *testing.T) {
	tests := []struct {
		name string
		edit func(*BatchRecord)
		want error
	}{
		{name: "zero record", edit: func(record *BatchRecord) { *record = BatchRecord{} }, want: ErrEmptyBatchRecordID},
		{name: "empty id", edit: func(record *BatchRecord) { record.id = "" }, want: ErrEmptyBatchRecordID},
		{name: "invalid scan window", edit: func(record *BatchRecord) { record.scanWindow = ScanWindowRecord{} }, want: ErrInvalidBatchRecord},
		{name: "invalid profile snapshot", edit: func(record *BatchRecord) { record.searchProfile.ID = "" }, want: ErrInvalidBatchRecord},
		{name: "invalid geographic mode", edit: func(record *BatchRecord) { record.geographicMode = "foreign" }, want: ErrInvalidBatchRecord},
		{name: "invalid group structure", edit: func(record *BatchRecord) { record.groups[0].GroupID = "" }, want: ErrInvalidBatchRecord},
		{name: "unsupported rule decision", edit: func(record *BatchRecord) { record.evaluatedPosts[0].Decision = "hold" }, want: ErrUnsupportedDecision},
		{name: "unsupported lead outcome", edit: func(record *BatchRecord) { record.allowedLeads[0].Match.Outcome = "blocked" }, want: ErrUnsupportedLeadOutcome},
		{name: "unsupported block outcome", edit: func(record *BatchRecord) { record.allowedLeads[0].Match.Outcome = "unknown" }, want: ErrUnsupportedBlockOutcome},
		{name: "inconsistent summary", edit: func(record *BatchRecord) { record.summary.InputPostCount++ }, want: ErrInconsistentBatchSummary},
		{name: "malformed source identity", edit: func(record *BatchRecord) { record.leads[0].Sources[0].Key = SourceIdentityRecord{} }, want: ErrInvalidBatchRecord},
		{name: "malformed lead identity", edit: func(record *BatchRecord) { record.leads[0].Key.Value = "" }, want: ErrInvalidBatchRecord},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := NewInMemoryBatchRepository()
			record := buildTestBatchRecord(t)
			test.edit(&record)
			if err := repo.SaveBatch(record); !errors.Is(err, test.want) {
				t.Fatalf("SaveBatch() error = %v, want %v", err, test.want)
			}
			if got := repo.Count(); got != 0 {
				t.Fatalf("Count() after failed save = %d, want 0", got)
			}
			if got := repo.Records(); len(got) != 0 {
				t.Fatalf("Records() after failed save = %#v, want empty", got)
			}
		})
	}
}

func TestInMemoryBatchRepositoryValidationFailureAfterSavePreservesState(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	valid := buildTestBatchRecordWithID(t, "batch-001")
	invalid := buildTestBatchRecordWithID(t, "batch-002")
	invalid.summary.InputPostCount++

	if err := repo.SaveBatch(valid); err != nil {
		t.Fatalf("SaveBatch(valid) error = %v", err)
	}
	before := repo.Records()
	if err := repo.SaveBatch(invalid); !errors.Is(err, ErrInconsistentBatchSummary) {
		t.Fatalf("SaveBatch(invalid) error = %v, want %v", err, ErrInconsistentBatchSummary)
	}
	after := repo.Records()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("failed save changed repository state:\nbefore %#v\nafter  %#v", before, after)
	}
}

func TestInMemoryBatchRepositoryDefensiveStorageAndReads(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	original := buildTestBatchRecordWithID(t, "batch-001")
	want := copyBatchRecord(original)

	if err := repo.SaveBatch(original); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	original.groups[0].Posts[0].Body = "mutated original"
	original.searchProfile.ProductTerms[0] = "mutated product"
	original.allowedLeads[0].Match.Reasons[0] = "mutated blocklist reason"
	original.allowedLeads[0].Lead.Sources[0].Post.Body = "mutated source"
	original.groupSummaries[0].InputPostCount = 0

	stored, ok := repo.RecordByID(want.ID())
	if !ok {
		t.Fatalf("RecordByID(%q) not found", want.ID())
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored record changed after original mutation:\ngot  %#v\nwant %#v", stored, want)
	}

	records := repo.Records()
	records[0].groups[0].Posts[0].Body = "mutated Records result"
	records[0].reviewPosts[0].Reasons[0] = "mutated rule reason"
	records[0].blockedLeads[0].Match.Reasons[0] = "mutated blocked reason"
	records[0].leads[0].Sources[0].Post.Body = "mutated source"
	records[0].groupSummaries[0].InputPostCount = 0
	if got := repo.Records()[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("Records result mutation changed stored state:\ngot  %#v\nwant %#v", got, want)
	}

	lookup, ok := repo.RecordByID(want.ID())
	if !ok {
		t.Fatalf("RecordByID(%q) not found on repeat", want.ID())
	}
	lookup.groups[0].Posts[0].Body = "mutated lookup result"
	lookup.allowedLeads[0].Lead.Sources[0].Post.Body = "mutated lookup source"
	lookup.allowedLeads[0].Match.Reasons[0] = "mutated lookup reason"
	if got := mustRecordByID(t, repo, want.ID()); !reflect.DeepEqual(got, want) {
		t.Fatalf("RecordByID result mutation changed stored state:\ngot  %#v\nwant %#v", got, want)
	}
	if first, second := repo.Records(), repo.Records(); !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated Records reads differ:\nfirst  %#v\nsecond %#v", first, second)
	}
}

func TestInMemoryBatchRepositoryStoredRecordsRemainIndependent(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	first := buildTestBatchRecordWithID(t, "batch-001")
	second := buildTestBatchRecordWithID(t, "batch-002")
	if err := repo.SaveBatch(first); err != nil {
		t.Fatalf("SaveBatch(first) error = %v", err)
	}
	if err := repo.SaveBatch(second); err != nil {
		t.Fatalf("SaveBatch(second) error = %v", err)
	}

	records := repo.Records()
	records[0].groups[0].Posts[0].Body = "mutated first read"
	records[0].allowedLeads[0].Lead.Sources[0].Post.Body = "mutated first source"
	records[0].allowedLeads[0].Match.Reasons[0] = "mutated first reason"

	if got := mustRecordByID(t, repo, first.ID()); !reflect.DeepEqual(got, first) {
		t.Fatalf("first stored record mutated:\ngot  %#v\nwant %#v", got, first)
	}
	if got := mustRecordByID(t, repo, second.ID()); !reflect.DeepEqual(got, second) {
		t.Fatalf("second stored record mutated through first:\ngot  %#v\nwant %#v", got, second)
	}
}

func TestInMemoryBatchRepositorySourceAndOutcomePreservation(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	record := buildTestBatchRecordWithID(t, "batch-001")
	record.groups[0].Posts[0].Body = "Cần mua MacBook Pro tại HCM, ngân sách rõ"
	record.flattenedPosts[0].Body = "Cần mua MacBook Pro tại HCM, ngân sách rõ"
	record.allowedLeads[0].Lead.Sources[0].Post.Body = "Cần mua MacBook Pro tại HCM, ngân sách rõ"
	record.leads[0].Sources[0].Post.Body = "Cần mua MacBook Pro tại HCM, ngân sách rõ"

	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	stored := mustRecordByID(t, repo, record.ID())

	if got, want := stored.FlattenedPosts()[0].Body, "Cần mua MacBook Pro tại HCM, ngân sách rõ"; got != want {
		t.Fatalf("RawPost body = %q, want %q", got, want)
	}
	if got, want := stored.FlattenedPosts()[0], record.FlattenedPosts()[0]; got.PostID != want.PostID ||
		got.GroupID != want.GroupID ||
		got.PostURL != want.PostURL ||
		got.Author != want.Author ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.CapturedAt.Equal(want.CapturedAt) {
		t.Fatalf("RawPost identity/times not preserved:\ngot  %#v\nwant %#v", got, want)
	}
	if got, want := stored.ExcludedPosts()[0].Reasons, []string{"excluded.seller_intent"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("excluded reasons = %#v, want %#v", got, want)
	}
	if got, want := stored.ReviewPosts()[0].Reasons, []string{"review.unknown_location"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("review reasons = %#v, want %#v", got, want)
	}
	if got, want := stored.AllowedLeads()[0].Match.Outcome, "not_blocked"; got != want {
		t.Fatalf("allowed outcome = %q, want %q", got, want)
	}
	if got, want := stored.BlockedLeads()[0].Match.Outcome, "blocked"; got != want {
		t.Fatalf("blocked outcome = %q, want %q", got, want)
	}
	if got, want := stored.Unaggregated()[0].Reasons, []string{"dedup.stable_author_identity_missing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unaggregated reasons = %#v, want %#v", got, want)
	}
	if got, want := postIDsFromSources(stored.AllowedLeads()[0].Lead.Sources), []string{"post-1", "post-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("source order = %#v, want %#v", got, want)
	}
}

func TestInMemoryBatchRepositoryEmptyAndNotFoundReadsAreSafe(t *testing.T) {
	repo := NewInMemoryBatchRepository()
	if got := repo.Count(); got != 0 {
		t.Fatalf("empty Count() = %d, want 0", got)
	}
	if got := repo.Records(); len(got) != 0 {
		t.Fatalf("empty Records() = %#v, want empty", got)
	}
	if got, ok := repo.RecordByID("missing"); ok || !reflect.DeepEqual(got, BatchRecord{}) {
		t.Fatalf("missing RecordByID = %#v, %v; want zero, false", got, ok)
	}
	if got, ok := repo.RecordByID(""); ok || !reflect.DeepEqual(got, BatchRecord{}) {
		t.Fatalf("empty ID RecordByID = %#v, %v; want zero, false", got, ok)
	}
	if got := repo.Count(); got != 0 {
		t.Fatalf("reads modified empty repo count = %d, want 0", got)
	}
}

func TestInMemoryBatchRepositoryDeterminism(t *testing.T) {
	buildRepo := func(t *testing.T) []BatchRecord {
		t.Helper()
		repo := NewInMemoryBatchRepository()
		for _, id := range []BatchRecordID{"batch-001", "batch-002", "batch-003"} {
			if err := repo.SaveBatch(buildTestBatchRecordWithID(t, id)); err != nil {
				t.Fatalf("SaveBatch(%q) error = %v", id, err)
			}
		}
		return repo.Records()
	}

	first := buildRepo(t)
	second := buildRepo(t)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical save sequences differ:\nfirst  %#v\nsecond %#v", first, second)
	}
}

func buildTestBatchRecordWithID(t *testing.T, id BatchRecordID) BatchRecord {
	t.Helper()

	record := buildTestBatchRecord(t)
	record.id = id
	if err := record.Validate(); err != nil {
		t.Fatalf("test record %q invalid: %v", id, err)
	}
	return record
}

func batchRecordIDs(records []BatchRecord) []BatchRecordID {
	ids := make([]BatchRecordID, len(records))
	for i, record := range records {
		ids[i] = record.ID()
	}
	return ids
}

func mustRecordByID(t *testing.T, repo *InMemoryBatchRepository, id BatchRecordID) BatchRecord {
	t.Helper()

	record, ok := repo.RecordByID(id)
	if !ok {
		t.Fatalf("RecordByID(%q) not found", id)
	}
	return record
}
