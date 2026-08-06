import XCTest

final class AppSectionTests: XCTestCase {
    func testAllCasesContainsExactlySixSectionsInStableOrder() {
        XCTAssertEqual(AppSection.allCases, [
            .overview,
            .leads,
            .dryRun,
            .groups,
            .blocklist,
            .settings,
        ])
    }

    func testEverySectionHasNonEmptyVietnameseTitleAndSymbolName() {
        for section in AppSection.allCases {
            XCTAssertFalse(section.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            XCTAssertFalse(section.symbolName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    func testIdentifiersAndTitlesAreUnique() {
        XCTAssertEqual(Set(AppSection.allCases.map(\.id)).count, AppSection.allCases.count)
        XCTAssertEqual(Set(AppSection.allCases.map(\.title)).count, AppSection.allCases.count)
    }

    func testDefaultSectionIsOverview() {
        XCTAssertEqual(AppSection.defaultSection, .overview)
    }
}
