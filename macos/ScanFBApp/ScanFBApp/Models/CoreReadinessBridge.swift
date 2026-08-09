import Darwin
import Foundation

enum CoreReadinessBridgeOperation: String, Codable, Equatable {
    case coreReadiness = "core_readiness"
}

struct CoreReadinessBridgeRequest: Codable, Equatable {
    let schemaVersion: Int
    let operation: CoreReadinessBridgeOperation

    static let current = CoreReadinessBridgeRequest(
        schemaVersion: CoreReadinessBridgeClient.schemaVersion,
        operation: .coreReadiness
    )

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case operation
    }
}

enum CoreReadinessStatus: String, Codable, Equatable {
    case ready
    case error
}

struct CoreReadinessBridgeResponse: Codable, Equatable {
    let schemaVersion: Int
    let readinessStatus: CoreReadinessStatus
    let coreIdentity: String

    private enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case readinessStatus = "readiness_status"
        case coreIdentity = "core_identity"
    }
}

enum CoreReadinessBridgeError: Error, Equatable, CustomStringConvertible {
    case helperExecutableMissing
    case launchFailed
    case malformedResponse
    case unsupportedResponseSchema
    case nonzeroExit(Int32)
    case timeout
    case cancelled
    case requestEncodingFailed
    case oversizedResponse

    var description: String {
        switch self {
        case .helperExecutableMissing:
            return "helper executable missing"
        case .launchFailed:
            return "helper launch failed"
        case .malformedResponse:
            return "malformed response"
        case .unsupportedResponseSchema:
            return "unsupported response schema"
        case let .nonzeroExit(code):
            return "nonzero helper exit \(code)"
        case .timeout:
            return "helper timeout"
        case .cancelled:
            return "bridge call cancelled"
        case .requestEncodingFailed:
            return "request encoding failed"
        case .oversizedResponse:
            return "oversized response"
        }
    }
}

enum CoreReadinessDisplayStatus: Equatable {
    case notChecked
    case checking
    case ready
    case failed

    var label: String {
        switch self {
        case .notChecked:
            return "Chưa kiểm tra"
        case .checking:
            return "Đang kiểm tra"
        case .ready:
            return "Sẵn sàng"
        case .failed:
            return "Lỗi"
        }
    }
}

struct CoreReadinessBridgeClient: Sendable {
    static let helperExecutableName = "scanfb-bridge-helper"
    static let helperBundleRelativePath = "Contents/Helpers/scanfb-bridge-helper"
    static let schemaVersion = 1
    static let coreIdentity = "scanfb-core"
    static let timeoutSeconds: TimeInterval = 2.0
    static let maxRequestBytes = 1024
    static let maxResponseBytes = 1024
    static let terminationGraceSeconds: TimeInterval = 0.5

    struct HelperExecution: Equatable, Sendable {
        let exitCode: Int32
        let stdout: Data
        let stderr: Data
    }

    typealias HelperRunner = @Sendable (URL, Data, TimeInterval) async -> Result<HelperExecution, CoreReadinessBridgeError>

    private let helperURLProvider: @Sendable () -> URL?
    private let helperRunner: HelperRunner

    init(
        helperURLProvider: @escaping @Sendable () -> URL? = {
            CoreReadinessBridgeClient.helperURL(inAppBundleURL: Bundle.main.bundleURL)
        },
        helperRunner: @escaping HelperRunner = CoreReadinessBridgeClient.runHelperProcess
    ) {
        self.helperURLProvider = helperURLProvider
        self.helperRunner = helperRunner
    }

    static func helperURL(
        inAppBundleURL appBundleURL: URL,
        fileManager: FileManager = .default
    ) -> URL? {
        let helperURL = appBundleURL.appendingPathComponent(helperBundleRelativePath)
        guard fileManager.isExecutableFile(atPath: helperURL.path) else {
            return nil
        }
        return helperURL
    }

    func checkReadiness() async -> Result<CoreReadinessBridgeResponse, CoreReadinessBridgeError> {
        guard !Task.isCancelled else {
            return .failure(.cancelled)
        }
        guard let helperURL = helperURLProvider() else {
            return .failure(.helperExecutableMissing)
        }

        let requestData: Data
        do {
            requestData = try Self.encodeRequest(.current)
        } catch {
            return .failure(.requestEncodingFailed)
        }

        guard requestData.count <= Self.maxRequestBytes else {
            return .failure(.requestEncodingFailed)
        }

        let executionResult = await helperRunner(helperURL, requestData, Self.timeoutSeconds)
        switch executionResult {
        case let .failure(error):
            return .failure(error)
        case let .success(execution):
            guard execution.stdout.count <= Self.maxResponseBytes else {
                return .failure(.oversizedResponse)
            }
            guard execution.exitCode == 0 else {
                return .failure(.nonzeroExit(execution.exitCode))
            }
            return Self.decodeResponse(execution.stdout)
        }
    }

    static func encodeRequest(_ request: CoreReadinessBridgeRequest) throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        return try encoder.encode(request)
    }

    static func decodeResponse(_ data: Data) -> Result<CoreReadinessBridgeResponse, CoreReadinessBridgeError> {
        do {
            let response = try JSONDecoder().decode(CoreReadinessBridgeResponse.self, from: data)
            guard response.schemaVersion == schemaVersion else {
                return .failure(.unsupportedResponseSchema)
            }
            return .success(response)
        } catch {
            return .failure(.malformedResponse)
        }
    }

    private static func runHelperProcess(
        helperURL: URL,
        requestData: Data,
        timeout: TimeInterval
    ) async -> Result<HelperExecution, CoreReadinessBridgeError> {
        let processState = BridgeProcessState()

        return await withTaskCancellationHandler {
            await withCheckedContinuation { continuation in
                processState.setContinuation(continuation)

                let process = Process()
                let stdinPipe = Pipe()
                let stdoutPipe = Pipe()
                let stderrPipe = Pipe()

                process.executableURL = helperURL
                process.standardInput = stdinPipe
                process.standardOutput = stdoutPipe
                process.standardError = stderrPipe
                processState.setProcess(process)

                process.terminationHandler = { terminatedProcess in
                    let stdout = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
                    let stderr = stderrPipe.fileHandleForReading.readDataToEndOfFile()
                    processState.complete(
                        HelperExecution(
                            exitCode: terminatedProcess.terminationStatus,
                            stdout: stdout,
                            stderr: stderr
                        )
                    )
                }

                do {
                    try process.run()
                    processState.stopIfNeeded()
                } catch {
                    processState.fail(.launchFailed)
                    return
                }

                DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + timeout) {
                    processState.fail(.timeout)
                }

                do {
                    try stdinPipe.fileHandleForWriting.write(contentsOf: requestData)
                    try stdinPipe.fileHandleForWriting.close()
                } catch {
                    processState.fail(.launchFailed)
                }
            }
        } onCancel: {
            processState.fail(.cancelled)
        }
    }
}

private final class BridgeProcessState: @unchecked Sendable {
    private let lock = NSLock()
    private var process: Process?
    private var continuation: CheckedContinuation<Result<CoreReadinessBridgeClient.HelperExecution, CoreReadinessBridgeError>, Never>?
    private var terminalError: CoreReadinessBridgeError?
    private var completed = false

    func setContinuation(
        _ continuation: CheckedContinuation<Result<CoreReadinessBridgeClient.HelperExecution, CoreReadinessBridgeError>, Never>
    ) {
        lock.lock()
        if completed {
            let error = terminalError ?? .cancelled
            lock.unlock()
            continuation.resume(returning: .failure(error))
            return
        }
        self.continuation = continuation
        lock.unlock()
    }

    func setProcess(_ process: Process) {
        lock.lock()
        self.process = process
        let shouldStop = terminalError != nil
        lock.unlock()

        if shouldStop {
            stop(process)
        }
    }

    func stopIfNeeded() {
        lock.lock()
        let processToStop = terminalError == nil ? nil : process
        lock.unlock()

        if let processToStop {
            stop(processToStop)
        }
    }

    func fail(_ error: CoreReadinessBridgeError) {
        lock.lock()
        guard !completed else {
            lock.unlock()
            return
        }

        terminalError = error
        let processToStop = process
        if processToStop == nil {
            completed = true
            let continuationToResume = continuation
            continuation = nil
            lock.unlock()
            continuationToResume?.resume(returning: .failure(error))
            return
        }
        lock.unlock()

        stop(processToStop)
    }

    func complete(_ execution: CoreReadinessBridgeClient.HelperExecution) {
        lock.lock()
        guard !completed else {
            lock.unlock()
            return
        }

        completed = true
        let error = terminalError
        let continuationToResume = continuation
        continuation = nil
        lock.unlock()

        if let error {
            continuationToResume?.resume(returning: .failure(error))
        } else {
            continuationToResume?.resume(returning: .success(execution))
        }
    }

    private func stop(_ process: Process?) {
        guard let process, process.isRunning else {
            return
        }

        process.terminate()
        DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + CoreReadinessBridgeClient.terminationGraceSeconds) {
            if process.isRunning {
                kill(process.processIdentifier, SIGKILL)
            }
        }
    }
}
