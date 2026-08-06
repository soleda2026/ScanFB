import SwiftUI

struct DryRunPostCardView: View {
    let post: DryRunPostFixture

    private let metadataColumns = [
        GridItem(.adaptive(minimum: 230), spacing: 12, alignment: .leading)
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(post.author)
                    .font(.headline)
                    .frame(maxWidth: .infinity, alignment: .leading)

                Text(post.category.title)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(.quaternary, in: Capsule())
            }

            Text(post.excerpt)
                .font(.body)
                .foregroundStyle(.primary)
                .fixedSize(horizontal: false, vertical: true)

            LazyVGrid(columns: metadataColumns, alignment: .leading, spacing: 4) {
                metadataLabel("Ngày", post.dateLabel, systemImage: "calendar")
                metadataLabel("Khu vực", post.location, systemImage: "mappin.and.ellipse")
                metadataLabel("Nhóm", post.groupName, systemImage: "rectangle.stack")
            }
            .font(.callout)
            .foregroundStyle(.secondary)

            DryRunReasonListView(reasons: post.reasons)

            action
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(post.author), \(post.category.title), \(post.dateLabel)")
    }

    private var action: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                disabledButton
                unavailableText
            }

            VStack(alignment: .leading, spacing: 5) {
                disabledButton
                unavailableText
            }
        }
    }

    private var disabledButton: some View {
        Button("Xem bài gốc") {}
            .disabled(true)
            .accessibilityHint("Chưa khả dụng trong dữ liệu minh họa")
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
