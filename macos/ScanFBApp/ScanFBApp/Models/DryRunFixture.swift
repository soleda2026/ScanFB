struct DryRunScreenFixture: Equatable {
    let title: String
    let stateLabel: String
    let disclaimer: String
    let tabs: [DryRunPresentationTab]
    let posts: [DryRunPostFixture]

    static let sample = DryRunScreenFixture(
        title: "Dry Run",
        stateLabel: "Dữ liệu minh họa",
        disclaimer: "Các quyết định bên dưới là dữ liệu mẫu, chưa kết nối Go core hoặc Facebook.",
        tabs: DryRunPresentationTab.allCases,
        posts: [
            DryRunPostFixture(
                id: "post-sample-001",
                author: "Tác giả mẫu 01",
                excerpt: "Cần MacBook Air M2 để học thiết kế, ưu tiên máy nhẹ và pin ổn.",
                category: .included,
                dateLabel: "05/08/2026",
                groupName: "Nhóm Dry Run mẫu A",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-001-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-001-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-002",
                author: "Tác giả mẫu 02",
                excerpt: "Đang tìm MacBook Pro 14-inch đã qua sử dụng cho công việc dựng video.",
                category: .included,
                dateLabel: "05/08/2026",
                groupName: "Nhóm Dry Run mẫu B",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-002-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-002-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-003",
                author: "Tác giả mẫu 03",
                excerpt: "Muốn mua MacBook trong tầm ngân sách cố định, ưu tiên giao dịch tại TP.HCM.",
                category: .included,
                dateLabel: "04/08/2026",
                groupName: "Nhóm Dry Run mẫu C",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-003-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-003-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-004",
                author: "Tác giả mẫu 04",
                excerpt: "Cần tìm MacBook phù hợp nhưng bài mẫu không nêu rõ khu vực giao dịch.",
                category: .review,
                dateLabel: "04/08/2026",
                groupName: "Nhóm Dry Run mẫu D",
                location: "Chưa xác định",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-004-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-004-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-004-unknown-location",
                        code: "review.unknown_location",
                        description: "Cần xem lại vì bằng chứng khu vực chưa rõ."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-005",
                author: "Tác giả mẫu 05",
                excerpt: "Muốn mua MacBook, nội dung mẫu nhắc hai khu vực không thống nhất.",
                category: .review,
                dateLabel: "03/08/2026",
                groupName: "Nhóm Dry Run mẫu E",
                location: "TP.HCM và Hà Nội",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-005-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-005-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-005-location-conflict",
                        code: "review.location_conflict",
                        description: "Cần xem lại vì bằng chứng khu vực mâu thuẫn."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-006",
                author: "Tác giả mẫu 06",
                excerpt: "Bán MacBook Pro cấu hình cao trong bài mẫu, không phải nhu cầu cần mua.",
                category: .excluded,
                dateLabel: "03/08/2026",
                groupName: "Nhóm Dry Run mẫu F",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-006-seller-intent",
                        code: "excluded.seller_intent",
                        description: "Nội dung thể hiện người đăng đang bán sản phẩm."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-007",
                author: "Tác giả mẫu 07",
                excerpt: "Hỏi kinh nghiệm chọn MacBook cho văn phòng nhưng chưa thể hiện nhu cầu mua.",
                category: .excluded,
                dateLabel: "02/08/2026",
                groupName: "Nhóm Dry Run mẫu G",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-007-buyer-intent-missing",
                        code: "excluded.buyer_intent_missing",
                        description: "Không có cụm từ buyer intent đủ rõ."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-008",
                author: "Tác giả mẫu 08",
                excerpt: "Cần MacBook Air cho học tập nhưng bài mẫu nằm ngoài khung thời gian scan.",
                category: .excluded,
                dateLabel: "02/08/2026",
                groupName: "Nhóm Dry Run mẫu H",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-008-previous-day",
                        code: "excluded.previous_day",
                        description: "Bài mẫu nằm trước cửa sổ thời gian của lần scan."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-009",
                author: "Tác giả mẫu 09",
                excerpt: "Cần MacBook cũ nhưng thông tin định danh tác giả trong fixture không đủ tin cậy.",
                category: .excluded,
                dateLabel: "01/08/2026",
                groupName: "Nhóm Dry Run mẫu I",
                location: "TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-009-author-name",
                        code: "excluded.author_name_has_no_space",
                        description: "Tác giả mẫu bị loại vì danh tính không đủ theo rule."
                    ),
                ]
            ),
            DryRunPostFixture(
                id: "post-sample-010",
                author: "Tác giả mẫu 10",
                excerpt: "Muốn mua MacBook nhưng khu vực mẫu nằm ngoài phạm vi địa lý đã chọn.",
                category: .excluded,
                dateLabel: "01/08/2026",
                groupName: "Nhóm Dry Run mẫu J",
                location: "Việt Nam — ngoài TP.HCM",
                reasons: [
                    DryRunReasonFixture(
                        id: "post-sample-010-outside-scope",
                        code: "excluded.outside_scope",
                        description: "Bài mẫu nằm ngoài geographic mode đang chọn."
                    ),
                    DryRunReasonFixture(
                        id: "post-sample-010-hcm-required",
                        code: "excluded.hcm_required_not_matched",
                        description: "Chế độ TP.HCM không khớp với khu vực của bài mẫu."
                    ),
                ]
            ),
        ]
    )

    static let initialTab: DryRunPresentationTab = .included

    var totalPostCount: Int {
        posts.count
    }

    func posts(for tab: DryRunPresentationTab) -> [DryRunPostFixture] {
        posts.filter { $0.category == tab.category }
    }

    func count(for tab: DryRunPresentationTab) -> Int {
        posts(for: tab).count
    }
}

enum DryRunPresentationTab: String, CaseIterable, Identifiable, Equatable {
    case included
    case review
    case excluded

    var id: String {
        rawValue
    }

    var title: String {
        switch self {
        case .included:
            "Được chọn"
        case .review:
            "Cần xem xét"
        case .excluded:
            "Đã loại"
        }
    }

    var category: DryRunPresentationCategory {
        switch self {
        case .included:
            .included
        case .review:
            .review
        case .excluded:
            .excluded
        }
    }
}

enum DryRunPresentationCategory: String, CaseIterable, Equatable {
    case included
    case review
    case excluded

    var title: String {
        switch self {
        case .included:
            "Được chọn"
        case .review:
            "Cần xem xét"
        case .excluded:
            "Đã loại"
        }
    }
}

struct DryRunPostFixture: Identifiable, Equatable {
    let id: String
    let author: String
    let excerpt: String
    let category: DryRunPresentationCategory
    let dateLabel: String
    let groupName: String
    let location: String
    let reasons: [DryRunReasonFixture]
}

struct DryRunReasonFixture: Identifiable, Equatable {
    let id: String
    let code: String
    let description: String
}
