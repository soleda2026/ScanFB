// Package persistence owns persistence-facing contracts, deterministic in-memory
// adapters, and local SQLite snapshot storage for completed scan batches.
//
// It intentionally does not implement migrations, network I/O, UI, CLI,
// Facebook integration, or broad list/update/delete/search repository APIs.
package persistence
