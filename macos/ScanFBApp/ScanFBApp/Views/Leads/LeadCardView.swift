import SwiftUI

struct LeadCardView: View {
    let lead: LeadFixture
    let interactionState: LeadInteractionState
    let onMarkViewed: () -> Void
    let onMarkContacted: () -> Void
    let onMarkIgnored: () -> Void

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

            statusLabel

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
        .accessibilityLabel(
            "\(lead.displayIdentity), \(lead.category.title), trạng thái \(interactionState.title), \(lead.sourcePostCount) source post mẫu"
        )
    }

    private var statusLabel: some View {
        Label("Trạng thái: \(interactionState.title)", systemImage: interactionState.symbolName)
            .font(.callout)
            .fontWeight(.medium)
            .foregroundStyle(.secondary)
            .accessibilityElement(children: .ignore)
            .accessibilityLabel("Trạng thái tương tác: \(interactionState.title)")
    }

    private var actions: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                actionButtons
            }

            VStack(alignment: .leading, spacing: 5) {
                actionButtons
            }
        }
    }

    private var actionButtons: some View {
        HStack(spacing: 8) {
            Button("Đánh dấu đã xem", action: onMarkViewed)
                .disabled(interactionState != .new)
                .accessibilityHint(viewedActionHint)
            Button("Đã liên hệ", action: onMarkContacted)
                .disabled(interactionState == .contacted)
                .accessibilityHint("Đặt trạng thái tương tác của lead mẫu thành Đã liên hệ")
            Button("Bỏ qua", action: onMarkIgnored)
                .disabled(interactionState == .ignored)
                .accessibilityHint("Đặt trạng thái tương tác của lead mẫu thành Bỏ qua")
        }
    }

    private var viewedActionHint: String {
        if interactionState == .new {
            return "Đặt trạng thái tương tác của lead mẫu thành Đã xem"
        }
        return "Chỉ khả dụng khi lead mẫu còn ở trạng thái Mới"
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
