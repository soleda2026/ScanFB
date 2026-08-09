import XCTest

final class LeadInteractionStateTests: XCTestCase {
    func testAllFixtureLeadsStartAsNew() {
        let model = freshModel()

        XCTAssertTrue(leadIDs.allSatisfy { model.state(for: $0) == .new })
    }

    func testLeadIDsRemainStable() {
        XCTAssertEqual(leadIDs, [
            "lead-sample-001",
            "lead-sample-002",
            "lead-sample-003",
            "lead-sample-004",
        ])
    }

    func testNewCanTransitionToViewed() {
        var model = freshModel()

        model.markViewed("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .viewed)
    }

    func testNewCanTransitionToContacted() {
        var model = freshModel()

        model.markContacted("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .contacted)
    }

    func testNewCanTransitionToIgnored() {
        var model = freshModel()

        model.markIgnored("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
    }

    func testViewedCanTransitionToContacted() {
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markContacted("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .contacted)
    }

    func testViewedCanTransitionToIgnored() {
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markIgnored("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
    }

    func testContactedAndIgnoredAreMutuallyExclusiveCurrentStates() {
        var model = freshModel()

        model.markContacted("lead-sample-001")
        XCTAssertEqual(model.state(for: "lead-sample-001"), .contacted)

        model.markIgnored("lead-sample-001")
        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)

        model.markContacted("lead-sample-001")
        XCTAssertEqual(model.state(for: "lead-sample-001"), .contacted)
    }

    func testTransitionAffectsOnlySelectedLead() {
        var model = freshModel()

        model.markContacted("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .contacted)
        XCTAssertEqual(model.state(for: "lead-sample-002"), .new)
        XCTAssertEqual(model.state(for: "lead-sample-003"), .new)
        XCTAssertEqual(model.state(for: "lead-sample-004"), .new)
    }

    func testEligibilityCategoryDoesNotChangeWhenInteractionStateChanges() {
        let before = LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.category.rawValue)" }
        var model = freshModel()

        model.markIgnored("lead-sample-001")
        model.markContacted("lead-sample-004")

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.category.rawValue)" }, before)
    }

    func testReasonCodesDoNotChangeWhenInteractionStateChanges() {
        let before = LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.reasons.map(\.code).joined(separator: ","))" }
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markIgnored("lead-sample-004")

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.reasons.map(\.code).joined(separator: ","))" }, before)
    }

    func testFixtureOrderDoesNotChangeWhenInteractionStateChanges() {
        let before = leadIDs
        var model = freshModel()

        model.markContacted("lead-sample-003")
        model.markIgnored("lead-sample-001")

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map(\.id), before)
    }

    func testRepeatedFreshConstructionResetsAllStatesToNew() {
        var model = freshModel()
        model.markContacted("lead-sample-001")

        let fresh = freshModel()

        XCTAssertTrue(leadIDs.allSatisfy { fresh.state(for: $0) == .new })
    }

    func testNoTimestampsOrPersistedIdentifiersAreCreated() {
        let model = freshModel()

        XCTAssertEqual(model, freshModel())
        XCTAssertEqual(model.statesByLeadID.keys.sorted(), leadIDs)
    }

    func testStatusLabelsAreNonEmptyAndDeterministic() {
        XCTAssertEqual(LeadInteractionState.allCases.map(\.title), [
            "Mới",
            "Đã xem",
            "Đã liên hệ",
            "Bỏ qua",
        ])
        XCTAssertTrue(LeadInteractionState.allCases.allSatisfy {
            !$0.symbolName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        })
    }

    private var leadIDs: [String] {
        LeadsScreenFixture.sample.leads.map(\.id)
    }

    private func freshModel() -> LeadInteractionStateModel {
        LeadInteractionStateModel(leadIDs: leadIDs)
    }
}
