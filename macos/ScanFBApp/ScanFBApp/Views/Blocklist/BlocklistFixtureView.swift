import SwiftUI

struct BlocklistFixtureView: View {
    let fixture: BlocklistScreenFixture

    init(fixture: BlocklistScreenFixture = .sample) {
        self.fixture = fixture
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                header

                VStack(alignment: .leading, spacing: 6) {
                    ForEach(fixture.entries) { entry in
                        BlocklistEntryCardView(
                            entry: entry,
                            identityNotice: fixture.identityNotice,
                            unavailableActionMessage: fixture.unavailableActionMessage
                        )
                    }
                }
            }
            .padding(18)
            .frame(maxWidth: 1120, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Text(fixture.title)
                    .font(.largeTitle)
                    .fontWeight(.semibold)

                Text(fixture.stateLabel)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(.quaternary, in: Capsule())
                    .accessibilityLabel("Trạng thái dữ liệu: \(fixture.stateLabel)")
            }

            Text(fixture.disclaimer)
                .font(.body)
                .foregroundStyle(.secondary)

            Label("\(fixture.entries.count) danh tính mẫu", systemImage: "hand.raised")
                .font(.callout)
                .foregroundStyle(.secondary)
                .accessibilityElement(children: .ignore)
                .accessibilityLabel("\(fixture.entries.count) danh tính mẫu")
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }
}

#Preview {
    BlocklistFixtureView()
}
