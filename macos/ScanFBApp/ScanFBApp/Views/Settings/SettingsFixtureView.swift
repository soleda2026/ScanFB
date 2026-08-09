import SwiftUI

struct SettingsFixtureView: View {
    let fixture: SettingsScreenFixture

    init(fixture: SettingsScreenFixture = .sample) {
        self.fixture = fixture
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                header

                LazyVGrid(columns: columns, alignment: .leading, spacing: 8) {
                    ForEach(fixture.sections) { section in
                        SettingsSectionView(section: section)
                    }
                }
            }
            .padding(18)
            .frame(maxWidth: 1120, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
    }

    private var columns: [GridItem] {
        [GridItem(.adaptive(minimum: 360), spacing: 8, alignment: .top)]
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
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }
}

#Preview {
    SettingsFixtureView()
}
