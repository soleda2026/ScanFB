import XCTest

final class LeadInteractionStateTests: XCTestCase {
    func testSupportedSessionStatesAreExactlyTheRequiredStates() {
        XCTAssertEqual(LeadInteractionState.allCases, [.new, .viewed, .ignored])
        XCTAssertEqual(LeadInteractionState.allCases.map(\.rawValue), ["new", "viewed", "ignored"])
    }

    func testStatusLabelsAreTheRequiredVietnameseLabels() {
        XCTAssertEqual(LeadInteractionState.allCases.map(\.title), [
            "Mới",
            "Đã xem",
            "Bỏ qua",
        ])
    }

    func testStatusSymbolsAreNonEmptyAndDeterministic() {
        XCTAssertEqual(LeadInteractionState.allCases.map(\.symbolName), [
            "sparkle",
            "eye",
            "hand.raised",
        ])
    }

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

    func testNewCanTransitionToIgnored() {
        var model = freshModel()

        model.markIgnored("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
    }

    func testViewedCanTransitionToIgnored() {
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markIgnored("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
    }

    func testViewedLeadStaysViewedWhenMarkedViewedAgain() {
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markViewed("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .viewed)
    }

    func testIgnoredLeadDoesNotReturnToViewed() {
        var model = freshModel()

        model.markIgnored("lead-sample-001")
        model.markViewed("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
    }

    func testTransitionAffectsOnlySelectedLead() {
        var model = freshModel()

        model.markIgnored("lead-sample-001")

        XCTAssertEqual(model.state(for: "lead-sample-001"), .ignored)
        XCTAssertEqual(model.state(for: "lead-sample-002"), .new)
        XCTAssertEqual(model.state(for: "lead-sample-003"), .new)
        XCTAssertEqual(model.state(for: "lead-sample-004"), .new)
    }

    func testHandoffActionDoesNotMutateNewLeadState() {
        var model = freshModel()
        let lead = LeadsScreenFixture.sample.leads[0]

        XCTAssertTrue(LeadSourceURLHandoff(sourceURLString: lead.sourceURLString).handoff { _ in })

        XCTAssertEqual(model.state(for: lead.id), .new)
    }

    func testHandoffActionDoesNotMutateViewedLeadState() {
        var model = freshModel()
        let lead = LeadsScreenFixture.sample.leads[0]

        model.markViewed(lead.id)
        XCTAssertTrue(LeadSourceURLHandoff(sourceURLString: lead.sourceURLString).handoff { _ in })

        XCTAssertEqual(model.state(for: lead.id), .viewed)
    }

    func testHandoffActionDoesNotMutateIgnoredLeadState() {
        var model = freshModel()
        let lead = LeadsScreenFixture.sample.leads[0]

        model.markIgnored(lead.id)
        XCTAssertTrue(LeadSourceURLHandoff(sourceURLString: lead.sourceURLString).handoff { _ in })

        XCTAssertEqual(model.state(for: lead.id), .ignored)
    }

    func testEligibilityCategoryDoesNotChangeWhenInteractionActionsRun() {
        let before = LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.category.rawValue)" }
        var model = freshModel()

        model.markIgnored("lead-sample-001")
        _ = LeadSourceURLHandoff(sourceURLString: LeadsScreenFixture.sample.leads[3].sourceURLString).handoff { _ in }

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.category.rawValue)" }, before)
    }

    func testReasonCodesDoNotChangeWhenInteractionActionsRun() {
        let before = LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.reasons.map(\.code).joined(separator: ","))" }
        var model = freshModel()

        model.markViewed("lead-sample-001")
        model.markIgnored("lead-sample-004")
        _ = LeadSourceURLHandoff(sourceURLString: LeadsScreenFixture.sample.leads[1].sourceURLString).handoff { _ in }

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map { "\($0.id):\($0.reasons.map(\.code).joined(separator: ","))" }, before)
    }

    func testFixtureOrderDoesNotChangeWhenInteractionActionsRun() {
        let before = leadIDs
        var model = freshModel()

        model.markIgnored("lead-sample-003")
        _ = LeadSourceURLHandoff(sourceURLString: LeadsScreenFixture.sample.leads[0].sourceURLString).handoff { _ in }

        XCTAssertEqual(LeadsScreenFixture.sample.leads.map(\.id), before)
    }

    func testEveryLeadHasDeterministicSourceURLFixture() {
        XCTAssertEqual(LeadsScreenFixture.sample.leads.map(\.sourceURLString), [
            "https://scanfb.invalid/leads/lead-sample-001/source",
            "https://scanfb.invalid/leads/lead-sample-002/source",
            "https://scanfb.invalid/leads/lead-sample-003/source",
            "https://scanfb.invalid/leads/lead-sample-004/source",
        ])
    }

    func testSourceURLFixturesAreStableAcrossRepeatedConstruction() {
        XCTAssertEqual(
            LeadsScreenFixture.sample.leads.map(\.sourceURLString),
            LeadsScreenFixture.sample.leads.map(\.sourceURLString)
        )
    }

    func testMalformedURLIsRejected() {
        XCTAssertNil(LeadSourceURLHandoff.validatedHTTPSURL(from: "not a url"))
        XCTAssertFalse(LeadSourceURLHandoff(sourceURLString: "not a url").canOpen)
    }

    func testNonHTTPSURLIsRejected() {
        XCTAssertNil(LeadSourceURLHandoff.validatedHTTPSURL(from: "http://scanfb.invalid/source"))
        XCTAssertNil(LeadSourceURLHandoff.validatedHTTPSURL(from: "file:///tmp/source"))
        XCTAssertNil(LeadSourceURLHandoff.validatedHTTPSURL(from: "javascript:alert(1)"))
    }

    func testValidHTTPSURLIsAccepted() {
        XCTAssertEqual(
            LeadSourceURLHandoff.validatedHTTPSURL(from: " https://scanfb.invalid/leads/lead-sample-001/source ")?.absoluteString,
            "https://scanfb.invalid/leads/lead-sample-001/source"
        )
    }

    func testURLValidationIsDeterministicAndDoesNotOpenAnything() {
        let values = (0..<5).map { _ in
            LeadSourceURLHandoff.validatedHTTPSURL(from: "https://scanfb.invalid/leads/lead-sample-001/source")?.absoluteString
        }

        XCTAssertEqual(values, Array(repeating: "https://scanfb.invalid/leads/lead-sample-001/source", count: 5))
    }

    func testBrowserHandoffInputIsExactlyTheValidatedURL() {
        let handoff = LeadSourceURLHandoff(sourceURLString: "https://scanfb.invalid/leads/lead-sample-001/source")
        var openedURL: URL?

        XCTAssertTrue(handoff.handoff { openedURL = $0 })

        XCTAssertEqual(openedURL?.absoluteString, "https://scanfb.invalid/leads/lead-sample-001/source")
    }

    func testBrowserHandoffRejectsInvalidURLWithoutCallingOpen() {
        let handoff = LeadSourceURLHandoff(sourceURLString: "scanfb-source")
        var didOpen = false

        XCTAssertFalse(handoff.handoff { _ in didOpen = true })

        XCTAssertFalse(didOpen)
    }

    func testRepeatedFreshConstructionResetsAllStatesToNew() {
        var model = freshModel()
        model.markIgnored("lead-sample-001")

        let fresh = freshModel()

        XCTAssertTrue(leadIDs.allSatisfy { fresh.state(for: $0) == .new })
    }

    func testNoTimestampsOrPersistedIdentifiersAreCreated() {
        let model = freshModel()

        XCTAssertEqual(model, freshModel())
        XCTAssertEqual(model.statesByLeadID.keys.sorted(), leadIDs)
    }

    private var leadIDs: [String] {
        LeadsScreenFixture.sample.leads.map(\.id)
    }

    private func freshModel() -> LeadInteractionStateModel {
        LeadInteractionStateModel(leadIDs: leadIDs)
    }
}
