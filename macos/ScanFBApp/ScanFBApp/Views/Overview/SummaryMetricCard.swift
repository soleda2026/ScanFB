import SwiftUI

struct SummaryMetricCard: View {
    enum Tone {
        case success
        case failure
        case review
        case neutral

        var foregroundStyle: Color {
            switch self {
            case .success:
                .green
            case .failure:
                .red
            case .review:
                .orange
            case .neutral:
                .secondary
            }
        }
    }

    let title: String
    let value: Int
    let symbolName: String
    let footnote: String
    let tone: Tone

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Image(systemName: symbolName)
                    .foregroundStyle(tone.foregroundStyle)
                    .accessibilityHidden(true)

                Text(title)
                    .font(.headline)
                    .foregroundStyle(.primary)
            }

            Text(value.formatted())
                .font(.system(.title, design: .rounded, weight: .semibold))
                .foregroundStyle(.primary)

            Text(footnote)
                .font(.callout)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, minHeight: 104, alignment: .leading)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(title): \(value.formatted()) \(footnote)")
    }
}
