package persistence

import "errors"

var ErrBatchRecordAlreadyExists = errors.New("batch record already exists")

// InMemoryBatchRepository is a deterministic, single-process adapter for tests
// and future wiring before durable local persistence exists.
type InMemoryBatchRepository struct {
	records []BatchRecord
	index   map[BatchRecordID]int
}

func NewInMemoryBatchRepository() *InMemoryBatchRepository {
	return &InMemoryBatchRepository{}
}

func (repo *InMemoryBatchRepository) SaveBatch(record BatchRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	repo.ensureIndex()
	if _, exists := repo.index[record.ID()]; exists {
		return ErrBatchRecordAlreadyExists
	}

	repo.records = append(repo.records, copyBatchRecord(record))
	repo.index[record.ID()] = len(repo.records) - 1
	return nil
}

func (repo *InMemoryBatchRepository) Count() int {
	return len(repo.records)
}

func (repo *InMemoryBatchRepository) Records() []BatchRecord {
	records := make([]BatchRecord, len(repo.records))
	for i, record := range repo.records {
		records[i] = copyBatchRecord(record)
	}
	return records
}

func (repo *InMemoryBatchRepository) RecordByID(id BatchRecordID) (BatchRecord, bool) {
	if id.String() == "" || repo.index == nil {
		return BatchRecord{}, false
	}
	position, ok := repo.index[id]
	if !ok {
		return BatchRecord{}, false
	}
	return copyBatchRecord(repo.records[position]), true
}

func (repo *InMemoryBatchRepository) ensureIndex() {
	if repo.index != nil {
		return
	}
	repo.index = make(map[BatchRecordID]int, len(repo.records))
	for i, record := range repo.records {
		repo.index[record.ID()] = i
	}
}

func copyBatchRecord(record BatchRecord) BatchRecord {
	return BatchRecord{
		id:              record.id,
		scanWindow:      record.scanWindow,
		searchProfile:   record.SearchProfile(),
		geographicMode:  record.geographicMode,
		groups:          copyGroupRecords(record.groups),
		flattenedPosts:  copyRawPosts(record.flattenedPosts),
		evaluatedPosts:  copyEvaluatedPostRecords(record.evaluatedPosts),
		includedPosts:   copyEvaluatedPostRecords(record.includedPosts),
		reviewPosts:     copyEvaluatedPostRecords(record.reviewPosts),
		excludedPosts:   copyEvaluatedPostRecords(record.excludedPosts),
		leads:           copyLeadRecords(record.leads),
		allowedLeads:    copyAllowedLeadRecords(record.allowedLeads),
		blockedLeads:    copyBlockedLeadRecords(record.blockedLeads),
		unresolvedLeads: copyUnresolvedLeadRecords(record.unresolvedLeads),
		unaggregated:    copyUnaggregatedPostRecords(record.unaggregated),
		conflicts:       copySourceConflictRecords(record.conflicts),
		summary:         record.summary,
		groupSummaries:  append([]GroupSummaryRecord(nil), record.groupSummaries...),
	}
}
