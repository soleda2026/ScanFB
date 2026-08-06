struct LeadsScreenFixture: Equatable {
    let title: String
    let stateLabel: String
    let disclaimer: String
    let tabs: [LeadPresentationTab]
    let leads: [LeadFixture]

    static let sample = LeadsScreenFixture(
        title: "Leads",
        stateLabel: "Dữ liệu minh họa",
        disclaimer: "Các lead bên dưới là dữ liệu mẫu, chưa kết nối Go core hoặc Facebook.",
        tabs: LeadPresentationTab.allCases,
        leads: [
            LeadFixture(
                id: "lead-sample-001",
                displayIdentity: "Người mua mẫu 01",
                excerpt: "Cần MacBook Air M2 để học thiết kế, ưu tiên máy gọn nhẹ và pin tốt.",
                category: .eligible,
                dateLabel: "05/08/2026",
                location: "TP.HCM",
                groupName: "Nhóm công nghệ mẫu A",
                sourcePostCount: 2,
                reasons: [
                    LeadReasonFixture(
                        id: "lead-sample-001-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-001-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                ]
            ),
            LeadFixture(
                id: "lead-sample-002",
                displayIdentity: "Người mua mẫu 02",
                excerpt: "Đang tìm MacBook Pro 14-inch đã qua sử dụng cho công việc dựng nội dung.",
                category: .eligible,
                dateLabel: "04/08/2026",
                location: "TP.HCM",
                groupName: "Cộng đồng thiết bị mẫu B",
                sourcePostCount: 1,
                reasons: [
                    LeadReasonFixture(
                        id: "lead-sample-002-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-002-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                ]
            ),
            LeadFixture(
                id: "lead-sample-003",
                displayIdentity: "Người mua mẫu 03",
                excerpt: "Muốn mua MacBook trong ngân sách cố định, ưu tiên cấu hình ổn cho văn phòng.",
                category: .eligible,
                dateLabel: "03/08/2026",
                location: "Việt Nam — ngoài TP.HCM",
                groupName: "Nhóm trao đổi mẫu C",
                sourcePostCount: 3,
                reasons: [
                    LeadReasonFixture(
                        id: "lead-sample-003-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-003-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-003-duplicate-need",
                        code: "dedup.duplicate_need_matched",
                        description: "Source mẫu được gom vào cùng một nhu cầu."
                    ),
                ]
            ),
            LeadFixture(
                id: "lead-sample-004",
                displayIdentity: "Người mua mẫu 04",
                excerpt: "Cần tìm MacBook phù hợp, nhưng thông tin khu vực trong bài chưa rõ.",
                category: .review,
                dateLabel: "02/08/2026",
                location: "Chưa xác định",
                groupName: "Cộng đồng máy tính mẫu D",
                sourcePostCount: 1,
                reasons: [
                    LeadReasonFixture(
                        id: "lead-sample-004-buyer-intent",
                        code: "included.buyer_intent",
                        description: "Có cụm từ thể hiện nhu cầu mua."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-004-target-keyword",
                        code: "included.target_keyword",
                        description: "Có từ khóa MacBook trong nội dung."
                    ),
                    LeadReasonFixture(
                        id: "lead-sample-004-unknown-location",
                        code: "review.unknown_location",
                        description: "Cần xem lại vì bằng chứng khu vực chưa rõ."
                    ),
                ]
            ),
        ]
    )

    func leads(for tab: LeadPresentationTab) -> [LeadFixture] {
        guard let category = tab.category else {
            return leads
        }
        return leads.filter { $0.category == category }
    }

    func count(for tab: LeadPresentationTab) -> Int {
        leads(for: tab).count
    }
}

enum LeadPresentationTab: String, CaseIterable, Identifiable, Equatable {
    case all
    case eligible
    case review

    var id: String {
        rawValue
    }

    var title: String {
        switch self {
        case .all:
            "Tất cả"
        case .eligible:
            "Đủ điều kiện"
        case .review:
            "Cần xem xét"
        }
    }

    var category: LeadPresentationCategory? {
        switch self {
        case .all:
            nil
        case .eligible:
            .eligible
        case .review:
            .review
        }
    }
}

enum LeadPresentationCategory: String, CaseIterable, Equatable {
    case eligible
    case review

    var title: String {
        switch self {
        case .eligible:
            "Đủ điều kiện"
        case .review:
            "Cần xem xét"
        }
    }
}

struct LeadFixture: Identifiable, Equatable {
    let id: String
    let displayIdentity: String
    let excerpt: String
    let category: LeadPresentationCategory
    let dateLabel: String
    let location: String
    let groupName: String
    let sourcePostCount: Int
    let reasons: [LeadReasonFixture]
}

struct LeadReasonFixture: Identifiable, Equatable {
    let id: String
    let code: String
    let description: String
}
