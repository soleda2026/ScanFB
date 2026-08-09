import XCTest

final class BlocklistFixtureTests: XCTestCase {
    func testFixtureContainsExactlyFourEntries() {
        XCTAssertEqual(BlocklistScreenFixture.sample.entries.count, 4)
    }

    func testEntryIDsAreUnique() {
        let entries = BlocklistScreenFixture.sample.entries

        XCTAssertEqual(Set(entries.map(\.id)).count, entries.count)
    }

    func testEntryIDsMatchRequiredStableIDs() {
        XCTAssertEqual(BlocklistScreenFixture.sample.entries.map(\.id), [
            "block-sample-001",
            "block-sample-002",
            "block-sample-003",
            "block-sample-004",
        ])
    }

    func testOnlySupportedIdentityKindsAreUsed() {
        let supportedKinds: Set<BlocklistIdentityKind> = [
            .facebookUserID,
            .canonicalProfileURL,
            .username,
        ]
        let usedKinds = Set(BlocklistScreenFixture.sample.entries.map(\.identityKind))

        XCTAssertTrue(usedKinds.isSubset(of: supportedKinds))
        XCTAssertEqual(Set(BlocklistIdentityKind.allCases), supportedKinds)
    }

    func testNoDisplayNameOnlyIdentityKindExists() {
        XCTAssertFalse(BlocklistIdentityKind.allCases.map(\.rawValue).contains("display_name"))
        XCTAssertTrue(BlocklistScreenFixture.sample.identityNotice.localizedCaseInsensitiveContains("không phải định danh"))
    }

    func testNoEmptyIdentityValueExists() {
        XCTAssertTrue(BlocklistScreenFixture.sample.entries.allSatisfy {
            !$0.identityValue.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        })
    }

    func testDatesAreDeterministic() {
        XCTAssertEqual(BlocklistScreenFixture.sample.entries.map(\.addedDate), [
            "01/08/2026",
            "02/08/2026",
            "03/08/2026",
            "04/08/2026",
        ])
    }

    func testReasonsAreNonEmpty() {
        XCTAssertTrue(BlocklistScreenFixture.sample.entries.allSatisfy {
            !$0.reason.description.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        })
    }

    func testInvalidURLsAreNonClickableFixtureStrings() {
        let urlValues = BlocklistScreenFixture.sample.entries
            .filter { $0.identityKind == .canonicalProfileURL }
            .map(\.identityValue)

        XCTAssertEqual(urlValues, ["https://example.invalid/profile/sample-002"])
        XCTAssertTrue(urlValues.allSatisfy { $0.contains(".invalid") })
    }

    func testNoRealFacebookDomainIsUsed() {
        for value in allFixtureVisibleStrings() {
            XCTAssertFalse(value.localizedCaseInsensitiveContains("facebook.com"))
            XCTAssertFalse(value.localizedCaseInsensitiveContains("fb.com"))
        }
    }

    func testRepeatedFixtureConstructionIsEqual() {
        XCTAssertEqual(BlocklistScreenFixture.sample, BlocklistScreenFixture.sample)
    }

    func testStableEntryOrder() {
        XCTAssertEqual(BlocklistScreenFixture.sample.entries.map(\.displayLabel), [
            "Danh tính mẫu 01",
            "Danh tính mẫu 02",
            "Danh tính mẫu 03",
            "Danh tính mẫu 04",
        ])
    }

    func testSampleLabelAndDisclaimerAreNonEmpty() {
        let fixture = BlocklistScreenFixture.sample

        XCTAssertFalse(fixture.stateLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertFalse(fixture.disclaimer.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        XCTAssertTrue(fixture.stateLabel.localizedCaseInsensitiveContains("minh họa"))
    }

    private func allFixtureVisibleStrings() -> [String] {
        let fixture = BlocklistScreenFixture.sample
        return [
            fixture.title,
            fixture.stateLabel,
            fixture.disclaimer,
            fixture.identityNotice,
            fixture.unavailableActionMessage,
        ] + fixture.entries.flatMap { entry in
            [
                entry.id,
                entry.displayLabel,
                entry.identityKind.title,
                entry.identityKind.rawValue,
                entry.identityValue,
                entry.reason.id,
                entry.reason.description,
                entry.addedDate,
            ]
        }
    }
}
