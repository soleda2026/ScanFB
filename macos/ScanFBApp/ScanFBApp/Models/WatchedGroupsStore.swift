import Combine
import Foundation

@MainActor
final class WatchedGroupsStore: ObservableObject {
    @Published private(set) var groups: [WatchedGroupBridgeValue] = []
    @Published private(set) var nextFive: [WatchedGroupBridgeValue] = []
    @Published private(set) var isBusy = false
    @Published private(set) var errorMessage: String?

    private let client: WatchedGroupsBridgeClient
    private let idProvider: () -> String
    private let dateProvider: () -> Date
    private var cursor = 0
    private var nextCursor: Int?
    private var hasLoaded = false

    init(
        client: WatchedGroupsBridgeClient = WatchedGroupsBridgeClient(),
        idProvider: @escaping () -> String = { UUID().uuidString },
        dateProvider: @escaping () -> Date = Date.init,
        initialGroups: [WatchedGroupBridgeValue] = []
    ) {
        self.client = client
        self.idProvider = idProvider
        self.dateProvider = dateProvider
        groups = initialGroups
    }

    var needsMoreActiveGroups: Bool {
        nextFive.isEmpty
    }

    var canAdvanceSelection: Bool {
        nextCursor != nil
    }

    func loadIfNeeded() async {
        guard !hasLoaded else {
            return
        }
        hasLoaded = true
        isBusy = true
        errorMessage = nil

        let response = await client.perform(request(operation: .list))
        guard applyCollectionResponse(response, operation: .list) else {
            isBusy = false
            return
        }
        await refreshSelection()
        isBusy = false
    }

    func addGroup(name: String, canonicalURL: String) async -> Bool {
        guard !isBusy else {
            return false
        }
        isBusy = true
        errorMessage = nil

        let newGroup = AddWatchedGroupBridgeValue(
            id: idProvider(),
            name: name,
            canonicalURL: canonicalURL,
            createdAt: Self.rfc3339String(from: dateProvider())
        )
        let response = await client.perform(request(operation: .add, newGroup: newGroup))
        guard applyCollectionResponse(response, operation: .add) else {
            isBusy = false
            return false
        }

        await refreshSelection()
        isBusy = false
        return true
    }

    func setActive(_ active: Bool, for groupID: String) async {
        guard !isBusy else {
            return
        }
        isBusy = true
        errorMessage = nil

        let response = await client.perform(request(
            operation: .setActive,
            groupID: groupID,
            active: active
        ))
        guard applyCollectionResponse(response, operation: .setActive) else {
            isBusy = false
            return
        }

        await refreshSelection()
        isBusy = false
    }

    func advanceSelection() async {
        guard !isBusy, let nextCursor else {
            return
        }
        cursor = nextCursor
        isBusy = true
        errorMessage = nil
        await refreshSelection()
        isBusy = false
    }

    private func refreshSelection() async {
        let result = await client.perform(request(operation: .nextFive))
        switch result {
        case let .failure(error):
            nextFive = []
            nextCursor = nil
            errorMessage = transportMessage(for: error)
        case let .success(response):
            guard response.operation == .nextFive else {
                nextFive = []
                nextCursor = nil
                errorMessage = "Phản hồi chọn nhóm không hợp lệ."
                return
            }
            if response.status == .ok,
               let selection = response.selection,
               selection.count == 5,
               let returnedCursor = response.nextCursor {
                nextFive = selection
                nextCursor = returnedCursor
                errorMessage = nil
                return
            }
            nextFive = []
            nextCursor = nil
            if response.errorCode == .insufficientActiveGroups {
                errorMessage = nil
            } else {
                errorMessage = domainMessage(for: response.errorCode)
            }
        }
    }

    private func applyCollectionResponse(
        _ result: Result<WatchedGroupsBridgeResponse, CoreReadinessBridgeError>,
        operation: WatchedGroupsBridgeOperation
    ) -> Bool {
        switch result {
        case let .failure(error):
            errorMessage = transportMessage(for: error)
            return false
        case let .success(response):
            guard response.operation == operation else {
                errorMessage = "Phản hồi quản lý nhóm không hợp lệ."
                return false
            }
            guard response.status == .ok else {
                errorMessage = domainMessage(for: response.errorCode)
                return false
            }
            groups = response.groups
            if groups.isEmpty {
                cursor = 0
            } else if cursor >= groups.count {
                cursor = 0
            }
            return true
        }
    }

    private func request(
        operation: WatchedGroupsBridgeOperation,
        newGroup: AddWatchedGroupBridgeValue? = nil,
        groupID: String? = nil,
        active: Bool? = nil
    ) -> WatchedGroupsBridgeRequest {
        WatchedGroupsBridgeRequest(
            schemaVersion: CoreReadinessBridgeClient.schemaVersion,
            operation: operation,
            groups: groups,
            cursor: cursor,
            newGroup: newGroup,
            groupID: groupID,
            active: active
        )
    }

    private func domainMessage(for code: WatchedGroupsBridgeErrorCode?) -> String {
        switch code {
        case .invalidGroup:
            return "Tên nhóm hoặc URL nhóm không hợp lệ. URL phải dùng HTTPS."
        case .duplicateGroup:
            return "Nhóm này đã có trong danh sách theo dõi."
        case .groupNotFound:
            return "Không tìm thấy nhóm cần cập nhật."
        case .invalidCursor:
            return "Vị trí chọn nhóm không còn hợp lệ."
        case .insufficientActiveGroups:
            return "Cần ít nhất 5 nhóm đang hoạt động."
        case nil:
            return "Go core từ chối thao tác quản lý nhóm."
        }
    }

    private func transportMessage(for error: CoreReadinessBridgeError) -> String {
        switch error {
        case .helperExecutableMissing:
            return "Không tìm thấy Go helper trong ứng dụng."
        case .timeout:
            return "Go helper phản hồi quá thời gian cho phép."
        case .cancelled:
            return "Thao tác đã bị hủy."
        default:
            return "Không thể hoàn tất thao tác với Go core."
        }
    }

    private static func rfc3339String(from date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }

}
