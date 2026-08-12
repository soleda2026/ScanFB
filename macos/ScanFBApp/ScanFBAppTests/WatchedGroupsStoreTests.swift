import Foundation
import XCTest

@MainActor
final class WatchedGroupsStoreTests: XCTestCase {
    func testBridgeRequestEncodingIsTypedAndDeterministic() throws {
        let request = WatchedGroupsBridgeRequest(
            schemaVersion: 1,
            operation: .list,
            groups: [],
            cursor: 0,
            newGroup: nil,
            groupID: nil,
            active: nil
        )

        let first = try WatchedGroupsBridgeClient.encodeRequest(request)
        let second = try WatchedGroupsBridgeClient.encodeRequest(request)

        XCTAssertEqual(first, second)
        XCTAssertEqual(
            String(data: first, encoding: .utf8),
            #"{"cursor":0,"groups":[],"operation":"watched_groups_list","schema_version":1}"#
        )
    }

    func testBridgeResponseDecodingPreservesSelectionOrder() throws {
        let data = Data(
            #"{"schema_version":1,"operation":"watched_groups_next_five","status":"ok","groups":[],"selection":[{"id":"group-c","name":"Group C","canonical_url":"https://www.facebook.com/groups/group-c","created_at":"2026-08-12T09:00:00+07:00","active":true},{"id":"group-a","name":"Group A","canonical_url":"https://www.facebook.com/groups/group-a","created_at":"2026-08-12T09:00:00+07:00","active":true},{"id":"group-e","name":"Group E","canonical_url":"https://www.facebook.com/groups/group-e","created_at":"2026-08-12T09:00:00+07:00","active":true},{"id":"group-b","name":"Group B","canonical_url":"https://www.facebook.com/groups/group-b","created_at":"2026-08-12T09:00:00+07:00","active":true},{"id":"group-d","name":"Group D","canonical_url":"https://www.facebook.com/groups/group-d","created_at":"2026-08-12T09:00:00+07:00","active":true}],"next_cursor":4}"#.utf8
        )

        let response = try WatchedGroupsBridgeClient.decodeResponse(data, operation: .nextFive).get()

        XCTAssertEqual(response.selection?.map(\.id), ["group-c", "group-a", "group-e", "group-b", "group-d"])
        XCTAssertEqual(response.nextCursor, 4)
    }

    func testEmptyStateNeedsFiveActiveGroups() async {
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: []))],
            .nextFive: [.success(insufficientResponse())],
        ])

        await store.loadIfNeeded()

        XCTAssertTrue(store.groups.isEmpty)
        XCTAssertTrue(store.nextFive.isEmpty)
        XCTAssertTrue(store.needsMoreActiveGroups)
        XCTAssertNil(store.errorMessage)
    }

    func testFewerThanFiveDoesNotExposePartialSelection() async {
        let groups = makeGroups(4)
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: groups))],
            .nextFive: [.success(insufficientResponse())],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.groups, groups)
        XCTAssertTrue(store.nextFive.isEmpty)
        XCTAssertTrue(store.needsMoreActiveGroups)
    }

    func testExactlyFiveDisplaysExactSelection() async {
        let groups = makeGroups(5)
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: groups))],
            .nextFive: [.success(selectionResponse(groups, nextCursor: 0))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.nextFive.map(\.id), groups.map(\.id))
        XCTAssertFalse(store.needsMoreActiveGroups)
    }

    func testMoreThanFivePreservesApplicationSelectionOrder() async {
        let groups = makeGroups(7)
        let selection = [groups[5], groups[6], groups[0], groups[1], groups[2]]
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: groups))],
            .nextFive: [.success(selectionResponse(selection, nextCursor: 3))],
        ])

        await store.loadIfNeeded()

        XCTAssertEqual(store.nextFive.map(\.id), ["group-f", "group-g", "group-a", "group-b", "group-c"])
    }

    func testInactiveGroupPresentationIsPreserved() async {
        var groups = makeGroups(5)
        groups[2] = group("group-c", active: false)
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: groups))],
            .nextFive: [.success(insufficientResponse())],
        ])

        await store.loadIfNeeded()

        XCTAssertFalse(store.groups[2].active)
        XCTAssertEqual(store.groups[2].name, "Group C")
    }

    func testAddGroupValidationErrorIsUnderstandableAndPreservesState() async {
        let groups = makeGroups(2)
        let store = makeStore(
            initialGroups: groups,
            responses: [
                .add: [.success(errorResponse(.add, code: .invalidGroup))],
            ]
        )

        let added = await store.addGroup(name: "", canonicalURL: "not-a-url")

        XCTAssertFalse(added)
        XCTAssertEqual(store.groups, groups)
        XCTAssertEqual(store.errorMessage, "Tên nhóm hoặc URL nhóm không hợp lệ. URL phải dùng HTTPS.")
    }

    func testActivationRefreshesGroupsAndNextFive() async {
        var initialGroups = makeGroups(6)
        initialGroups[1] = group("group-b", active: false)
        let initialSelection = [initialGroups[0], initialGroups[2], initialGroups[3], initialGroups[4], initialGroups[5]]
        let activatedGroups = makeGroups(6)
        let activatedSelection = Array(activatedGroups.prefix(5))
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: initialGroups))],
            .setActive: [.success(collectionResponse(.setActive, groups: activatedGroups))],
            .nextFive: [
                .success(selectionResponse(initialSelection, nextCursor: 0)),
                .success(selectionResponse(activatedSelection, nextCursor: 5)),
            ],
        ])

        await store.loadIfNeeded()
        await store.setActive(true, for: "group-b")

        XCTAssertTrue(store.groups[1].active)
        XCTAssertEqual(store.nextFive.map(\.id), activatedSelection.map(\.id))
    }

    func testDeactivationRefreshesWithoutSwiftReordering() async {
        let initialGroups = makeGroups(6)
        var deactivatedGroups = initialGroups
        deactivatedGroups[1] = group("group-b", active: false)
        let refreshedSelection = [
            deactivatedGroups[5],
            deactivatedGroups[0],
            deactivatedGroups[2],
            deactivatedGroups[3],
            deactivatedGroups[4],
        ]
        let store = makeStore(responses: [
            .list: [.success(collectionResponse(.list, groups: initialGroups))],
            .setActive: [.success(collectionResponse(.setActive, groups: deactivatedGroups))],
            .nextFive: [
                .success(selectionResponse(Array(initialGroups.prefix(5)), nextCursor: 5)),
                .success(selectionResponse(refreshedSelection, nextCursor: 5)),
            ],
        ])

        await store.loadIfNeeded()
        await store.setActive(false, for: "group-b")

        XCTAssertFalse(store.groups[1].active)
        XCTAssertEqual(store.nextFive.map(\.id), ["group-f", "group-a", "group-c", "group-d", "group-e"])
    }

    func testAdvanceUsesReturnedCursorAndPreservesNextResultOrder() async {
        let groups = makeGroups(7)
        let firstSelection = Array(groups.prefix(5))
        let secondSelection = [groups[5], groups[6], groups[0], groups[1], groups[2]]
        let stub = WatchedGroupsBridgeStub(responses: [
            .list: [.success(collectionResponse(.list, groups: groups))],
            .nextFive: [
                .success(selectionResponse(firstSelection, nextCursor: 5)),
                .success(selectionResponse(secondSelection, nextCursor: 3)),
            ],
        ])
        let store = WatchedGroupsStore(client: WatchedGroupsBridgeClient { request in
            await stub.perform(request)
        })

        await store.loadIfNeeded()
        await store.advanceSelection()

        XCTAssertEqual(store.nextFive.map(\.id), secondSelection.map(\.id))
        let requests = await stub.recordedRequests()
        XCTAssertEqual(requests.filter { $0.operation == .nextFive }.map(\.cursor), [0, 5])
    }

    func testAddSuppliesGeneratedIdentityAndTimestampWithoutUserMetadata() async {
        let addedGroup = group("generated-id", active: true)
        let stub = WatchedGroupsBridgeStub(responses: [
            .add: [.success(collectionResponse(.add, groups: [addedGroup]))],
            .nextFive: [.success(insufficientResponse())],
        ])
        let fixedDate = Date(timeIntervalSince1970: 1_786_502_400)
        let store = WatchedGroupsStore(
            client: WatchedGroupsBridgeClient { request in
                await stub.perform(request)
            },
            idProvider: { "generated-id" },
            dateProvider: { fixedDate }
        )

        let added = await store.addGroup(
            name: "Group G",
            canonicalURL: "https://www.facebook.com/groups/generated-id"
        )

        XCTAssertTrue(added)
        let requests = await stub.recordedRequests()
        let request = requests.first { $0.operation == .add }
        XCTAssertEqual(request?.newGroup?.id, "generated-id")
        XCTAssertEqual(request?.newGroup?.name, "Group G")
        XCTAssertEqual(request?.newGroup?.canonicalURL, "https://www.facebook.com/groups/generated-id")
        XCTAssertNotNil(request?.newGroup?.createdAt)
    }

    private func makeStore(
        initialGroups: [WatchedGroupBridgeValue] = [],
        responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]
    ) -> WatchedGroupsStore {
        let stub = WatchedGroupsBridgeStub(responses: responses)
        return WatchedGroupsStore(
            client: WatchedGroupsBridgeClient { request in
                await stub.perform(request)
            },
            initialGroups: initialGroups
        )
    }
}

private actor WatchedGroupsBridgeStub {
    private var responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]
    private var requests: [WatchedGroupsBridgeRequest] = []

    init(responses: [WatchedGroupsBridgeOperation: [Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>]]) {
        self.responses = responses
    }

    func perform(
        _ request: WatchedGroupsBridgeRequest
    ) -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
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

private func collectionResponse(
    _ operation: WatchedGroupsBridgeOperation,
    groups: [WatchedGroupBridgeValue]
) -> WatchedGroupsBridgeResponse {
    WatchedGroupsBridgeResponse(
        schemaVersion: 1,
        operation: operation,
        status: .ok,
        groups: groups,
        selection: nil,
        nextCursor: nil,
        errorCode: nil
    )
}

private func selectionResponse(
    _ selection: [WatchedGroupBridgeValue],
    nextCursor: Int
) -> WatchedGroupsBridgeResponse {
    WatchedGroupsBridgeResponse(
        schemaVersion: 1,
        operation: .nextFive,
        status: .ok,
        groups: [],
        selection: selection,
        nextCursor: nextCursor,
        errorCode: nil
    )
}

private func insufficientResponse() -> WatchedGroupsBridgeResponse {
    errorResponse(.nextFive, code: .insufficientActiveGroups)
}

private func errorResponse(
    _ operation: WatchedGroupsBridgeOperation,
    code: WatchedGroupsBridgeErrorCode
) -> WatchedGroupsBridgeResponse {
    WatchedGroupsBridgeResponse(
        schemaVersion: 1,
        operation: operation,
        status: .error,
        groups: [],
        selection: nil,
        nextCursor: nil,
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
        name: "Group \(suffix)",
        canonicalURL: "https://www.facebook.com/groups/\(id)",
        createdAt: "2026-08-12T09:00:00+07:00",
        active: active
    )
}
