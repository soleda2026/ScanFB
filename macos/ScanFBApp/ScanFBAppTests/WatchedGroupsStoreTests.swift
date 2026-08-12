import Foundation
import XCTest

@MainActor
final class WatchedGroupsStoreTests: XCTestCase {
    func testBridgeRequestEncodingOmitsClientOwnedStateAndIsDeterministic() throws {
        let request = WatchedGroupsBridgeRequest(
            schemaVersion: WatchedGroupsBridgeClient.schemaVersion,
            operation: .list,
            newGroup: nil,
            groupID: nil,
            active: nil
        )

        let first = try WatchedGroupsBridgeClient.encodeRequest(request)
        let second = try WatchedGroupsBridgeClient.encodeRequest(request)

        XCTAssertEqual(first, second)
        XCTAssertEqual(
            String(data: first, encoding: .utf8),
            #"{"operation":"watched_groups_list","schema_version":2}"#
        )
    }

    func testBridgeRequestEncodingRemainsBounded() {
        let request = WatchedGroupsBridgeRequest(
            schemaVersion: WatchedGroupsBridgeClient.schemaVersion,
            operation: .add,
            newGroup: AddWatchedGroupBridgeValue(
                id: "group-a",
                name: String(repeating: "x", count: WatchedGroupsBridgeClient.maxRequestBytes),
                canonicalURL: "https://www.facebook.com/groups/group-a",
                createdAt: "2026-08-12T09:00:00+07:00"
            ),
            groupID: nil,
            active: nil
        )

        XCTAssertThrowsError(try WatchedGroupsBridgeClient.encodeRequest(request))
    }

    func testBridgeResponseDecodingPreservesFullStateSelectionOrderAndCursor() throws {
        let data = Data(
            #"{"schema_version":2,"operation":"watched_groups_next_five","status":"ok","groups":[],"selection":[{"id":"group-c","facebook_group_id":"facebook-c","name":"Group C","canonical_url":"https://www.facebook.com/groups/group-c","created_at":"2026-08-12T09:00:00+07:00","active":false,"notes":"note","last_successful_scan_at":"2026-08-12T10:00:00+07:00","last_error":"error","display_order":3},{"id":"group-a","facebook_group_id":"","name":"Group A","canonical_url":"https://www.facebook.com/groups/group-a","created_at":"2026-08-12T09:00:00+07:00","active":true,"notes":"","last_successful_scan_at":"","last_error":"","display_order":1},{"id":"group-e","facebook_group_id":"","name":"Group E","canonical_url":"https://www.facebook.com/groups/group-e","created_at":"2026-08-12T09:00:00+07:00","active":true,"notes":"","last_successful_scan_at":"","last_error":"","display_order":5},{"id":"group-b","facebook_group_id":"","name":"Group B","canonical_url":"https://www.facebook.com/groups/group-b","created_at":"2026-08-12T09:00:00+07:00","active":true,"notes":"","last_successful_scan_at":"","last_error":"","display_order":2},{"id":"group-d","facebook_group_id":"","name":"Group D","canonical_url":"https://www.facebook.com/groups/group-d","created_at":"2026-08-12T09:00:00+07:00","active":true,"notes":"","last_successful_scan_at":"","last_error":"","display_order":4}],"current_cursor":4}"#.utf8
        )

        let response = try WatchedGroupsBridgeClient.decodeResponse(data, operation: .nextFive).get()

        XCTAssertEqual(response.selection.map(\.id), ["group-c", "group-a", "group-e", "group-b", "group-d"])
        XCTAssertEqual(response.currentCursor, 4)
        XCTAssertEqual(response.selection[0].facebookGroupID, "facebook-c")
        XCTAssertEqual(response.selection[0].notes, "note")
        XCTAssertFalse(response.selection[0].active)
    }

    func testInitialLoadExposesLoadingThenLoadedState() async {
        let bridge = DeferredWatchedGroupsBridge()
        let store = WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in
            await bridge.perform(request)
        })

        let load = Task { await store.loadIfNeeded() }
        await bridge.waitUntilRequested()
        XCTAssertEqual(store.loadState, .loading)
        XCTAssertTrue(store.isBusy)

        await bridge.resume(with: .success(response(.list, groups: [], selection: [], cursor: 0)))
        await load.value

        XCTAssertEqual(store.loadState, .loaded)
        XCTAssertFalse(store.isBusy)
        XCTAssertTrue(store.groups.isEmpty)
    }

    func testRestoreUsesBackendAuthoritativeGroupsActiveStateAndSelection() async {
        var groups = makeGroups(6)
        groups[1] = group("group-b", active: false)
        let selection = [groups[5], groups[0], groups[2], groups[3], groups[4]]
        let store = makeStore(responses: [
            .list: [.success(response(.list, groups: groups, selection: selection, cursor: 5))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.groups, groups)
        XCTAssertFalse(store.groups[1].active)
        XCTAssertEqual(store.nextFive.map(\.id), ["group-f", "group-a", "group-c", "group-d", "group-e"])
    }

    func testEmptyRestoreIsARealLoadedEmptyState() async {
        let store = makeStore(responses: [
            .list: [.success(response(.list, groups: [], selection: [], cursor: 0))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.loadState, .loaded)
        XCTAssertTrue(store.groups.isEmpty)
        XCTAssertTrue(store.needsMoreActiveGroups)
        XCTAssertNil(store.errorMessage)
    }

    func testInsufficientActiveGroupsKeepPreviewEmpty() async {
        let groups = makeGroups(4)
        let store = makeStore(responses: [
            .list: [.success(response(.list, groups: groups, selection: [], cursor: 0))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.groups, groups)
        XCTAssertTrue(store.nextFive.isEmpty)
        XCTAssertTrue(store.needsMoreActiveGroups)
    }

    func testExactlyFiveActiveGroupsPreserveAuthoritativePreviewOrder() async {
        let groups = makeGroups(5)
        let selection = [groups[2], groups[3], groups[4], groups[0], groups[1]]
        let store = makeStore(responses: [
            .list: [.success(response(.list, groups: groups, selection: selection, cursor: 2))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.nextFive.map(\.id), ["group-c", "group-d", "group-e", "group-a", "group-b"])
        XCTAssertFalse(store.needsMoreActiveGroups)
    }

    func testStorageLoadFailureIsVisibleAndNotPresentedAsEmptySuccess() async {
        let store = makeStore(responses: [
            .list: [.success(errorResponse(.list, code: .storageError))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.loadState, .failed)
        XCTAssertFalse(store.needsMoreActiveGroups)
        XCTAssertEqual(store.errorMessage, "Không thể mở hoặc đọc dữ liệu nhóm đã lưu.")
    }

    func testAddAppliesSingleBackendAuthoritativeResponse() async {
        let initial = makeGroups(1)
        let refreshed = makeGroups(2)
        let stub = WatchedGroupsBridgeStub(responses: [
            .list: [.success(response(.list, groups: initial, selection: [], cursor: 0))],
            .add: [.success(response(.add, groups: refreshed, selection: [], cursor: 0))],
        ])
        let store = WatchedGroupsStore(
            client: WatchedGroupsBridgeClient { request in await stub.perform(request) },
            idProvider: { "group-b" },
            dateProvider: { Date(timeIntervalSince1970: 1_786_502_400) }
        )

        await store.loadIfNeeded()
        let added = await store.addGroup(name: "Group B", canonicalURL: "https://www.facebook.com/groups/group-b")

        XCTAssertTrue(added)
        XCTAssertEqual(store.groups, refreshed)
        let requests = await stub.recordedRequests()
        XCTAssertEqual(requests.map(\.operation), [.list, .add])
        XCTAssertEqual(requests[1].newGroup?.id, "group-b")
    }

    func testToggleAppliesSingleBackendAuthoritativeResponse() async {
        let initial = makeGroups(5)
        var refreshed = initial
        refreshed[1] = group("group-b", active: false)
        let stub = WatchedGroupsBridgeStub(responses: [
            .list: [.success(response(.list, groups: initial, selection: initial, cursor: 0))],
            .setActive: [.success(response(.setActive, groups: refreshed, selection: [], cursor: 0))],
        ])
        let store = WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in await stub.perform(request) })

        await store.loadIfNeeded()
        await store.setActive(false, for: "group-b")

        XCTAssertFalse(store.groups[1].active)
        XCTAssertTrue(store.nextFive.isEmpty)
        let requests = await stub.recordedRequests()
        XCTAssertEqual(requests.map(\.operation), [.list, .setActive])
        XCTAssertEqual(requests[1].groupID, "group-b")
        XCTAssertEqual(requests[1].active, false)
    }

    func testRenderingPreviewDoesNotRequestCursorAdvance() async {
        let groups = makeGroups(6)
        let first = Array(groups.prefix(5))
        let stub = WatchedGroupsBridgeStub(responses: [
            .list: [.success(response(.list, groups: groups, selection: first, cursor: 0))],
        ])
        let store = WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in await stub.perform(request) })

        await store.loadIfNeeded()

        XCTAssertEqual(store.nextFive.map(\.id), ["group-a", "group-b", "group-c", "group-d", "group-e"])
        let requests = await stub.recordedRequests()
        XCTAssertEqual(requests.map(\.operation), [.list])
    }

    func testPrimaryGroupsViewShowsEnrollmentAndNoManualQueueAdvance() throws {
        let source = try groupsViewSource()

        XCTAssertTrue(source.contains("AddWatchedGroupSheet"))
        XCTAssertTrue(source.contains("Thêm nhóm theo dõi"))
        XCTAssertFalse(source.contains("Chuyển lượt chọn"))
        XCTAssertFalse(source.contains("advanceSelection"))
    }

    func testPrimaryGroupsViewOmitsAutomaticDiscoveryAndKeepsReadOnlyPreview() throws {
        let source = try groupsViewSource()

        XCTAssertFalse(source.contains("Đồng bộ nhóm đã tham gia"))
        XCTAssertFalse(source.contains("Tính năng đồng bộ chưa khả dụng."))
        XCTAssertTrue(source.contains("Bản xem trước chỉ đọc"))
        XCTAssertTrue(source.contains("Xem trước không đổi lượt."))
    }

    func testPrimaryGroupsViewEmptyStateExplainsOneTimeLocalEnrollment() throws {
        let source = try groupsViewSource()

        XCTAssertTrue(source.contains("Chưa thêm nhóm theo dõi"))
        XCTAssertTrue(source.contains("Mỗi nhóm được lưu cục bộ và chỉ cần thêm một lần."))
    }

    func testEnrollmentSheetExplainsOneTimeSetup() throws {
        let source = try enrollmentSheetSource()

        XCTAssertTrue(source.contains("Thêm nhóm theo dõi"))
        XCTAssertTrue(source.contains("Thêm nhóm một lần bằng URL Facebook."))
        XCTAssertTrue(source.contains("store.addGroup"))
    }

    func testLoadIfNeededDoesNotPollOrReload() async {
        let stub = WatchedGroupsBridgeStub(responses: [
            .list: [.success(response(.list, groups: [], selection: [], cursor: 0))],
        ])
        let store = WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in await stub.perform(request) })

        await store.loadIfNeeded()
        await store.loadIfNeeded()

        let requests = await stub.recordedRequests()
        XCTAssertEqual(requests.map(\.operation), [.list])
    }

    private func makeStore(
        responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]
    ) -> WatchedGroupsStore {
        let stub = WatchedGroupsBridgeStub(responses: responses)
        return WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in
            await stub.perform(request)
        })
    }
}

private func groupsViewSource() throws -> String {
    let projectDirectory = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent()
        .deletingLastPathComponent()
    let sourceURL = projectDirectory
        .appendingPathComponent("ScanFBApp/Views/Groups/WatchedGroupsView.swift")
    return try String(contentsOf: sourceURL, encoding: .utf8)
}

private func enrollmentSheetSource() throws -> String {
    let projectDirectory = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent()
        .deletingLastPathComponent()
    let sourceURL = projectDirectory
        .appendingPathComponent("ScanFBApp/Views/Groups/AddWatchedGroupSheet.swift")
    return try String(contentsOf: sourceURL, encoding: .utf8)
}

private actor WatchedGroupsBridgeStub {
    private var responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]
    private var requests: [WatchedGroupsBridgeRequest] = []

    init(responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]) {
        self.responses = responses
    }

    func perform(_ request: WatchedGroupsBridgeRequest) -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        requests.append(request)
        guard var operationResponses = responses[request.operation], !operationResponses.isEmpty else {
            return .failure(.malformedResponse)
        }
        let response = operationResponses.removeFirst()
        responses[request.operation] = operationResponses
        return response
    }

    func recordedRequests() -> [WatchedGroupsBridgeRequest] {
        requests
    }
}

private actor DeferredWatchedGroupsBridge {
    private var continuation: CheckedContinuation<Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>, Never>?
    private var waiters: [CheckedContinuation<Void, Never>] = []

    func perform(_ request: WatchedGroupsBridgeRequest) async -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        _ = request
        return await withCheckedContinuation { continuation in
            self.continuation = continuation
            waiters.forEach { $0.resume() }
            waiters.removeAll()
        }
    }

    func waitUntilRequested() async {
        guard continuation == nil else { return }
        await withCheckedContinuation { waiters.append($0) }
    }

    func resume(with result: Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>) {
        continuation?.resume(returning: result)
        continuation = nil
    }
}

private func response(
    _ operation: WatchedGroupsBridgeOperation,
    groups: [WatchedGroupBridgeValue],
    selection: [WatchedGroupBridgeValue],
    cursor: Int
) -> WatchedGroupsBridgeResponse {
    WatchedGroupsBridgeResponse(
        schemaVersion: WatchedGroupsBridgeClient.schemaVersion,
        operation: operation,
        status: .ok,
        groups: groups,
        selection: selection,
        currentCursor: cursor,
        errorCode: nil
    )
}

private func errorResponse(
    _ operation: WatchedGroupsBridgeOperation,
    code: WatchedGroupsBridgeErrorCode
) -> WatchedGroupsBridgeResponse {
    WatchedGroupsBridgeResponse(
        schemaVersion: WatchedGroupsBridgeClient.schemaVersion,
        operation: operation,
        status: .error,
        groups: [],
        selection: [],
        currentCursor: 0,
        errorCode: code
    )
}

private func makeGroups(_ count: Int) -> [WatchedGroupBridgeValue] {
    (0..<count).map { index in
        let scalar = UnicodeScalar(97 + index)!
        return group("group-\(Character(scalar))", active: true)
    }
}

private func group(_ id: String, active: Bool) -> WatchedGroupBridgeValue {
    let suffix = id.last.map(String.init)?.uppercased() ?? "?"
    return WatchedGroupBridgeValue(
        id: id,
        facebookGroupID: "",
        name: "Group \(suffix)",
        canonicalURL: "https://www.facebook.com/groups/\(id)",
        createdAt: "2026-08-12T09:00:00+07:00",
        active: active,
        notes: "",
        lastSuccessfulScanAt: "",
        lastError: "",
        displayOrder: 0
    )
}
