import SwiftUI

struct SettingsSectionView: View {
    let section: SettingsSectionFixture
    let bridgeStatus: CoreReadinessDisplayStatus
    let isBridgeChecking: Bool
    let onCheckBridge: () -> Void

    init(
        section: SettingsSectionFixture,
        bridgeStatus: CoreReadinessDisplayStatus = .notChecked,
        isBridgeChecking: Bool = false,
        onCheckBridge: @escaping () -> Void = {}
    ) {
        self.section = section
        self.bridgeStatus = bridgeStatus
        self.isBridgeChecking = isBridgeChecking
        self.onCheckBridge = onCheckBridge
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(section.title)
                .font(.title3)
                .fontWeight(.semibold)

            VStack(spacing: 0) {
                ForEach(section.rows) { row in
                    if row.id == "go-bridge" {
                        BridgeReadinessRowView(
                            row: row,
                            status: bridgeStatus,
                            isChecking: isBridgeChecking,
                            onCheck: onCheckBridge
                        )
                    } else {
                        SettingsRowView(row: row)
                    }

                    if row.id != section.rows.last?.id {
                        Divider()
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 4)
            .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
    }
}

private struct BridgeReadinessRowView: View {
    let row: SettingsRowFixture
    let status: CoreReadinessDisplayStatus
    let isChecking: Bool
    let onCheck: () -> Void

    var body: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                label
                statusBadge
                checkButton
            }

            VStack(alignment: .leading, spacing: 7) {
                label
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    statusBadge
                    checkButton
                }
            }
        }
        .padding(.vertical, 7)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(row.label): \(status.label)")
    }

    private var label: some View {
        Text(row.label)
            .font(.body)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var statusBadge: some View {
        Text(status.label)
            .font(.callout)
            .fontWeight(.medium)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(.quaternary, in: Capsule())
            .fixedSize(horizontal: false, vertical: true)
    }

    private var checkButton: some View {
        Button("Kiểm tra kết nối", action: onCheck)
            .disabled(isChecking)
    }
}

private struct SettingsRowView: View {
    let row: SettingsRowFixture

    var body: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                label
                value
            }

            VStack(alignment: .leading, spacing: 4) {
                label
                value
            }
        }
        .padding(.vertical, 7)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(row.label): \(row.value)")
    }

    private var label: some View {
        Text(row.label)
            .font(.body)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private var value: some View {
        switch row.style {
        case .standard:
            Text(row.value)
                .font(.body)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        case .badge:
            Text(row.value)
                .font(.callout)
                .fontWeight(.medium)
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(.quaternary, in: Capsule())
                .fixedSize(horizontal: false, vertical: true)
        }
    }
}
