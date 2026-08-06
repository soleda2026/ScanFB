struct DashboardFixture: Equatable {
    let title: String
    let stateLabel: String
    let dateLabel: String
    let geographicMode: String
    let searchProfile: String
    let dryRunLabel: String
    let sampleDisclaimer: String
    let groupStatus: BatchStatusSummary
    let primaryMetrics: [MetricSummary]
    let decisionSummary: DecisionSummary
    let exclusionReasons: [ReasonBreakdownItem]

    static let sample = DashboardFixture(
        title: "Batch mẫu — MacBook",
        stateLabel: "Dữ liệu minh họa",
        dateLabel: "05/08/2026",
        geographicMode: "TP.HCM",
        searchProfile: "MacBook",
        dryRunLabel: "Đang bật",
        sampleDisclaimer: "Dashboard này dùng dữ liệu mẫu, không đến từ Facebook và chưa kết nối Go core.",
        groupStatus: BatchStatusSummary(
            successfulGroups: 4,
            failedGroups: 1,
            pendingGroups: 0,
            totalGroups: 5
        ),
        primaryMetrics: [
            MetricSummary(id: "posts-reviewed", title: "Bài đã xem xét", value: 128, symbolName: "doc.text.magnifyingglass"),
            MetricSummary(id: "target-keywords", title: "Bài có từ khóa", value: 37, symbolName: "tag"),
            MetricSummary(id: "matching-leads", title: "Lead phù hợp", value: 12, symbolName: "person.crop.circle.badge.checkmark"),
            MetricSummary(id: "duplicates-merged", title: "Source được gom trùng", value: 5, symbolName: "rectangle.stack.badge.plus"),
        ],
        decisionSummary: DecisionSummary(
            included: 12,
            review: 8,
            excluded: 108
        ),
        exclusionReasons: [
            ReasonBreakdownItem(id: "no-buyer-intent", label: "Không có ý định mua", count: 46),
            ReasonBreakdownItem(id: "seller-or-noise", label: "Bài bán hàng / nhiễu", count: 31),
            ReasonBreakdownItem(id: "outside-geography", label: "Ngoài phạm vi địa lý", count: 18),
            ReasonBreakdownItem(id: "insufficient-author", label: "Tác giả không đủ danh tính", count: 9),
            ReasonBreakdownItem(id: "outside-time-window", label: "Ngoài khung thời gian", count: 4),
        ]
    )

    var postsReviewed: Int {
        primaryMetrics.first { $0.id == "posts-reviewed" }?.value ?? 0
    }
}

struct BatchStatusSummary: Equatable {
    let successfulGroups: Int
    let failedGroups: Int
    let pendingGroups: Int
    let totalGroups: Int
}

struct MetricSummary: Identifiable, Equatable {
    let id: String
    let title: String
    let value: Int
    let symbolName: String
}

struct DecisionSummary: Equatable {
    let included: Int
    let review: Int
    let excluded: Int
}

struct ReasonBreakdownItem: Identifiable, Equatable {
    let id: String
    let label: String
    let count: Int
}
