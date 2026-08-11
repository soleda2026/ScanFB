# Safari Rendered-DOM Acquisition Decision

Date: 2026-08-11

Milestone: Phase 10B2c - rendered-DOM acquisition decision only.

Outcome: **APPROVE one narrow future mechanism: Safari Apple Events targeting
the current tab and executing a fixed, bounded, read-only page-side JavaScript
expression.** No mechanism is implemented by this milestone.

## Context

Phase 10B1 proved bounded, user-triggered acquisition of the current tab URL,
title and Safari `tab.source()` through direct `/usr/bin/osascript` JXA. Manual
validation succeeded on one user-prepared Facebook group page. Phase 10B2a then
showed that `tab.source()` was bootstrap/source HTML rather than a sufficient
representation of the visible feed: stable post container, body, author,
permalink/ID and absolute timestamp markers were not found. Phase 10B2b
production selector work therefore remains blocked.

This decision evaluates how a later slice could acquire a bounded
representation of the already-rendered DOM from exactly that one active Safari
tab. It does not approve selectors, scan execution or general browser
automation.

## Decision bar

An approved mechanism must be user-triggered, target only one current Safari
tab, avoid login/navigation/scrolling, avoid cookie/session/profile reads,
avoid hidden listeners, return finite output, fail closed, expose an explicit
permission model and remain testable without private Facebook fixtures.

## Candidate comparison

| Candidate | Rendered-DOM capability | Permission and setup | Active-tab/one-shot fit | Boundary result |
| --- | --- | --- | --- | --- |
| A. Safari Apple Events plus page-side JavaScript | Yes. Safari documents that its developer setting allows JavaScript to execute on webpages through AppleScript. A fixed expression can read the live `document` without mutation. | User enables Safari developer features and `Allow JavaScript from Apple Events`; macOS Automation consent applies. Phase 10B2d must validate packaged-process TCC attribution and the final purpose-string, entitlement, signing, Hardened Runtime and App Sandbox configuration. | Strong: target the current tab of the front window once, return a finite result, then exit. | **APPROVE for a separate acquisition-only milestone.** |
| B. Safari Web Inspector or WebDriver | Web Inspector can inspect rendered content manually. WebDriver can automate DOM-capable test sessions. | Web Inspector requires developer features. WebDriver requires remote automation and uses `safaridriver`. | Fails. Apple documents WebDriver as a localhost REST service using isolated automation windows, not the existing normal user tab. Apple does not document a public app API for attaching Web Inspector programmatically to that one tab. | Reject for ScanFB's active-tab path. |
| C. Safari App Extension or Web Extension content script | Yes. Injected/content scripts have webpage access and can read or modify content. | User enables an extension and grants website access; packaging adds an extension target, signing and distribution work. | Technically possible with `activeTab`, but creates a persistent browser component and website-access permission surface. | Reject under the current no-extension boundary. |
| D. Accessibility or System Events UI scripting | Reads an accessibility/UI tree, not a reliable webpage DOM. | Broad Accessibility consent; UI scripting may also involve Automation consent. | Weak and brittle; UI hierarchy and visible controls are presentation state, and the mechanism can control the Mac. | Reject. |
| E. Embedded WebKit or `WKWebView` | Yes, but only for content loaded into the app-owned web view. | App packaging and, for remote pages, network capability; the web view owns a separate website data store. | Fails: it cannot adopt the prepared Safari tab without another login/navigation or session transfer. | Reject. |
| F. Safari profile/session/cache/database scraping | Does not reliably represent the rendered DOM. | File/container or broader disk access may be required. | Fails privacy, session-ownership and stability requirements. | Reject. |
| G. Hidden localhost/network bridge | Not an acquisition mechanism by itself; it transports commands/data for another mechanism. | Opens a local listener and requires protocol/lifecycle security. | Fails the explicit no-listener requirement. WebDriver also depends on this shape. | Reject. |

The following matrix records the remaining decision criteria explicitly. A
candidate described as capable is not thereby approved.

| Candidate | Session and browser mutation | Bounds and isolation | Packaging, sandbox and privacy | Facebook stability and fixture-free testing |
| --- | --- | --- | --- | --- |
| A. Apple Events page-side JavaScript | The approved fixed script reads only the current rendered document. It must not navigate, scroll, mutate, or read cookies, storage, credentials, session stores, profiles, or browser files. | One owned `osascript` process and one finite response are feasible; timeout, cancellation and oversize output fail closed. | Safari's developer setting and Automation consent apply. `NSAppleEventsUsageDescription` is the user-facing purpose string required when an app uses APIs that send Apple Events. Phase 10B2d must validate subprocess TCC attribution and the final entitlement, Hardened Runtime, App Sandbox and signing configuration. Its authority over private rendered content requires strict fixed-script and no-log boundaries. | Facebook DOM changes can invalidate later selectors, so acquisition does not prove selector stability. A fake runner and synthetic DOM can test the transport and bounds without private fixtures. |
| B. Web Inspector or WebDriver | WebDriver operates an automation browsing context rather than adopting the user's prepared normal tab; using it would require another automated browsing path. Web Inspector remains manual under the documented interfaces. | WebDriver has process isolation but requires a local REST service. A bounded response could be imposed, yet the listener and wrong-tab model already fail the decision bar. | Requires Safari developer or remote-automation settings and adds an automation-driver/runtime surface. The isolated context avoids direct profile adoption but cannot satisfy the prepared-session objective. | DOM automation remains selector-sensitive. Synthetic WebDriver pages are testable, but they do not validate access to the required normal active tab. |
| C. Safari extension | A content script can read the page and could be written not to mutate it, but it executes inside the user's browsing context with granted website access. | Messaging can be bounded, but the extension is a persistent browser component rather than one ephemeral owned helper process. | Requires an extension target, website permission, user enablement, signing/distribution and extension sandbox review. It expands access to private page content beyond the current product boundary. | Content scripts remain Facebook-DOM-sensitive. Synthetic pages can test scripts, but extension permission and lifecycle behavior add separate integration work. |
| D. Accessibility or System Events | UI scripting can focus, click, scroll, type, and otherwise control presentation state; the accessible tree is not an authoritative DOM or session store. | Output could be capped and helper failure isolated, but target/UI state is brittle and broad control fails closed only imperfectly. | Requires broad Accessibility consent and potentially Automation consent, with sensitive control authority and associated sandbox/signing review. | Highly sensitive to Safari/Facebook UI and accessibility-tree changes. Synthetic DOM fixtures cannot faithfully test this OS-level route. |
| E. Embedded WebKit | It does not touch the prepared Safari tab unless session or credentials are transferred, which is forbidden; loading the page would introduce navigation and likely network activity in another browser context. | An app-owned web view can bound evaluation, but its renderer is embedded in the app rather than isolated as the required Safari-tab acquisition process. | Adds WebKit/runtime and network/sandbox configuration plus responsibility for a separate website data store. Reusing Safari private session data is outside the supported boundary. | DOM logic remains Facebook-sensitive. Synthetic `WKWebView` content is testable, but it cannot prove access to the user's prepared Safari session. |
| F. Browser-file scraping | Directly touches profile/session/cache/database files and cannot reliably reconstruct the already-rendered document. It need not visibly mutate Safari, but violates private-state boundaries. | File reads could be capped, yet cache/database shape, locking and partial state undermine reliable fail-closed acquisition. | Requires access to protected browser data and creates unacceptable privacy, sandbox and distribution implications. | Private formats are unsupported and unstable. Testing realistically would require forbidden private browser artifacts. |
| G. Localhost/network bridge | A listener neither obtains DOM by itself nor prevents its paired browser-side component from mutation or private-state access. | Protocol limits are possible, but a hidden long-lived listener weakens process ownership and adds network lifecycle/failure modes. | Requires listener protocol security, possible network/sandbox configuration and a larger local attack surface. It is explicitly outside ScanFB's boundary. | It does not reduce selector instability. Transport can be tested synthetically, but the actual acquisition problem remains unsolved. |

## Approved mechanism boundary

The approval is limited to a future acquisition probe with all of these
invariants:

- The user manually opens Safari, logs in, navigates and leaves one desired tab
  active.
- One explicit ScanFB action targets only the current tab of Safari's front
  window.
- ScanFB validates the current absolute HTTPS URL before accepting any result.
- The invoked page-side script is a fixed application resource, not caller
  input or a generic JavaScript command surface.
- The script performs read-only DOM serialization or an equivalently neutral
  structural snapshot. It must contain no click, focus, scroll, navigation,
  reload, form submission or DOM mutation operation.
- The script and host code must not read `document.cookie`, storage APIs,
  IndexedDB, credentials, browser files or Safari profile state.
- The script must not use `fetch`, XMLHttpRequest, WebSocket or another network
  channel.
- The decoded result and stdout transport envelope are finite and explicit;
  oversize output fails closed without truncation.
- One call owns at most one direct `/usr/bin/osascript` process, with explicit
  timeout, cancellation, bounded stderr and no shell or PATH fallback.
- No polling, retry, background monitoring, multi-tab traversal or mass profile
  access is allowed.
- The acquisition result remains page material only. It does not emit
  `RawPost`, call Phase 10A extraction or execute Phase 11.

Approval does not claim that Facebook selectors are stable. A successful
rendered-DOM snapshot must undergo a separate redacted reconnaissance before
any selector implementation can be considered.

## Permission and security model

Apple documents `Allow JavaScript from Apple Events` as a Safari developer
setting that permits JavaScript execution on webpages through AppleScript. The
user must explicitly enable developer features and that setting. macOS
Automation consent separately controls whether ScanFB may send Apple Events to
Safari; denial must fail closed with no fallback.

Apple documents `NSAppleEventsUsageDescription` as the user-facing privacy
purpose string required when an application uses APIs that send Apple Events.
The approved process shape invokes `/usr/bin/osascript` as an owned subprocess,
but Phase 10B2c does not establish whether TCC attributes that packaged path to
the parent app, the subprocess or another signing identity. Phase 10B2d must
test the packaged app and validate the actual TCC/signing attribution before
settling its permission configuration.

Apple documents `com.apple.security.automation.apple-events` as an entitlement
for Apple Events automation under Hardened Runtime. Phase 10B2c does not claim
that the selected subprocess shape categorically requires it. Phase 10B2d must
validate the final Hardened Runtime and App Sandbox configuration; where that
configuration requires the entitlement, it must be added narrowly for Safari
automation. Arbitrary-app automation and broad temporary exceptions remain
unapproved.

Executing code in webpage context is privacy-sensitive because it can see the
rendered private page content. ScanFB reduces that authority through fixed
source, one-shot user action, exact current-tab ownership, URL validation,
prohibited API audits, finite output, local processing and no raw-content logs.
It does not transfer Safari cookies or session state to another browser.

## Process, packaging and crash isolation

The approved shape reuses the Phase 10B1 direct-subprocess boundary:
`/usr/bin/osascript` is one owned process with separate bounded stdout/stderr.
This isolates command-runner failure from the app, although page-side execution
still occurs inside Safari and must be treated as a fail-closed external
dependency. Exact timeout, decoded-output limit, transport envelope and process
shutdown tests belong to the implementing slice.

No extension target, WebKit framework integration, local daemon, socket or
network entitlement is required by the selected mechanism. Distribution
analysis must validate purpose-string applicability, direct `osascript` TCC and
signing attribution, code signing, Hardened Runtime and App Sandbox policy
together. If the validated configuration requires the Apple Events automation
entitlement, it must be scoped narrowly to Safari. This decision does not
change the Xcode project or settle Release signing/notarization.

## Testability

The future slice can be tested without private Facebook fixtures:

- unit-test the fixed script and reject forbidden mutation, storage, cookie and
  network tokens;
- inject a command runner to test exact executable/arguments, timeout,
  cancellation, TCC errors and nonzero exit without launching Safari;
- test strict response parsing, URL validation and byte limits with synthetic
  rendered-DOM fixtures;
- test that malformed, empty and oversized output returns zero result;
- add one separately approved, user-guided live validation that prints only
  redacted structural metadata.

No live Safari test, private HTML fixture or production selector belongs in the
implementing unit-test suite.

## Rejected candidates

- **WebDriver:** Apple documents a localhost REST server and isolated automation
  windows separated from normal browsing data. That cannot inspect the already
  authenticated normal active tab and violates the no-listener boundary.
- **Web Inspector:** useful for manual inspection, but Apple documentation does
  not provide a supported programmatic API for ScanFB to attach to one normal
  active tab and return a bounded payload. Official evidence is insufficient.
- **Safari extensions:** capable, but add persistent browser code, website
  permissions, extension packaging/signing and a broader privacy surface than
  this product currently permits.
- **Accessibility/System Events:** broad control permission, presentation-level
  structure and mutation capability conflict with the product boundary.
- **WKWebView:** rendered DOM belongs to a separate app-owned browsing context
  and data store, not the user's prepared Safari session.
- **Profile/database scraping:** directly violates session/profile boundaries
  and still does not guarantee rendered DOM.
- **Local listeners:** add unnecessary hidden networking and protocol attack
  surface.

## Future milestone

Phase 10B2d may implement only a **bounded Safari active-tab rendered-DOM
acquisition probe through Apple Events page-side JavaScript**. Before editing
code, that milestone must define the exact fixed script, finite decoded and
transport bounds, timeout/cancellation behavior, TCC and Safari-setting errors,
purpose-string applicability, packaged `osascript` TCC/signing attribution,
final Hardened Runtime/App Sandbox policy, any narrowly required Safari
automation entitlement, test seam and user-guided manual validation procedure.

Phase 10B2d must not implement selectors, `RawPost`, Phase 10A automatic
extraction, Phase 11, UI/bridge wiring, persistence or browser mutation. A
successful 10B2d does not automatically unblock selectors; rendered output
must first pass a separate redacted evidence milestone.

## Non-goals and stop conditions

This decision does not enable Safari settings, request TCC permission, launch
Safari, execute JavaScript, add entitlements or modify packaging.

Stop the future implementation if it requires arbitrary script input, cookie or
storage access, session/profile files, a listener, Accessibility, an extension,
WKWebView, navigation, scrolling, mutation, polling, background work, unbounded
output, multiple tabs or private fixtures.

## Official sources

- [Changing Developer settings in Safari on macOS](https://developer.apple.com/documentation/safari-developer-tools/developer-settings)
- [Apple Events Entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.automation.apple-events)
- [`NSAppleEventsUsageDescription`](https://developer.apple.com/documentation/bundleresources/information-property-list/nsappleeventsusagedescription)
- [Allow apps to automate and control other apps](https://support.apple.com/guide/mac-help/mchl108e1718/mac)
- [WebDriver](https://developer.apple.com/documentation/safari-developer-tools/webdriver/)
- [Testing with WebDriver in Safari](https://developer.apple.com/documentation/webkit/testing-with-webdriver-in-safari)
- [Managing Safari web extension permissions](https://developer.apple.com/documentation/safariservices/managing-safari-web-extension-permissions)
- [Injecting a script into a webpage](https://developer.apple.com/documentation/safariservices/injecting-a-script-into-a-webpage)
- [Building a Safari app extension](https://developer.apple.com/documentation/safariservices/building-a-safari-app-extension)
- [Allow accessibility apps to access your Mac](https://support.apple.com/guide/mac-help/mh43185/mac)
- [`AXIsProcessTrustedWithOptions`](https://developer.apple.com/documentation/applicationservices/1459186-axisprocesstrustedwithoptions)
- [`WKWebView`](https://developer.apple.com/documentation/webkit/wkwebview/)
- [`WKWebsiteDataStore`](https://developer.apple.com/documentation/webkit/wkwebsitedatastore)
- [Protecting user data with App Sandbox](https://developer.apple.com/documentation/security/protecting-user-data-with-app-sandbox)
