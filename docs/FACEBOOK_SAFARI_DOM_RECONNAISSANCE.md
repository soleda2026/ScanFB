# Facebook Safari DOM Reconnaissance

Date: 2026-08-11

Milestone: Phase 10B2a - one-page live Facebook DOM reconnaissance only.

## Scope and privacy

This reconnaissance used exactly one read-only call through the Phase 10B1
Safari active-tab acquisition boundary. The user had already opened Safari,
logged in, navigated to one Facebook group page and left that tab active. The
acquired absolute HTTPS URL matched the expected prepared group page.

Live source remained outside the repository in a mode-0600 temporary file under
`/tmp` and was deleted after offline analysis. No raw HTML, post body, author,
Facebook identifier, permalink, credential, token, cookie or session value is
included here. Counts below describe only this single page.

The bounded page source was approximately 1.5-1.6 MB, below the Phase 10B1
decoded-content limit of 4 MiB.

## Structural findings

The acquired source behaved as an HTML/bootstrap shell rather than a source
snapshot of the visible group feed:

- 352 `script` elements, 668 `div` elements and 25 anchor elements were present.
- One `role="main"` marker was present.
- No `article` element, `role="article"`, `role="feed"` or `data-pagelet`
  marker was present.
- No group-post URL shape, `/posts/` segment, `post_id` key or `story_fbid` key
  was present.
- No story-message, message-preview, profile-name or machine-readable time
  marker was present.
- Two generic `TextWithEntities` type-name occurrences were present but could
  not be associated with a post, body or author.
- Eleven generic `edges` keys and three generic `cursor` keys were present but
  could not be associated with the visible group feed or post order.
- Bootstrap categories such as scheduled server JavaScript, Relay-prefetched
  data and Comet symbols were present. These are runtime implementation details,
  not credible post selectors.
- No structural login form, checkpoint path or CAPTCHA field marker was found.

## Confidence by concept

| Concept | Confidence | Redacted evidence |
| --- | --- | --- |
| Post/article container | NOT FOUND | Zero article/feed/container semantic markers. |
| Post permalink or post ID | NOT FOUND | Zero group-post URL shapes, `/posts/` segments or post-ID keys. |
| Post body container | NOT FOUND | Zero story-message or message-preview markers; generic text types were not attributable. |
| Author identity/display container | NOT FOUND | Zero profile-name, actor, author, owner or profile-URL markers tied to posts. |
| Created-at timestamp | NOT FOUND | Zero `time`, `datetime`, `data-utime`, creation-time or publish-time markers. |
| Group identity | STRONG at acquisition boundary; NOT FOUND in source | The validated active-tab HTTPS URL identified the expected group page. Source contained no canonical group marker suitable for extraction. |
| Ordering of visible posts | NOT FOUND | No post containers or permalinks existed to establish a source order; generic edges/cursors were not attributable. |

## Marker decisions

The only stable marker category observed was the validated active-tab absolute
HTTPS URL for page-level group identity. No stable post-level marker category
was found.

The following were explicitly rejected as production selector evidence:

- generated or obfuscated CSS classes;
- generic `div` hierarchy, nth-child position or presentation wrappers;
- localized visible text, relative-time text or page-title wording;
- generic React/Comet/bootstrap symbol names;
- unattributed `TextWithEntities`, `edges` or `cursor` keys;
- script ordering or counts.

## Blocker and recommendation

The visible group feed was not represented sufficiently in Safari
`tab.source()` to identify a post container, body, author, permalink or absolute
timestamp fail closed. This meets the Phase 10B2a stop condition for primarily
client-rendered content that is absent from the acquired source.

**Recommendation: stop. Do not proceed to Phase 10B2b production selectors on
the basis of `tab.source()`.** Phase 10B2a is blocked/inconclusive. Any future
attempt would first require a separately approved acquisition-decision milestone
that can obtain a bounded, read-only representation of the live rendered DOM
without weakening ScanFB's privacy and browser-automation boundaries.
