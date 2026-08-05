import SwiftUI

struct PlaceholderDetailView: View {
    let section: AppSection

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: section.symbolName)
                .font(.system(size: 44, weight: .regular))
                .foregroundStyle(.secondary)
                .accessibilityHidden(true)

            Text(section.title)
                .font(.largeTitle)
                .fontWeight(.semibold)

            Text(section.placeholderSentence)
                .font(.body)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 420)

            Text("Giao diện nền tảng — chưa kết nối Go core")
                .font(.callout)
                .foregroundStyle(.tertiary)
        }
        .padding(32)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#Preview {
    PlaceholderDetailView(section: .overview)
}
