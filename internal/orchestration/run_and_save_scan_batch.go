package orchestration

import (
	"errors"
	"reflect"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/persistence"
)

var ErrNilBatchRepository = errors.New("orchestration: nil batch repository")

// RunAndSaveScanBatchResult returns both the completed application result and
// the record that was accepted by the repository.
type RunAndSaveScanBatchResult struct {
	ScanBatchResult application.ScanBatchResult
	BatchRecord     persistence.BatchRecord
}

// RunAndSaveScanBatch runs one deterministic in-memory batch, converts the
// completed result into a BatchRecord, then saves it through the repository.
func RunAndSaveScanBatch(recordID persistence.BatchRecordID, input application.ScanBatchInput, repository persistence.BatchRepository) (RunAndSaveScanBatchResult, error) {
	if batchRepositoryIsNil(repository) {
		return RunAndSaveScanBatchResult{}, ErrNilBatchRepository
	}
	if _, err := persistence.NewBatchRecordID(recordID.String()); err != nil {
		return RunAndSaveScanBatchResult{}, err
	}

	scanResult, err := application.RunScanBatch(input)
	if err != nil {
		return RunAndSaveScanBatchResult{}, err
	}

	record, err := persistence.NewBatchRecord(recordID, input, scanResult)
	if err != nil {
		return RunAndSaveScanBatchResult{}, err
	}

	if err := repository.SaveBatch(record); err != nil {
		return RunAndSaveScanBatchResult{}, err
	}

	return RunAndSaveScanBatchResult{
		ScanBatchResult: scanResult,
		BatchRecord:     record,
	}, nil
}

func batchRepositoryIsNil(repository persistence.BatchRepository) bool {
	if repository == nil {
		return true
	}
	value := reflect.ValueOf(repository)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
