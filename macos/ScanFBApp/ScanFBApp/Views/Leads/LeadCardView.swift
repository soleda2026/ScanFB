import SwiftUI

struct LeadCardView: View {
    let lead: LeadFixture

    private let metadataColumns = [
        GridItem(.adaptive(minimum: 230), spacing: 12, alignment: .leading)
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(lead.displayIdentity)
                    .font(.headline)
                    .frame(maxWidth: .infinity, alignment: .leading)

                Text(lead.category.title)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(.quaternary, in: Capsule())
            }

            Text(lead.excerpt)
                .font(.body)
                .foregroundStyle(.primary)
                .fixedSize(horizontal: false, vertical: true)

            LazyVGrid(columns: metadataColumns, alignment: .leading, spacing: 4) {
                metadataLabel("Ngày", lead.dateLabel, systemImage: "calendar")
                metadataLabel("Khu vực", lead.location, systemImage: "mappin.and.ellipse")
                metadataLabel("Nhóm", lead.groupName, systemImage: "rectangle.stack")
                metadataLabel("Source", "\(lead.sourcePostCount) bài mẫu", systemImage: "doc.on.doc")
            }
            .font(.callout)
            .foregroundStyle(.secondary)

            LeadReasonListView(reasons: lead.reasons)

            actions
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(lead.displayIdentity), \(lead.category.title), \(lead.sourcePostCount) source post mẫu")
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
            Button("Xem nguồn") {}
                .disabled(true)
                .accessibilityHint("Chưa khả dụng trong dữ liệu minh họa")
            Button("Đánh dấu đã xem") {}
                .disabled(true)
                .accessibilityHint("Chưa khả dụng trong dữ liệu minh họa")
        }
    }

    private var unavailableText: some View {
        Text("Chưa khả dụng trong dữ liệu minh họa")
            .font(.callout)
            .foregroundStyle(.secondary)
            .fixedSize(horizontal: false, vertical: true)
    }

    private func metadataLabel(_ title: String, _ value: String, systemImage: String) -> some View {
        Label {
            Text("\(title): \(value)")
                .fixedSize(horizontal: false, vertical: true)
        } icon: {
            Image(systemName: systemImage)
                .accessibilityHidden(true)
        }
    }
}
