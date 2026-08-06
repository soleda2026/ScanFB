import XCTest

final class DashboardFixtureTests: XCTestCase {
    func testFixtureContainsExactlyFiveGroupsInTotal() {
        XCTAssertEqual(DashboardFixture.sample.groupStatus.totalGroups, 5)
    }

    func testGroupCountsSumToTotal() {
        let status = DashboardFixture.sample.groupStatus

        XCTAssertEqual(
            status.successfulGroups + status.failedGroups + status.pendingGroups,
            status.totalGroups
        )
    }

    func testFixtureUsesFixedDeclaredValues() {
        let fixture = DashboardFixture.sample

        XCTAssertEqual(fixture.title, "Batch mẫu — MacBook")
        XCTAssertEqual(fixture.stateLabel, "Dữ liệu minh họa")
        XCTAssertEqual(fixture.dateLabel, "05/08/2026")
        XCTAssertEqual(fixture.geographicMode, "TP.HCM")
        XCTAssertEqual(fixture.searchProfile, "MacBook")
        XCTAssertEqual(fixture.dryRunLabel, "Đang bật")
    }

    func testMetricValuesAreNonNegative() {
        XCTAssertTrue(DashboardFixture.sample.primaryMetrics.allSatisfy { $0.value >= 0 })
    }

    func testDecisionCountsAreNonNegative() {
        let summary = DashboardFixture.sample.decisionSummary

        XCTAssertGreaterThanOrEqual(summary.included, 0)
        XCTAssertGreaterThanOrEqual(summary.review, 0)
        XCTAssertGreaterThanOrEqual(summary.excluded, 0)
    }

    func testDecisionCountsEqualPostsReviewed() {
        let fixture = DashboardFixture.sample
        let summary = fixture.decisionSummary

        XCTAssertEqual(summary.included + summary.review + summary.excluded, fixture.postsReviewed)
    }

    func testExclusionReasonsAreInStableRequiredOrder() {
        XCTAssertEqual(DashboardFixture.sample.exclusionReasons.map(\.label), [
            "Không có ý định mua",
            "Bài bán hàng / nhiễu",
            "Ngoài phạm vi địa lý",
            "Tác giả không đủ danh tính",
            "Ngoài khung thời gian",
        ])
    }

    func testExclusionReasonCountsSumToExcluded() {
        let fixture = DashboardFixture.sample

        XCTAssertEqual(
            fixture.exclusionReasons.map(\.count).reduce(0, +),
            fixture.decisionSummary.excluded
        )
    }

    func testExclusionReasonIDsAreUnique() {
        let reasons = DashboardFixture.sample.exclusionReasons

        XCTAssertEqual(Set(reasons.map(\.id)).count, reasons.count)
    }

    func testMetricIDsAreUnique() {
        let metrics = DashboardFixture.sample.primaryMetrics

        XCTAssertEqual(Set(metrics.map(\.id)).count, metrics.count)
    }

    func testFixtureLabelClearlyIndicatesSampleData() {
        let fixture = DashboardFixture.sample

        XCTAssertTrue(fixture.stateLabel.localizedCaseInsensitiveContains("minh họa"))
        XCTAssertTrue(fixture.sampleDisclaimer.localizedCaseInsensitiveContains("dữ liệu mẫu"))
        XCTAssertTrue(fixture.sampleDisclaimer.localizedCaseInsensitiveContains("không đến từ Facebook"))
    }

    func testFixtureDataIsDeterministicAcrossRepeatedConstruction() {
        XCTAssertEqual(DashboardFixture.sample, DashboardFixture.sample)
    }

    func testNoEmptyUserFacingTitlesExist() {
        let fixture = DashboardFixture.sample
        let strings = [
            fixture.title,
            fixture.stateLabel,
            fixture.dateLabel,
            fixture.geographicMode,
            fixture.searchProfile,
            fixture.dryRunLabel,
            fixture.sampleDisclaimer,
        ] + fixture.primaryMetrics.map(\.title) + fixture.exclusionReasons.map(\.label)

        XCTAssertTrue(strings.allSatisfy { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty })
    }
}
