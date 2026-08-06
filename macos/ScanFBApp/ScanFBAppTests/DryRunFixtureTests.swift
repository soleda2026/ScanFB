import XCTest

final class DryRunFixtureTests: XCTestCase {
    func testFixtureContainsExactlyTenPosts() {
        XCTAssertEqual(DryRunScreenFixture.sample.posts.count, 10)
    }

    func testPostIDsAreUnique() {
        let posts = DryRunScreenFixture.sample.posts

        XCTAssertEqual(Set(posts.map(\.id)).count, posts.count)
    }

    func testPostIDsMatchRequiredStableIDs() {
        XCTAssertEqual(DryRunScreenFixture.sample.posts.map(\.id), [
            "post-sample-001",
            "post-sample-002",
            "post-sample-003",
            "post-sample-004",
            "post-sample-005",
            "post-sample-006",
            "post-sample-007",
            "post-sample-008",
            "post-sample-009",
            "post-sample-010",
        ])
    }

    func testCategoryDistributionIsStable() {
        let posts = DryRunScreenFixture.sample.posts

        XCTAssertEqual(posts.filter { $0.category == .included }.count, 3)
        XCTAssertEqual(posts.filter { $0.category == .review }.count, 2)
        XCTAssertEqual(posts.filter { $0.category == .excluded }.count, 5)
    }

    func testTabCountsMatchFilteredResults() {
        let fixture = DryRunScreenFixture.sample

        for tab in fixture.tabs {
            XCTAssertEqual(fixture.count(for: tab), fixture.posts(for: tab).count)
        }
    }

    func testInitialCategoryIsIncluded() {
        XCTAssertEqual(DryRunScreenFixture.initialTab, .included)
        XCTAssertEqual(DryRunScreenFixture.initialTab.category, .included)
    }

    func testTabsAreExactlyTheRequiredPresentationTabs() {
        XCTAssertEqual(DryRunScreenFixture.sample.tabs, [.included, .review, .excluded])
        XCTAssertEqual(DryRunScreenFixture.sample.tabs.map(\.title), [
            "Được chọn",
            "Cần xem xét",
            "Đã loại",
        ])
    }

    func testStableOrderPerTab() {
        let fixture = DryRunScreenFixture.sample

        XCTAssertEqual(fixture.posts(for: .included).map(\.id), [
            "post-sample-001",
            "post-sample-002",
            "post-sample-003",
        ])
        XCTAssertEqual(fixture.posts(for: .review).map(\.id), [
            "post-sample-004",
            "post-sample-005",
        ])
        XCTAssertEqual(fixture.posts(for: .excluded).map(\.id), [
            "post-sample-006",
            "post-sample-007",
            "post-sample-008",
            "post-sample-009",
            "post-sample-010",
        ])
    }

    func testReturningToACategoryPreservesOrder() {
        let fixture = DryRunScreenFixture.sample
        _ = fixture.posts(for: .review)
        _ = fixture.posts(for: .excluded)

        XCTAssertEqual(fixture.posts(for: .included).map(\.id), [
            "post-sample-001",
            "post-sample-002",
            "post-sample-003",
        ])
    }

    func testVisibleStringsAreNonEmpty() {
        let fixture = DryRunScreenFixture.sample

        XCTAssertFalse(fixture.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertFalse(fixture.stateLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertFalse(fixture.disclaimer.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

        for post in fixture.posts {
            XCTAssertFalse(post.author.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(post.excerpt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(post.category.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(post.dateLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(post.groupName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(post.location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    func testDatesAreDeterministic() {
        XCTAssertEqual(DryRunScreenFixture.sample.posts.map(\.dateLabel), [
            "05/08/2026",
            "05/08/2026",
            "04/08/2026",
            "04/08/2026",
            "03/08/2026",
            "03/08/2026",
            "02/08/2026",
            "02/08/2026",
            "01/08/2026",
            "01/08/2026",
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

    func testEveryPostHasAtLeastOneReason() {
        XCTAssertTrue(DryRunScreenFixture.sample.posts.allSatisfy { !$0.reasons.isEmpty })
    }

    func testReasonIDsAndCodesAreNonEmpty() {
        for reason in DryRunScreenFixture.sample.posts.flatMap(\.reasons) {
            XCTAssertFalse(reason.id.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(reason.code.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    func testOnlyRepositoryExistingReasonCodesAreUsed() {
        let repositoryExistingCodes: Set<String> = [
            "included.buyer_intent",
            "included.target_keyword",
            "review.unknown_location",
            "review.location_conflict",
            "excluded.seller_intent",
            "excluded.buyer_intent_missing",
            "excluded.previous_day",
            "excluded.author_name_has_no_space",
            "excluded.outside_scope",
            "excluded.hcm_required_not_matched",
        ]
        let usedCodes = Set(DryRunScreenFixture.sample.posts.flatMap(\.reasons).map(\.code))

        XCTAssertTrue(usedCodes.isSubset(of: repositoryExistingCodes))
    }

    func testReasonOrderIsDeterministic() {
        XCTAssertEqual(DryRunScreenFixture.sample.posts.map { $0.reasons.map(\.code) }, [
            ["included.buyer_intent", "included.target_keyword"],
            ["included.buyer_intent", "included.target_keyword"],
            ["included.buyer_intent", "included.target_keyword"],
            ["included.buyer_intent", "included.target_keyword", "review.unknown_location"],
            ["included.buyer_intent", "included.target_keyword", "review.location_conflict"],
            ["excluded.seller_intent"],
            ["excluded.buyer_intent_missing"],
            ["excluded.previous_day"],
            ["excluded.author_name_has_no_space"],
            ["excluded.outside_scope", "excluded.hcm_required_not_matched"],
        ])
    }

    func testRepeatedFixtureConstructionIsEqual() {
        XCTAssertEqual(DryRunScreenFixture.sample, DryRunScreenFixture.sample)
    }

    func testFilteringDoesNotInspectReasonStrings() {
        let post = DryRunPostFixture(
            id: "post-sample-test",
            author: "Tác giả mẫu kiểm tra",
            excerpt: "Cần MacBook trong fixture kiểm tra.",
            category: .included,
            dateLabel: "05/08/2026",
            groupName: "Nhóm Dry Run mẫu kiểm tra",
            location: "TP.HCM",
            reasons: [
                DryRunReasonFixture(
                    id: "post-sample-test-excluded-reason",
                    code: "excluded.seller_intent",
                    description: "Reason text cố ý không quyết định tab."
                ),
            ]
        )
        let fixture = DryRunScreenFixture(
            title: "Dry Run",
            stateLabel: "Dữ liệu minh họa",
            disclaimer: "Các quyết định bên dưới là dữ liệu mẫu, chưa kết nối Go core hoặc Facebook.",
            tabs: DryRunPresentationTab.allCases,
            posts: [post]
        )

        XCTAssertEqual(fixture.posts(for: .included).map(\.id), ["post-sample-test"])
        XCTAssertTrue(fixture.posts(for: .excluded).isEmpty)
    }

    func testNoCurrentClockOrRandomValueIsInvolved() {
        XCTAssertEqual(DryRunScreenFixture.sample, DryRunScreenFixture.sample)
        XCTAssertEqual(DryRunScreenFixture.sample.totalPostCount, 10)
    }

    private func allFixtureVisibleStrings() -> [String] {
        let fixture = DryRunScreenFixture.sample
        return [
            fixture.title,
            fixture.stateLabel,
            fixture.disclaimer,
        ] + fixture.tabs.map(\.title) + fixture.posts.flatMap { post in
            [
                post.id,
                post.author,
                post.excerpt,
                post.category.title,
                post.dateLabel,
                post.groupName,
                post.location,
            ] + post.reasons.flatMap { [$0.id, $0.code, $0.description] }
        }
    }
}
