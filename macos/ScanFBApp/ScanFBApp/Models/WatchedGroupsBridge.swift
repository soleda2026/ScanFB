import Foundation

enum WatchedGroupsBridgeOperation: String, Codable, Equatable, Hashable, Sendable {
    case list = "watched_groups_list"
    case add = "watched_groups_add"
    case setActive = "watched_groups_set_active"
    case nextFive = "watched_groups_next_five"
}

struct WatchedGroupBridgeValue: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let name: String
    let canonicalURL: String
    let createdAt: String
    let active: Bool

    private enum CodingKeys: String, CodingKey {
        case id
        case name
        case canonicalURL = "canonical_url"
        case createdAt = "created_at"
        case active
    }
}

struct AddWatchedGroupBridgeValue: Codable, Equatable, Sendable {
    let id: String
    let name: String
    let canonicalURL: String
    let createdAt: String

    private enum CodingKeys: String, CodingKey {
        case id
        case name
        case canonicalURL = "canonical_url"
        case createdAt = "created_at"
    }
}

struct WatchedGroupsBridgeRequest: Codable, Equatable, Sendable {
    let schemaVersion: Int
    let operation: WatchedGroupsBridgeOperation
    let groups: [WatchedGroupBridgeValue]
    let cursor: Int
    let newGroup: AddWatchedGroupBridgeValue?
    let groupID: String?
    let active: Bool?

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case operation
        case groups
        case cursor
        case newGroup = "new_group"
        case groupID = "group_id"
        case active
    }
}

enum WatchedGroupsBridgeStatus: String, Codable, Equatable, Sendable {
    case ok
    case error
}

enum WatchedGroupsBridgeErrorCode: String, Codable, Equatable, Sendable {
    case invalidGroup = "invalid_group"
    case duplicateGroup = "duplicate_group"
    case groupNotFound = "group_not_found"
    case insufficientActiveGroups = "insufficient_active_groups"
    case invalidCursor = "invalid_cursor"
}

struct WatchedGroupsBridgeResponse: Codable, Equatable, Sendable {
    let schemaVersion: Int
    let operation: WatchedGroupsBridgeOperation
    let status: WatchedGroupsBridgeStatus
    let groups: [WatchedGroupBridgeValue]
    let selection: [WatchedGroupBridgeValue]?
    let nextCursor: Int?
    let errorCode: WatchedGroupsBridgeErrorCode?

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case operation
        case status
        case groups
        case selection
        case nextCursor = "next_cursor"
        case errorCode = "error_code"
    }
}

struct WatchedGroupsBridgeClient: Sendable {
    static let maxResponseBytes = 1024 * 1024

    typealias OperationExecutor = @Sendable (
        WatchedGroupsBridgeRequest
    ) async -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>

    private let operationExecutor: OperationExecutor

    init(
        helperURLProvider: @escaping @Sendable () -> URL? = {
            CoreReadinessBridgeClient.helperURL(inAppBundleURL: Bundle.main.bundleURL)
        },
        helperRunner: @escaping CoreReadinessBridgeClient.HelperRunner = CoreReadinessBridgeClient.runHelperProcess
    ) {
        operationExecutor = { request in
            await Self.execute(
                request,
                helperURLProvider: helperURLProvider,
                helperRunner: helperRunner
            )
        }
    }

    init(operationExecutor: @escaping OperationExecutor) {
        self.operationExecutor = operationExecutor
    }

    func perform(
        _ request: WatchedGroupsBridgeRequest
    ) async -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        await operationExecutor(request)
    }

    static func encodeRequest(_ request: WatchedGroupsBridgeRequest) throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        return try encoder.encode(request)
    }

    static func decodeResponse(
        _ data: Data,
        operation: WatchedGroupsBridgeOperation
    ) -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        do {
            let response = try JSONDecoder().decode(WatchedGroupsBridgeResponse.self, from: data)
            guard response.schemaVersion == CoreReadinessBridgeClient.schemaVersion,
                  response.operation == operation else {
                return .failure(.unsupportedResponseSchema)
            }
            return .success(response)
        } catch {
            return .failure(.malformedResponse)
        }
    }

    private static func execute(
        _ request: WatchedGroupsBridgeRequest,
        helperURLProvider: @Sendable () -> URL?,
        helperRunner: CoreReadinessBridgeClient.HelperRunner
    ) async -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        guard !Task.isCancelled else {
            return .failure(.cancelled)
        }
        guard let helperURL = helperURLProvider() else {
            return .failure(.helperExecutableMissing)
        }

        let requestData: Data
        do {
            requestData = try encodeRequest(request)
        } catch {
            return .failure(.requestEncodingFailed)
        }

        let execution = await helperRunner(
            helperURL,
            requestData,
            CoreReadinessBridgeClient.timeoutSeconds
        )
        switch execution {
        case let .failure(error):
            return .failure(error)
        case let .success(value):
            guard value.stdout.count <= maxResponseBytes else {
                return .failure(.oversizedResponse)
            }
            guard value.exitCode == 0 else {
                return .failure(.nonzeroExit(value.exitCode))
            }
            return decodeResponse(value.stdout, operation: request.operation)
        }
    }
}
