import SwiftUI

struct ReasonBreakdownView: View {
    let items: [ReasonBreakdownItem]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Lý do loại")
                .font(.title3)
                .fontWeight(.semibold)

            VStack(spacing: 0) {
                ForEach(items) { item in
                    HStack(spacing: 12) {
                        Text(item.label)
                            .font(.body)
                            .frame(maxWidth: .infinity, alignment: .leading)

                        Text(item.count.formatted())
                            .font(.headline)
                            .monospacedDigit()
                    }
                    .padding(.vertical, 8)
                    .accessibilityElement(children: .ignore)
                    .accessibilityLabel("\(item.label): \(item.count.formatted()) bài")

                    if item.id != items.last?.id {
                        Divider()
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 4)
            .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
    }
}
