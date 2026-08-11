// Package facebook owns the outer adapter boundary for prepared Facebook page data.
//
// Phase 10A accepts typed local prepared snapshots. Phase 10B1 adds bounded,
// user-triggered acquisition of Safari's active tab, Phase 10B2d adds a
// separate bounded rendered-document acquisition probe, and Phase 10B2f adds
// pure count/shape-only reconnaissance reporting. Production DOM selector
// validation remains deferred, and domain must not import this package.
package facebook
