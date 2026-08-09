struct BlocklistScreenFixture: Equatable {
    let title: String
    let stateLabel: String
    let disclaimer: String
    let identityNotice: String
    let unavailableActionMessage: String
    let entries: [BlocklistEntryFixture]

    static let sample = BlocklistScreenFixture(
        title: "Blocklist",
        stateLabel: "Dữ liệu minh họa",
        disclaimer: "Blocklist bên dưới là dữ liệu mẫu, chưa kết nối Go core hoặc Facebook.",
        identityNotice: "Tên hiển thị chỉ là nhãn phụ; không phải định danh block authoritative.",
        unavailableActionMessage: "Chưa khả dụng trong dữ liệu minh họa",
        entries: [
            BlocklistEntryFixture(
                id: "block-sample-001",
                displayLabel: "Danh tính mẫu 01",
                identityKind: .facebookUserID,
                identityValue: "sample-user-id-001",
                reason: BlocklistReasonFixture(
                    id: "block-sample-001-reason",
                    description: "Fixture minh họa một danh tính bị bỏ qua theo Facebook user ID."
                ),
                addedDate: "01/08/2026"
            ),
            BlocklistEntryFixture(
                id: "block-sample-002",
                displayLabel: "Danh tính mẫu 02",
                identityKind: .canonicalProfileURL,
                identityValue: "https://example.invalid/profile/sample-002",
                reason: BlocklistReasonFixture(
                    id: "block-sample-002-reason",
                    description: "Fixture minh họa một danh tính bị bỏ qua theo canonical profile URL."
                ),
                addedDate: "02/08/2026"
            ),
            BlocklistEntryFixture(
                id: "block-sample-003",
                displayLabel: "Danh tính mẫu 03",
                identityKind: .username,
                identityValue: "sample_user_003",
                reason: BlocklistReasonFixture(
                    id: "block-sample-003-reason",
                    description: "Fixture minh họa một danh tính bị bỏ qua theo username."
                ),
                addedDate: "03/08/2026"
            ),
            BlocklistEntryFixture(
                id: "block-sample-004",
                displayLabel: "Danh tính mẫu 04",
                identityKind: .facebookUserID,
                identityValue: "sample-user-id-004",
                reason: BlocklistReasonFixture(
                    id: "block-sample-004-reason",
                    description: "Fixture minh họa một danh tính mẫu khác bị bỏ qua."
                ),
                addedDate: "04/08/2026"
            ),
        ]
    )
}

struct BlocklistEntryFixture: Identifiable, Equatable {
    let id: String
    let displayLabel: String
    let identityKind: BlocklistIdentityKind
    let identityValue: String
    let reason: BlocklistReasonFixture
    let addedDate: String
}

enum BlocklistIdentityKind: String, CaseIterable, Equatable {
    case facebookUserID = "facebook_user_id"
    case canonicalProfileURL = "canonical_profile_url"
    case username

    var title: String {
        switch self {
        case .facebookUserID:
            "Facebook user ID"
        case .canonicalProfileURL:
            "Canonical profile URL"
        case .username:
            "Username"
        }
    }
}

struct BlocklistReasonFixture: Identifiable, Equatable {
    let id: String
    let description: String
}
