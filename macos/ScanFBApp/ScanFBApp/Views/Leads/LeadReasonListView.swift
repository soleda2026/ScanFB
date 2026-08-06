import SwiftUI

struct LeadReasonListView: View {
    let reasons: [LeadReasonFixture]

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("Reason codes")
                .font(.callout)
                .fontWeight(.medium)

            ForEach(reasons) { reason in
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text(reason.code)
                        .font(.callout)
                        .monospaced()
                    Text(reason.description)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
                .accessibilityElement(children: .ignore)
                .accessibilityLabel("\(reason.code): \(reason.description)")
            }
        }
    }
}
