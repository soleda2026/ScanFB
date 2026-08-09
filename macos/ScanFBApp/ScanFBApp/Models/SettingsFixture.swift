struct SettingsScreenFixture: Equatable {
    let title: String
    let stateLabel: String
    let disclaimer: String
    let sections: [SettingsSectionFixture]

    static let sample = SettingsScreenFixture(
        title: "Cài đặt",
        stateLabel: "Dữ liệu minh họa",
        disclaimer: "Các cài đặt bên dưới là dữ liệu mẫu, chỉ đọc và chưa kết nối Go core hoặc Facebook.",
        sections: [
            SettingsSectionFixture(
                id: "scan-defaults",
                title: "Scan defaults",
                rows: [
                    SettingsRowFixture(id: "search-profile", label: "Search profile", value: "MacBook", style: .standard),
                    SettingsRowFixture(id: "geographic-mode", label: "Geographic mode", value: "TP.HCM", style: .standard),
                    SettingsRowFixture(id: "maximum-groups-per-batch", label: "Maximum groups per batch", value: "5", style: .badge),
                    SettingsRowFixture(id: "dry-run-default", label: "Dry Run default", value: "Bật", style: .badge),
                ]
            ),
            SettingsSectionFixture(
                id: "privacy",
                title: "Privacy",
                rows: [
                    SettingsRowFixture(id: "local-first-processing", label: "Local-first processing", value: "Bật", style: .badge),
                    SettingsRowFixture(id: "no-credentials-stored", label: "No credentials stored", value: "Không lưu", style: .standard),
                    SettingsRowFixture(id: "no-cookies-stored", label: "No cookies stored", value: "Không lưu", style: .standard),
                    SettingsRowFixture(id: "no-browser-session-data-stored", label: "No browser session data stored", value: "Không lưu", style: .standard),
                    SettingsRowFixture(id: "no-cloud-ai-used", label: "No cloud AI used", value: "Không sử dụng", style: .standard),
                ]
            ),
            SettingsSectionFixture(
                id: "data-and-storage",
                title: "Data and storage",
                rows: [
                    SettingsRowFixture(id: "completed-batches", label: "Completed batches", value: "Khái niệm local-only", style: .standard),
                    SettingsRowFixture(id: "current-screen", label: "Current screen", value: "Dữ liệu mẫu", style: .badge),
                    SettingsRowFixture(id: "swiftui-direct-database-access", label: "Direct SQL" + "ite access from SwiftUI", value: "Không", style: .badge),
                    SettingsRowFixture(id: "go-core-authoritative", label: "Go core remains authoritative", value: "Có", style: .badge),
                ]
            ),
            SettingsSectionFixture(
                id: "integration-status",
                title: "Integration status",
                rows: [
                    SettingsRowFixture(id: "go-bridge", label: "Go bridge", value: "Chưa kết nối", style: .badge),
                    SettingsRowFixture(id: "facebook-integration", label: "Facebook integration", value: "Chưa triển khai", style: .badge),
                    SettingsRowFixture(id: "swiftui-persistence-writes", label: "Persistence writes from SwiftUI", value: "Không", style: .badge),
                ]
            ),
        ]
    )
}

struct SettingsSectionFixture: Identifiable, Equatable {
    let id: String
    let title: String
    let rows: [SettingsRowFixture]
}

struct SettingsRowFixture: Identifiable, Equatable {
    let id: String
    let label: String
    let value: String
    let style: SettingsValueStyle
}

enum SettingsValueStyle: String, Equatable {
    case standard
    case badge
}
