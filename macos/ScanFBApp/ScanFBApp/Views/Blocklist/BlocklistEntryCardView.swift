import SwiftUI

struct BlocklistEntryCardView: View {
    let entry: BlocklistEntryFixture
    let identityNotice: String
    let unavailableActionMessage: String

    private let metadataColumns = [
        GridItem(.adaptive(minimum: 250), spacing: 12, alignment: .leading)
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(entry.displayLabel)
                    .font(.headline)
                    .frame(maxWidth: .infinity, alignment: .leading)

                Text(entry.identityKind.title)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(.quaternary, in: Capsule())
            }

            LazyVGrid(columns: metadataColumns, alignment: .leading, spacing: 4) {
                metadataLabel("Định danh", entry.identityValue, systemImage: "person.text.rectangle")
                metadataLabel("Ngày thêm", entry.addedDate, systemImage: "calendar")
            }
            .font(.callout)
            .foregroundStyle(.secondary)

            Text(entry.reason.description)
                .font(.body)
                .fixedSize(horizontal: false, vertical: true)

            Label(identityNotice, systemImage: "info.circle")
                .font(.callout)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            actions
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(
            "\(entry.displayLabel), \(entry.identityKind.title), \(entry.identityValue), \(entry.reason.description), thêm ngày \(entry.addedDate). \(identityNotice)"
        )
    }

    private var actions: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                actionButtons
                unavailableText
            }

            VStack(alignment: .leading, spacing: 5) {
                actionButtons
                unavailableText
            }
        }
    }

    private var actionButtons: some View {
        HStack(spacing: 8) {
            Button("Xóa khỏi blocklist") {}
                .disabled(true)
                .accessibilityHint(unavailableActionMessage)
            Button("Xem chi tiết") {}
                .disabled(true)
                .accessibilityHint(unavailableActionMessage)
        }
    }

    private var unavailableText: some View {
        Text(unavailableActionMessage)
            .font(.callout)
            .foregroundStyle(.secondary)
            .fixedSize(horizontal: false, vertical: true)
    }

    private func metadataLabel(_ title: String, _ value: String, systemImage: String) -> some View {
        Label {
            Text("\(title): \(value)")
                .fixedSize(horizontal: false, vertical: true)
                .textSelection(.enabled)
        } icon: {
            Image(systemName: systemImage)
                .accessibilityHidden(true)
        }
    }
}
