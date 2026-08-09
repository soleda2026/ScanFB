import SwiftUI

struct SettingsSectionView: View {
    let section: SettingsSectionFixture

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(section.title)
                .font(.title3)
                .fontWeight(.semibold)

            VStack(spacing: 0) {
                ForEach(section.rows) { row in
                    SettingsRowView(row: row)

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
