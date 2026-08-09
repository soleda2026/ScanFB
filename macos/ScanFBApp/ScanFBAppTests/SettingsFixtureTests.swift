import XCTest

final class SettingsFixtureTests: XCTestCase {
    func testRequiredFourSectionsExistInStableOrder() {
        XCTAssertEqual(SettingsScreenFixture.sample.sections.map(\.title), [
            "Scan defaults",
            "Privacy",
            "Data and storage",
            "Integration status",
        ])
    }

    func testRequiredRowsExist() {
        let rowIDs = Set(SettingsScreenFixture.sample.sections.flatMap { $0.rows.map(\.id) })

        XCTAssertTrue(rowIDs.isSuperset(of: [
            "search-profile",
            "geographic-mode",
            "maximum-groups-per-batch",
            "dry-run-default",
            "local-first-processing",
            "no-credentials-stored",
            "no-cookies-stored",
            "no-browser-session-data-stored",
            "no-cloud-ai-used",
            "completed-batches",
            "current-screen",
            "swiftui-direct-database-access",
            "go-core-authoritative",
            "go-bridge",
            "facebook-integration",
            "swiftui-persistence-writes",
        ]))
    }

    func testAllLabelsAndValuesAreNonEmpty() {
        for section in SettingsScreenFixture.sample.sections {
            XCTAssertFalse(section.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

            for row in section.rows {
                XCTAssertFalse(row.label.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                XCTAssertFalse(row.value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    func testDryRunDefaultDisplaysEnabled() {
        XCTAssertEqual(rowValue("dry-run-default"), "Bật")
    }

    func testMaximumGroupsDisplaysFive() {
        XCTAssertEqual(rowValue("maximum-groups-per-batch"), "5")
    }

    func testBridgeStatusDisplaysNotChecked() {
        XCTAssertEqual(rowValue("go-bridge"), "Chưa kiểm tra")
    }

    func testFacebookStatusDisplaysNotImplemented() {
        XCTAssertEqual(rowValue("facebook-integration"), "Chưa triển khai")
    }

    func testSwiftUIDirectDatabaseAccessDisplaysNo() {
        XCTAssertEqual(rowValue("swiftui-direct-database-access"), "Không")
    }

    func testNoWritableSettingStateExists() {
        XCTAssertEqual(Set(SettingsValueStyle.allFixtureStyles), [.standard, .badge])
        XCTAssertFalse(allFixtureVisibleStrings().contains("Lưu cài đặt"))
        XCTAssertFalse(allFixtureVisibleStrings().contains("Reset"))
    }

    func testRepeatedFixtureConstructionIsEqual() {
        XCTAssertEqual(SettingsScreenFixture.sample, SettingsScreenFixture.sample)
    }

    func testStableRowOrder() {
        XCTAssertEqual(SettingsScreenFixture.sample.sections.map { $0.rows.map(\.id) }, [
            ["search-profile", "geographic-mode", "maximum-groups-per-batch", "dry-run-default"],
            [
                "local-first-processing",
                "no-credentials-stored",
                "no-cookies-stored",
                "no-browser-session-data-stored",
                "no-cloud-ai-used",
            ],
            ["completed-batches", "current-screen", "swiftui-direct-database-access", "go-core-authoritative"],
            ["go-bridge", "facebook-integration", "swiftui-persistence-writes"],
        ])
    }

    func testSampleLabelAndDisclaimerAreNonEmpty() {
        let fixture = SettingsScreenFixture.sample

        XCTAssertFalse(fixture.stateLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertFalse(fixture.disclaimer.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertTrue(fixture.disclaimer.localizedCaseInsensitiveContains("chỉ đọc"))
    }

    private func rowValue(_ id: String) -> String? {
        SettingsScreenFixture.sample.sections
            .flatMap(\.rows)
            .first { $0.id == id }?
            .value
    }

    private func allFixtureVisibleStrings() -> [String] {
        let fixture = SettingsScreenFixture.sample
        return [
            fixture.title,
            fixture.stateLabel,
            fixture.disclaimer,
        ] + fixture.sections.flatMap { section in
            [section.id, section.title] + section.rows.flatMap { row in
                [row.id, row.label, row.value, row.style.rawValue]
            }
        }
    }
}

private extension SettingsValueStyle {
    static let allFixtureStyles: [SettingsValueStyle] = [.standard, .badge]
}
