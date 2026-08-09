import Foundation
import Darwin
import SwiftUI
import XCTest

final class CoreReadinessBridgeTests: XCTestCase {
    func testRequestEncodingIsDeterministic() throws {
        let first = try CoreReadinessBridgeClient.encodeRequest(.current)
        let second = try CoreReadinessBridgeClient.encodeRequest(.current)

        XCTAssertEqual(first, second)
        XCTAssertEqual(String(data: first, encoding: .utf8), #"{"operation":"core_readiness","schema_version":1}"#)
    }

    func testValidResponseDecoding() throws {
        let data = Data(#"{"schema_version":1,"readiness_status":"ready","core_identity":"scanfb-core"}"#.utf8)

        let result = CoreReadinessBridgeClient.decodeResponse(data)

        XCTAssertEqual(try result.get(), CoreReadinessBridgeResponse(
            schemaVersion: 1,
            readinessStatus: .ready,
            coreIdentity: "scanfb-core"
        ))
    }

    func testExactStatusEnumMapping() {
        XCTAssertEqual(CoreReadinessStatus.ready.rawValue, "ready")
        XCTAssertEqual(CoreReadinessStatus.error.rawValue, "error")
    }

    func testUnsupportedResponseSchemaRejected() {
        let data = Data(#"{"schema_version":2,"readiness_status":"ready","core_identity":"scanfb-core"}"#.utf8)

        let result = CoreReadinessBridgeClient.decodeResponse(data)

        XCTAssertEqual(result.failure, .unsupportedResponseSchema)
    }

    func testMalformedResponseRejected() {
        let data = Data(#"{"schema_version":1,"readiness_status":"almost","core_identity":"scanfb-core"}"#.utf8)

        let result = CoreReadinessBridgeClient.decodeResponse(data)

        XCTAssertEqual(result.failure, .malformedResponse)
    }

    func testNonzeroExitMapsToExplicitError() async {
        let client = CoreReadinessBridgeClient(helperURLProvider: { URL(fileURLWithPath: "/tmp/helper") }) { _, _, _ in
            .success(CoreReadinessBridgeClient.HelperExecution(
                exitCode: 2,
                stdout: Data(#"{"schema_version":1,"readiness_status":"error","core_identity":"scanfb-core"}"#.utf8),
                stderr: Data("request rejected".utf8)
            ))
        }

        let result = await client.checkReadiness()

        XCTAssertEqual(result.failure, .nonzeroExit(2))
    }

    func testTimeoutMapsToExplicitError() async {
        let client = CoreReadinessBridgeClient(helperURLProvider: { URL(fileURLWithPath: "/tmp/helper") }) { _, _, _ in
            .failure(.timeout)
        }

        let result = await client.checkReadiness()

        XCTAssertEqual(result.failure, .timeout)
    }

    func testCancellationMapsToExplicitError() async {
        let client = CoreReadinessBridgeClient(helperURLProvider: { URL(fileURLWithPath: "/tmp/helper") }) { _, _, _ in
            .failure(.cancelled)
        }

        let result = await client.checkReadiness()

        XCTAssertEqual(result.failure, .cancelled)
    }

    func testHelperPathMissingMapsToExplicitError() async {
        let client = CoreReadinessBridgeClient(helperURLProvider: { nil }) { _, _, _ in
            XCTFail("helper should not launch when path is missing")
            return .failure(.launchFailed)
        }

        let result = await client.checkReadiness()

        XCTAssertEqual(result.failure, .helperExecutableMissing)
    }

    func testRuntimeResolvesExpectedBundleRelativeHelperLocation() throws {
        let appBundleURL = try makeTemporaryAppBundle()
        let helperURL = appBundleURL.appendingPathComponent(CoreReadinessBridgeClient.helperBundleRelativePath)
        try FileManager.default.createDirectory(at: helperURL.deletingLastPathComponent(), withIntermediateDirectories: true)
        FileManager.default.createFile(atPath: helperURL.path, contents: Data())
        chmod(helperURL.path, 0o755)

        XCTAssertEqual(CoreReadinessBridgeClient.helperURL(inAppBundleURL: appBundleURL), helperURL)
    }

    func testMissingBundleRelativeHelperFailsClosed() throws {
        let appBundleURL = try makeTemporaryAppBundle()

        XCTAssertNil(CoreReadinessBridgeClient.helperURL(inAppBundleURL: appBundleURL))
    }

    func testNoPATHSearchOrFallbackExistsForHelperLookup() throws {
        let appBundleURL = try makeTemporaryAppBundle()
        let misplacedHelperURL = appBundleURL.deletingLastPathComponent()
            .appendingPathComponent(CoreReadinessBridgeClient.helperExecutableName)
        FileManager.default.createFile(atPath: misplacedHelperURL.path, contents: Data())
        chmod(misplacedHelperURL.path, 0o755)

        XCTAssertNil(CoreReadinessBridgeClient.helperURL(inAppBundleURL: appBundleURL))
    }

    func testHelperNameIsDeterministic() {
        XCTAssertEqual(CoreReadinessBridgeClient.helperExecutableName, "scanfb-bridge-helper")
        XCTAssertEqual(
            CoreReadinessBridgeClient.helperBundleRelativePath,
            "Contents/Helpers/scanfb-bridge-helper"
        )
    }

    func testStderrContentIsNotSurfacedAsRawDiagnostics() async {
        let client = CoreReadinessBridgeClient(helperURLProvider: { URL(fileURLWithPath: "/tmp/helper") }) { _, _, _ in
            .success(CoreReadinessBridgeClient.HelperExecution(
                exitCode: 2,
                stdout: Data(#"{"schema_version":1,"readiness_status":"error","core_identity":"scanfb-core"}"#.utf8),
                stderr: Data("secret=/Users/mac/private".utf8)
            ))
        }

        let result = await client.checkReadiness()

        XCTAssertEqual(result.failure, .nonzeroExit(2))
        XCTAssertFalse(String(describing: result.failure).contains("/Users/mac/private"))
    }

    func testReadinessActionDoesNotMutateOtherSettingsFixtureState() {
        let before = nonBridgeRows()
        let after = nonBridgeRows()

        XCTAssertEqual(before, after)
    }

    @MainActor
    func testNoAutoRunOnViewCreation() {
        final class Counter: @unchecked Sendable {
            var runCount = 0
        }

        let counter = Counter()
        let client = CoreReadinessBridgeClient(helperURLProvider: { URL(fileURLWithPath: "/tmp/helper") }) { _, _, _ in
            counter.runCount += 1
            return .failure(.launchFailed)
        }

        let view = SettingsFixtureView(bridgeClient: client)
        _ = view.body

        XCTAssertEqual(counter.runCount, 0)
    }

    private func nonBridgeRows() -> [SettingsRowFixture] {
        SettingsScreenFixture.sample.sections
            .flatMap(\.rows)
            .filter { $0.id != "go-bridge" }
    }

    private func makeTemporaryAppBundle() throws -> URL {
        let appBundleURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString)
            .appendingPathComponent("ScanFB.app")
        try FileManager.default.createDirectory(at: appBundleURL, withIntermediateDirectories: true)
        return appBundleURL
    }
}

private extension Result where Failure == CoreReadinessBridgeError {
    var failure: CoreReadinessBridgeError? {
        guard case let .failure(error) = self else {
            return nil
        }
        return error
    }
}
