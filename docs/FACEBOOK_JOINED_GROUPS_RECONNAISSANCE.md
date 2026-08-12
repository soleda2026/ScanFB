# Facebook Joined-Groups Reconnaissance

Date: 2026-08-12

Milestone: Phase 9E3a - joined-groups page rendered-DOM reconnaissance only.

Recommendation: **STOP / INCONCLUSIVE.** The current rendered page does not
provide strong enough structure to justify Phase 9E3 joined-group discovery.

## Scope and privacy

The user manually opened Safari, remained authenticated, navigated to a
Facebook page visibly listing joined groups and left that tab active in the
front window. One call to the committed
`AcquireSafariActiveTabRenderedDOM()` API acquired 3,208,287 rendered-DOM
bytes, approximately 3.1 MiB and below the existing 8 MiB decoded bound.

The raw DOM existed only in process memory. A temporary helper wrote only a
1,716-byte typed redacted JSON report under `/tmp` with mode `0600`. No group
name, group ID, full group URL, account identity, cookie, token, session value
or raw DOM entered the repository or command output. There was no retry or
second acquisition.

## Redacted result

The active page was a valid HTTPS Facebook page under the `/groups/...` path.
Together with the user-confirmed visible page, this made it a plausible groups
listing page for this one-page reconnaissance. It did not prove which links
represented joined membership.

| Evidence | Count | Confidence |
| --- | ---: | --- |
| Candidate joined-group section | 1 | TENTATIVE |
| Candidate semantic group item | 0 | NOT_FOUND |
| Candidate canonical group link | 1 | STRONG identity shape |
| Candidate directly numeric Facebook group ID | 0 | NOT_FOUND |
| Candidate with structurally associated display name | 1 | STRONG association |
| Candidate with explicit machine-readable joined membership | 0 | NOT_FOUND |
| Ambiguous or unclassified group link | 1 | NOT_FOUND membership classification |
| Candidate group link inside navigation | 0 | NOT_FOUND |
| Deterministic traversal candidates | 1 | TENTATIVE |

Counts describe only the one user-prepared page state. They are not an account
inventory and must not be interpreted as the number of groups the account has
joined.

## Confidence assessment

| Dimension | Confidence | Assessment |
| --- | --- | --- |
| Joined-group list or section | TENTATIVE | One semantic section contained a candidate group link, but no stable marker attributed that section specifically to joined membership. |
| Individual joined-group item | NOT_FOUND | The candidate link had no semantic `li` or `role=listitem` relationship that could define a fail-closed item boundary. |
| Authoritative group identity | STRONG | One canonical `href=/groups/<group>/` shape was present; no value was retained. |
| Display name association | STRONG | Non-empty name evidence was structurally associated with the same candidate link; the value was not retained. |
| Stable ordering | TENTATIVE | DOM source order was deterministic, but only one candidate existed, so list traversal could not be established strongly. |
| Joined versus recommended or unrelated | NOT_FOUND | No explicit machine-readable membership state or stable section discriminator was found. |

## Marker assessment

Stable categories observed:

- `href=/groups/<group>/`
- same-link name association
- semantic section containing group links
- DOM source order

Rejected unstable categories:

- arbitrary DOM depth
- broad `/groups/` link search
- generated or obfuscated classes
- generic React or Comet symbols
- localized visible text
- nth-child position
- title wording
- transient internal IDs

The canonical URL and display-name association are useful evidence for an
individual link, but they do not establish that the link belongs to the joined
set. Promoting all matching group links would mix joined groups with possible
recommendations, navigation or unrelated content and would violate the
fail-closed requirement.

## Decision

The proceed bar is not met. Joined-group item structure is missing, section
attribution and traversal are only tentative, and joined-versus-recommended
distinction is not found. Phase 9E3 implementation remains blocked.

Do not implement discovery by broad group-link scraping, localized labels,
generated classes, positional selectors or arbitrary ancestor depth. Any
future attempt requires a separately approved evidence or acquisition
milestone that produces strong machine-readable membership and item-boundary
evidence without scrolling, clicking, navigation or private browser-state
access.

## Boundaries preserved

No production discovery/parser/selector, WatchedGroup synchronization,
persistence write, active-state change, cursor advancement, `RawPost`, scan,
Phase 11/12 execution, bridge/UI change, dependency, browser mutation,
network/listener, Accessibility, WebKit, extension, WebDriver or browser
profile/session/cookie access was added.
