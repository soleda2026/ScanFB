// Package persistence owns persistence-facing contracts and deterministic in-memory
// adapters for completed scan batch snapshots.
//
// It intentionally does not implement durable local storage, schemas, migrations,
// file I/O, network I/O, UI, CLI, or Facebook integration.
package persistence
