// Package dedup owns deterministic duplicate identity primitives.
//
// Phase 4B implements in-memory lead aggregation while preserving every source
// post. It intentionally does not implement persistence, repositories, or
// Facebook integration.
package dedup
