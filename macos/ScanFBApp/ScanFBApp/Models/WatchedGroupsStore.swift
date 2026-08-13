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

struct PreparedPostDraft: Identifiable, Equatable {
    let id: UUID
    var body: String
    var authorDisplayName: String
    var authorUsername: String
    var authorFacebookUserID: String
    var authorCanonicalProfileURL: String
    var createdAt: Date
    var postURL: String
    var postID: String

    init(id: UUID = UUID(), createdAt: Date) {
        self.id = id
        body = ""
        authorDisplayName = ""
        authorUsername = ""
        authorFacebookUserID = ""
        authorCanonicalProfileURL = ""
        self.createdAt = createdAt
        postURL = ""
        postID = ""
    }
}

@MainActor
final class PreparedGroupScanStore: ObservableObject {
    static let minimumPostCount = 1
    static let maximumPostCount = 100
    static let hoChiMinhTimeZone = TimeZone(identifier: "Asia/Ho_Chi_Minh")!

    @Published var posts: [PreparedPostDraft]
    @Published private(set) var isSubmitting = false
    @Published private(set) var result: PreparedGroupScanBridgeResponse?
    @Published private(set) var errorMessage: String?

    private let client: PreparedGroupScanBridgeClient
    private let idProvider: () -> String
    private let dateProvider: () -> Date
    private var isPresentingLocalValidation = false

    init(
        client: PreparedGroupScanBridgeClient = PreparedGroupScanBridgeClient(),
        idProvider: @escaping () -> String = { UUID().uuidString },
        dateProvider: @escaping () -> Date = Date.init
    ) {
        self.client = client
        self.idProvider = idProvider
        self.dateProvider = dateProvider
        posts = [PreparedPostDraft(createdAt: dateProvider())]
    }

    static func hasActiveGroup(_ groups: [WatchedGroupBridgeValue]) -> Bool {
        groups.contains(where: \.active)
    }

    func beginSession() {
        posts = [PreparedPostDraft(createdAt: dateProvider())]
        result = nil
        errorMessage = nil
        isSubmitting = false
        isPresentingLocalValidation = false
    }

    func addPost() {
        guard posts.count < Self.maximumPostCount else { return }
        posts.append(PreparedPostDraft(createdAt: dateProvider()))
    }

    func removePost(id: UUID) {
        guard posts.count > Self.minimumPostCount else { return }
        posts.removeAll { $0.id == id }
    }

    func validationMessage(for group: WatchedGroupBridgeValue) -> String? {
        guard group.active else {
            return "Nhóm này đang tắt. Hãy bật nhóm trước khi quét."
        }
        guard (Self.minimumPostCount...Self.maximumPostCount).contains(posts.count) else {
            return "Cần nhập từ 1 đến 100 bài viết."
        }
        for (index, post) in posts.enumerated() {
            if post.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                return "Bài \(index + 1) chưa có nội dung."
            }
            let hasAuthor = [
                post.authorDisplayName,
                post.authorUsername,
                post.authorFacebookUserID,
                post.authorCanonicalProfileURL,
            ].contains { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
            if !hasAuthor {
                return "Bài \(index + 1) chưa có tác giả."
            }
        }
        return nil
    }

    func formDidChange(group: WatchedGroupBridgeValue) {
        guard isPresentingLocalValidation else { return }
        let validation = validationMessage(for: group)
        errorMessage = validation
        isPresentingLocalValidation = validation != nil
    }

    func submit(group: WatchedGroupBridgeValue) async {
        guard !isSubmitting else { return }
        if let validation = validationMessage(for: group) {
            errorMessage = validation
            result = nil
            isPresentingLocalValidation = true
            return
        }

        let request = makeRequest(group: group, actionAt: dateProvider())
        isSubmitting = true
        errorMessage = nil
        result = nil
        isPresentingLocalValidation = false
        let bridgeResult = await client.perform(request)
        switch bridgeResult {
        case let .failure(error):
            errorMessage = transportMessage(for: error)
        case let .success(response):
            guard response.status == .ok, response.attemptStatus == "succeeded" else {
                errorMessage = domainMessage(for: response.errorCode)
                isSubmitting = false
                return
            }
            result = response
        }
        isSubmitting = false
    }

    func makeRequest(group: WatchedGroupBridgeValue, actionAt: Date) -> PreparedGroupScanBridgeRequest {
        let values = posts.map { post in
            PreparedSnapshotPostBridgeValue(
                postID: Self.optionalValue(post.postID),
                postURL: Self.optionalValue(post.postURL),
                author: PreparedSnapshotAuthorBridgeValue(
                    facebookUserID: Self.optionalValue(post.authorFacebookUserID),
                    canonicalProfileURL: Self.optionalValue(post.authorCanonicalProfileURL),
                    username: Self.optionalValue(post.authorUsername),
                    displayName: post.authorDisplayName
                ),
                body: post.body,
                createdAt: Self.rfc3339String(from: post.createdAt)
            )
        }
        return PreparedGroupScanBridgeRequest(
            schemaVersion: PreparedGroupScanBridgeClient.schemaVersion,
            operation: .scan,
            groupID: group.id,
            scanID: idProvider(),
            attemptID: idProvider(),
            actionAt: Self.rfc3339String(from: actionAt),
            preparedSnapshot: PreparedSnapshotBridgeValue(
                schemaVersion: PreparedGroupScanBridgeClient.preparedSnapshotSchemaVersion,
                posts: values
            )
        )
    }

    static func rfc3339String(from date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.timeZone = hoChiMinhTimeZone
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }

    private static func optionalValue(_ value: String) -> String? {
        value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : value
    }

    private func domainMessage(for code: PreparedGroupScanBridgeErrorCode?) -> String {
        switch code {
        case .invalidRequest:
            return "Yêu cầu quét không hợp lệ."
        case .groupNotFound:
            return "Không tìm thấy nhóm đã chọn trong dữ liệu đã lưu."
        case .inactiveGroup:
            return "Nhóm đã chọn hiện không hoạt động."
        case .invalidPreparedSnapshot:
            return "Dữ liệu bài viết không hợp lệ. Hãy kiểm tra nội dung, tác giả và thời gian."
        case .storageError:
            return "Không thể đọc dữ liệu nhóm đã lưu."
        case .scanFailed, nil:
            return "Không thể hoàn tất lần quét dữ liệu đã nhập."
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
            return "Không thể gửi dữ liệu quét tới Go core."
        }
    }
}
