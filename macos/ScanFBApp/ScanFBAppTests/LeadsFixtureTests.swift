import XCTest

final class LeadsFixtureTests: XCTestCase {
    func testFixtureContainsExactlyFourLeads() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads.count, 4)
    }

    func testLeadIDsAreUnique() {
        let leads = LeadsScreenFixture.sample.leads

        XCTAssertEqual(Set(leads.map(\.id)).count, leads.count)
    }

    func testLeadIDsMatchRequiredStableIDs() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads.map(\.id), [
            "lead-sample-001",
            "lead-sample-002",
            "lead-sample-003",
            "lead-sample-004",
        ])
    }

    func testCategoryDistributionIsBuyerLeadOnly() {
        let leads = LeadsScreenFixture.sample.leads

        XCTAssertEqual(leads.filter { $0.category == .eligible }.count, 3)
        XCTAssertEqual(leads.filter { $0.category == .review }.count, 1)
        XCTAssertEqual(LeadPresentationCategory.allCases, [.eligible, .review])
    }

    func testAllTabReturnsAllLeadsInStableOrder() {
        let fixture = LeadsScreenFixture.sample

        XCTAssertEqual(fixture.leads(for: .all).map(\.id), fixture.leads.map(\.id))
    }

    func testEligibleTabReturnsThreeLeadsInSourceOrder() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads(for: .eligible).map(\.id), [
            "lead-sample-001",
            "lead-sample-002",
            "lead-sample-003",
        ])
    }

    func testReviewTabReturnsOneLead() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads(for: .review).map(\.id), [
            "lead-sample-004",
        ])
    }

    func testTabCountsMatchFilteredResults() {
        let fixture = LeadsScreenFixture.sample

        for tab in fixture.tabs {
            XCTAssertEqual(fixture.count(for: tab), fixture.leads(for: tab).count)
        }
    }

    func testSourceCountsArePositiveAndStable() {
        let sourceCounts = LeadsScreenFixture.sample.leads.map(\.sourcePostCount)

        XCTAssertTrue(sourceCounts.allSatisfy { $0 > 0 })
        XCTAssertEqual(sourceCounts, [2, 1, 3, 1])
    }

    func testVisibleTitlesAndExcerptsAreNonEmpty() {
        for lead in LeadsScreenFixture.sample.leads {
            XCTAssertFalse(lead.displayIdentity.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(lead.excerpt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(lead.groupName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(lead.location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    func testFixedDatesAreNonEmptyAndDeterministic() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads.map(\.dateLabel), [
            "05/08/2026",
            "04/08/2026",
            "03/08/2026",
            "02/08/2026",
        ])
    }

    func testFixtureVisibleStringsContainNoLinks() {
        for value in allFixtureVisibleStrings() {
            XCTAssertFalse(value.localizedCaseInsensitiveContains("http://"))
            XCTAssertFalse(value.localizedCaseInsensitiveContains("https://"))
            XCTAssertFalse(value.localizedCaseInsensitiveContains("www."))
            XCTAssertFalse(value.localizedCaseInsensitiveContains(".com/"))
        }
    }

    func testEveryLeadHasAtLeastOneSuppliedReason() {
        XCTAssertTrue(LeadsScreenFixture.sample.leads.allSatisfy { !$0.reasons.isEmpty })
    }

    func testReasonIDsAndCodesAreNonEmpty() {
        for reason in LeadsScreenFixture.sample.leads.flatMap(\.reasons) {
            XCTAssertFalse(reason.id.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(reason.code.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    func testOnlyRepositoryExistingReasonCodesAreUsed() {
        let repositoryExistingCodes: Set<String> = [
            "included.buyer_intent",
            "included.target_keyword",
            "dedup.duplicate_need_matched",
            "review.unknown_location",
        ]
        let usedCodes = Set(LeadsScreenFixture.sample.leads.flatMap(\.reasons).map(\.code))

        XCTAssertTrue(usedCodes.isSubset(of: repositoryExistingCodes))
    }

    func testReasonOrderIsDeterministic() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads.map { $0.reasons.map(\.code) }, [
            ["included.buyer_intent", "included.target_keyword"],
            ["included.buyer_intent", "included.target_keyword"],
            ["included.buyer_intent", "included.target_keyword", "dedup.duplicate_need_matched"],
            ["included.buyer_intent", "included.target_keyword", "review.unknown_location"],
        ])
    }

    func testRepeatedFixtureConstructionIsEqual() {
        XCTAssertEqual(LeadsScreenFixture.sample, LeadsScreenFixture.sample)
    }

    func testPresentationFilteringDoesNotMutateFixtureOrdering() {
        let fixture = LeadsScreenFixture.sample
        _ = fixture.leads(for: .eligible)
        _ = fixture.leads(for: .review)

        XCTAssertEqual(fixture.leads(for: .all).map(\.id), [
            "lead-sample-001",
            "lead-sample-002",
            "lead-sample-003",
            "lead-sample-004",
        ])
    }

    func testSampleLabelAndDisclaimerAreNonEmpty() {
        let fixture = LeadsScreenFixture.sample

        XCTAssertFalse(fixture.stateLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertFalse(fixture.disclaimer.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
    }

    func testTabsAreExactlyTheRequiredPresentationTabs() {
        XCTAssertEqual(LeadsScreenFixture.sample.tabs, [.all, .eligible, .review])
        XCTAssertEqual(LeadsScreenFixture.sample.tabs.map(\.title), [
            "Tất cả",
            "Đủ điều kiện",
            "Cần xem xét",
        ])
    }

    private func allFixtureVisibleStrings() -> [String] {
        let fixture = LeadsScreenFixture.sample
        return [
            fixture.title,
            fixture.stateLabel,
            fixture.disclaimer,
        ] + fixture.tabs.map(\.title) + fixture.leads.flatMap { lead in
            [
                lead.id,
                lead.displayIdentity,
                lead.excerpt,
                lead.category.title,
                lead.dateLabel,
                lead.location,
                lead.groupName,
            ] + lead.reasons.flatMap { [$0.id, $0.code, $0.description] }
        }
    }
}
