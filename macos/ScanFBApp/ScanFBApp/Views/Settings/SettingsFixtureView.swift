import SwiftUI

struct SettingsFixtureView: View {
    let fixture: SettingsScreenFixture
    let bridgeClient: CoreReadinessBridgeClient
    @State private var bridgeStatus: CoreReadinessDisplayStatus = .notChecked
    @State private var isCheckingBridge = false

    init(
        fixture: SettingsScreenFixture = .sample,
        bridgeClient: CoreReadinessBridgeClient = CoreReadinessBridgeClient()
    ) {
        self.fixture = fixture
        self.bridgeClient = bridgeClient
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                header

                LazyVGrid(columns: columns, alignment: .leading, spacing: 8) {
                    ForEach(fixture.sections) { section in
                        SettingsSectionView(
                            section: section,
                            bridgeStatus: bridgeStatus,
                            isBridgeChecking: isCheckingBridge,
                            onCheckBridge: checkBridgeReadiness
                        )
                    }
                }
            }
            .padding(18)
            .frame(maxWidth: 1120, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
    }

    private var columns: [GridItem] {
        [GridItem(.adaptive(minimum: 360), spacing: 8, alignment: .top)]
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Text(fixture.title)
                    .font(.largeTitle)
                    .fontWeight(.semibold)

                Text(fixture.stateLabel)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(.quaternary, in: Capsule())
                    .accessibilityLabel("Trạng thái dữ liệu: \(fixture.stateLabel)")
            }

            Text(fixture.disclaimer)
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }

    private func checkBridgeReadiness() {
        guard !isCheckingBridge else {
            return
        }

        bridgeStatus = .checking
        isCheckingBridge = true

        Task {
            let result = await bridgeClient.checkReadiness()

            await MainActor.run {
                isCheckingBridge = false
                switch result {
                case let .success(response) where response.readinessStatus == .ready:
                    bridgeStatus = .ready
                default:
                    bridgeStatus = .failed
                }
            }
        }
    }
}

#Preview {
    SettingsFixtureView()
}
