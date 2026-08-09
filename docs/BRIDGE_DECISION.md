# Bridge Decision

Phase 8I.1 selects the bridge model for future native macOS integration slices.
This is a documentation-only decision. No bridge code, generated binding,
runtime process, socket, build phase, package, dependency, schema implementation
or app behavior is added in this milestone.

## 1. Context

ScanFB has an authoritative Go core and a native SwiftUI macOS app under
`macos/ScanFBApp/`. The SwiftUI app currently uses deterministic fixture data.
Go owns domain rules, persistence semantics, blocklist semantics and future scan
orchestration. Swift owns native presentation and transient local UI state.

Phase 8I must connect these layers later without turning Swift into a business
rules layer and without exposing browser state, credentials, cookies, direct
SQLite access or hidden network listeners.

## 2. Non-Goals

- No bridge implementation in Phase 8I.1.
- No Go or Swift source change.
- No dependency, generated binding, build script or shell build phase.
- No localhost HTTP server, WebSocket server, cloud API or embedded web UI.
- No direct Swift SQLite access.
- No Facebook adapter, browser extension or browser automation bridge.
- No arbitrary JSON dictionary API, generic command bus or broad RPC surface.
- No credential, cookie, browser session or database handle transfer.

## 3. Candidate Options

### Local Subprocess Request/Response Bridge

Swift launches a bundled Go helper for one explicit request, writes a bounded
typed request, reads one bounded typed response, then the helper exits. The
transport is local process stdin/stdout or equivalent inherited file handles,
not a long-lived hidden server.

### Local IPC Boundary

Swift talks to a long-lived local helper over a Unix domain socket or equivalent
local-only framed protocol. The helper owns Go core calls and returns explicit
responses.

### In-Process Go Library Binding

Go is compiled as a C ABI-compatible shared library or manually wrapped library.
Swift calls exported Go functions through a compiled binary boundary.

### Rejected By Policy

Localhost HTTP, WebSocket, cloud APIs, direct SQLite, browser extension bridges
and Facebook adapter bridges are not selected candidates for Phase 8I because
they conflict with the local-first, no hidden listener, no browser/session
transfer and Go-authority boundaries.

## 4. Comparison Table

| Criteria | Local subprocess | Local IPC | In-process Go library |
| --- | --- | --- | --- |
| Deterministic request/response | Strong for one request per process | Strong if framed carefully | Strong but in-process failures can leak state |
| Explicit typed schemas | Required and simple | Required, more protocol surface | Required, but ABI wrappers add parallel types |
| Error propagation | Exit code plus typed error response | Typed protocol plus socket errors | Return codes/errors through ABI wrapper |
| Cancellation | Bounded per-call cancellation; exact shutdown policy deferred | Requires request IDs and cancellation frames | Harder; must thread cancellation through ABI |
| Crash isolation | Strong: helper crash does not crash SwiftUI | Strong if helper is separate | Weak: Go crash can crash app process |
| No hidden networking | Strong | Strong only if socket lifecycle is tightly owned | Strong |
| No credential/browser transfer | Strong by contract | Strong by contract | Strong by contract |
| SwiftUI ownership boundaries | Clear | Clear but helper lifecycle leaks into UI if broad | Risk of Swift-side service abstractions growing |
| Go core authority | Clear | Clear | Clear but wrappers can duplicate shape |
| Build complexity | Moderate, understandable | Higher due helper daemon/socket | Higher due c-shared, ABI and linking |
| Xcode integration complexity | Moderate future helper packaging | Higher lifecycle/signing setup | Higher library search/signing/linking setup |
| Packaging | Bundle signed helper in app | Bundle helper plus socket lifecycle policy | Bundle signed dynamic library |
| Apple Silicon support | Good with native helper binary | Good with native helper binary | Good but ABI/linking must be validated |
| Testability | Good: fixture stdin/stdout tests | Good but needs lifecycle tests | Unit tests plus ABI integration complexity |
| Debugging | Simple diagnostics on stderr plus exit codes | More moving parts | Harder mixed Swift/Go process debugging |
| Startup overhead | Higher per request | Lower after helper start | Lowest |
| Runtime overhead | Acceptable for narrow slices | Lower for repeated calls | Lowest |
| Versioning/schema evolution | Explicit envelope | Explicit framed protocol | ABI and schema both versioned |
| Observability without secrets | stderr/structured logs, redacted | protocol logs, redacted | app logs, redacted |
| Fail-closed behavior | Strong: nonzero exit/no response fails closed | Strong if lifecycle is correct | Weaker if crash brings down app |
| Notarization/signing | Signed helper inside app bundle | Signed helper plus runtime policy | Signed library and link settings |
| Keeps slices narrow | Strong, because each request has cost | Medium, long-lived helper invites broad API | Medium, library surface can expand silently |

## 5. Selected Bridge

Select the local subprocess request/response bridge.

Future milestones should package a small Go helper binary with the macOS app.
Swift will launch that helper only for explicit user- or UI-requested slices,
send one versioned typed request, read one versioned typed response from stdout,
surface any transport/domain error, and let the helper exit. The bridge must not
become a generic command bus.

## 6. Why Alternatives Were Rejected

Local IPC is deferred because a long-lived helper requires socket lifecycle,
framing, request IDs, cancellation frames, stale-process cleanup and extra
signing/notarization reasoning before ScanFB has proven one integration slice.
It is more machinery than the current fixture-only UI needs.

In-process Go library binding is rejected for Phase 8I because C ABI wrappers
and generated or manual bindings add build/link complexity, weaken crash
isolation, make cancellation harder, and can crash the SwiftUI app if Go panics
or the library boundary misbehaves.

Localhost HTTP and WebSocket servers are rejected by policy. They create a
network listener shape that ScanFB does not need for a local macOS app.

Direct Swift SQLite access is rejected because Go owns persistence semantics and
SQLite reconstruction/fail-closed behavior.

Browser extension and Facebook adapter bridges are rejected because the bridge
between SwiftUI and Go must not transfer browser state or automate Facebook.

## 7. Ownership Boundaries

- Go owns domain rules, SearchProfile behavior, geographic classification,
  deduplication, blocklist semantics, batch evaluation, orchestration and
  persistence semantics.
- Swift owns windows, navigation, formatting, accessibility, fixture screens and
  transient local UI state.
- Swift must not duplicate Go business rules or reason-code generation.
- Swift must not access SQLite directly or receive database-local IDs.
- The subprocess bridge is only a transport boundary to typed Go-owned use
  cases, not a new business layer.

## 8. Request/Response Contract Principles

Future bridge schemas must use:

- explicit versioned request and response structs;
- finite enums;
- stable field names;
- deterministic serialization;
- bounded payload sizes;
- stdout reserved for the bounded machine-readable bridge response;
- diagnostic logging written to stderr only;
- bounded and redacted diagnostics that never corrupt the response channel;
- an explicit success/error envelope or equally explicit stderr/exit-code plus
  typed error response policy;
- no arbitrary dictionaries or untyped maps;
- no implicit nullable semantics;
- no secrets, credentials, cookies or browser session data;
- no raw database handles, SQL transport or database-local IDs;
- no filesystem path assumptions until a later milestone defines path policy.

The first implementation may use JSON if every payload is typed and versioned.
JSON must not mean arbitrary maps.

## 9. Error Model

Future bridge calls must fail closed and distinguish at least:

- helper not found or startup failure;
- transport write/read failure;
- malformed request;
- unsupported request schema version;
- unsupported response schema version;
- Go domain/application error;
- cancellation;
- timeout;
- helper nonzero exit;
- malformed response;
- oversized response.

Errors must be explicit enough for UI copy and tests, but logs must redact any
future sensitive values.

## 10. Cancellation Model

For the selected subprocess bridge, cancellation is per-call and fail-closed:

- One bridge call owns at most one helper process.
- This does not decide whether future app code serializes calls globally,
  allows limited concurrent calls, or queues calls by feature.
- Cross-call concurrency and serialization policy is deferred to the future
  implementing slice.
- Every request has bounded waiting behavior defined by the future slice.
- UI cancellation must produce an explicit cancelled/failed-closed outcome.
- The exact shutdown escalation policy must be defined and tested by the
  implementing slice.
- No hidden retries or infinite waits.
- UI cancellation must not silently mutate Go state.

The first future slice is read-only, so cancellation cannot leave partial Go
state. Any later mutating slice must define idempotency and rollback semantics
before implementation.

## 11. Packaging And Build Implications

Future implementation must decide how Xcode builds or receives the Go helper
binary. An illustrative packaging shape is a signed helper inside the `.app`
bundle, for example under `Contents/Helpers`, with no hidden launch agent and no
network listener. Phase 8I.1 does not decide the exact bundle location, signing
mechanics, build phase, concurrency model, timeout value or process-termination
implementation.

Phase 8I.1 does not add that build integration. A later milestone must define
the exact helper target, output path, signing behavior, notarization checks,
Apple Silicon build mode, debug logging and cleanup strategy.

## 12. Security And Privacy Implications

The selected bridge preserves local-first boundaries:

- no hidden network listener;
- no cloud service;
- no Facebook credential, cookie or browser-session transfer;
- no direct browser profile access;
- no Swift direct SQLite access;
- no SQL strings over the bridge;
- no secrets in logs;
- stdout reserved for the bounded machine-readable response;
- stderr reserved for bounded, redacted diagnostics;
- diagnostic output must never corrupt stdout response parsing.

The bridge must never log raw private Facebook content unless a later milestone
defines a redaction and consent policy.

## 13. Test Strategy

Future implementation milestones must add tests in layers:

- Go unit tests for the bridge-facing adapter/use case.
- Go serialization/schema tests for request and response structs.
- Swift unit tests for response decoding, error mapping and cancellation
  mapping.
- One narrow integration test for the selected subprocess transport.
- Xcode build.
- Manual app launch for UI slices that expose bridge status.
- Go `go test ./...`.
- Go `go vet ./...`.
- CLI output unchanged unless a future milestone explicitly changes it.

Phase 8I.1 itself requires documentation verification only.

## 14. First Future Integration Slice

Phase 8I.2 should implement one read-only readiness slice:

Swift requests a deterministic Go core readiness value. Go returns a minimal
typed response that proves the helper can start, parse the request, serialize a
response and propagate errors.

The response concept should be no broader than:

- schema_version;
- readiness_status;
- core_identity.

The slice must not read Facebook, open SQLite production paths, mutate settings,
write blocklists, write lead state, load broad lead lists, expose a search API,
or expose SearchProfile, lead data, persistence information, business-rule
summaries, capability inventory or other product/domain data. The exact code
schema remains deferred to the implementing slice.

## 15. Explicit Deferred Work

- Actual helper implementation.
- Exact request/response structs.
- Helper build target and Xcode packaging.
- Production database path policy.
- Bridge observability format.
- Error taxonomy in code.
- Integration tests.
- Any mutating bridge slice.
- Scan orchestration UI wiring.
- Lead list/search loading.
- Facebook adapter integration.
- Signing, notarization and sandbox entitlement decisions.

## 16. Stop Conditions

Stop and request a new decision if a future slice requires:

- hidden networking or a long-lived listener;
- direct Swift SQLite access;
- arbitrary map payloads or generic command bus behavior;
- browser credential/cookie/session transfer;
- broad lead search/list APIs before the readiness slice works;
- mutating persistence before idempotency and cancellation semantics are defined;
- Facebook automation or browser extension behavior.
