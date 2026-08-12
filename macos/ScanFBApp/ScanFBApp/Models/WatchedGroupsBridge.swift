import Foundation

enum WatchedGroupsBridgeOperation: String, Codable, Equatable, Hashable, Sendable {
    case list = "watched_groups_list"
    case add = "watched_groups_add"
    case setActive = "watched_groups_set_active"
    case nextFive = "watched_groups_next_five"
}

struct WatchedGroupBridgeValue: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let facebookGroupID: String
    let name: String
    let canonicalURL: String
    let createdAt: String
    let active: Bool
    let notes: String
    let lastSuccessfulScanAt: String
    let lastError: String
    let displayOrder: Int

    private enum CodingKeys: String, CodingKey {
        case id
        case facebookGroupID = "facebook_group_id"
        case name
        case canonicalURL = "canonical_url"
        case createdAt = "created_at"
        case active
        case notes
        case lastSuccessfulScanAt = "last_successful_scan_at"
        case lastError = "last_error"
        case displayOrder = "display_order"
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
    let newGroup: AddWatchedGroupBridgeValue?
    let groupID: String?
    let active: Bool?

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case operation
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
    case storageError = "storage_error"
}

struct WatchedGroupsBridgeResponse: Codable, Equatable, Sendable {
    let schemaVersion: Int
    let operation: WatchedGroupsBridgeOperation
    let status: WatchedGroupsBridgeStatus
    let groups: [WatchedGroupBridgeValue]
    let selection: [WatchedGroupBridgeValue]
    let currentCursor: Int
    let errorCode: WatchedGroupsBridgeErrorCode?

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case operation
        case status
        case groups
        case selection
        case currentCursor = "current_cursor"
        case errorCode = "error_code"
    }
}

struct WatchedGroupsBridgeClient: Sendable {
    static let schemaVersion = 2
    static let maxRequestBytes = 64 * 1024
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
        let data = try encoder.encode(request)
        guard data.count <= maxRequestBytes else {
            throw WatchedGroupsBridgeEncodingError.oversizedRequest
        }
        return data
    }

    static func decodeResponse(
        _ data: Data,
        operation: WatchedGroupsBridgeOperation
    ) -> Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError> {
        do {
            let response = try JSONDecoder().decode(WatchedGroupsBridgeResponse.self, from: data)
            guard response.schemaVersion == schemaVersion,
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

private enum WatchedGroupsBridgeEncodingError: Error {
    case oversizedRequest
}
