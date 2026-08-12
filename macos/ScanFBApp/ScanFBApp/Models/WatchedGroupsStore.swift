import Combine
import Foundation

enum WatchedGroupsLoadState: Equatable {
    case idle
    case loading
    case loaded
    case failed
}

@MainActor
final class WatchedGroupsStore: ObservableObject {
    @Published private(set) var groups: [WatchedGroupBridgeValue] = []
    @Published private(set) var nextFive: [WatchedGroupBridgeValue] = []
    @Published private(set) var loadState: WatchedGroupsLoadState = .idle
    @Published private(set) var isBusy = false
    @Published private(set) var errorMessage: String?

    private let client: WatchedGroupsBridgeClient
    private let idProvider: () -> String
    private let dateProvider: () -> Date

    init(
        client: WatchedGroupsBridgeClient = WatchedGroupsBridgeClient(),
        idProvider: @escaping () -> String = { UUID().uuidString },
        dateProvider: @escaping () -> Date = Date.init
    ) {
        self.client = client
        self.idProvider = idProvider
        self.dateProvider = dateProvider
    }

    var needsMoreActiveGroups: Bool {
        loadState == .loaded && nextFive.isEmpty
    }

    func loadIfNeeded() async {
        guard loadState == .idle else {
            return
        }
        loadState = .loading
        isBusy = true
        errorMessage = nil

        let result = await client.perform(request(operation: .list))
        if applyAuthoritativeResponse(result, operation: .list) {
            loadState = .loaded
        } else {
            loadState = .failed
        }
        isBusy = false
    }

    func addGroup(name: String, canonicalURL: String) async -> Bool {
        guard loadState == .loaded, !isBusy else {
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
        let result = await client.perform(request(operation: .add, newGroup: newGroup))
        let succeeded = applyAuthoritativeResponse(result, operation: .add)
        isBusy = false
        return succeeded
    }

    func setActive(_ active: Bool, for groupID: String) async {
        guard loadState == .loaded, !isBusy else {
            return
        }
        isBusy = true
        errorMessage = nil

        let result = await client.perform(request(
            operation: .setActive,
            groupID: groupID,
            active: active
        ))
        _ = applyAuthoritativeResponse(result, operation: .setActive)
        isBusy = false
    }

    private func applyAuthoritativeResponse(
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
            guard response.selection.isEmpty || response.selection.count == 5 else {
                errorMessage = "Phản hồi chọn nhóm không hợp lệ."
                return false
            }
            groups = response.groups
            nextFive = response.selection
            errorMessage = nil
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
            schemaVersion: WatchedGroupsBridgeClient.schemaVersion,
            operation: operation,
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
            return "Vị trí chọn nhóm đã lưu không hợp lệ."
        case .insufficientActiveGroups:
            return "Cần ít nhất 5 nhóm đang hoạt động."
        case .storageError:
            return "Không thể mở hoặc đọc dữ liệu nhóm đã lưu."
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
