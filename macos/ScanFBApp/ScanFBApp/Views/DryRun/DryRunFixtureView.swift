import SwiftUI

struct DryRunFixtureView: View {
    let fixture: DryRunScreenFixture
    @State private var selectedTab: DryRunPresentationTab

    init(fixture: DryRunScreenFixture = .sample) {
        self.fixture = fixture
        _selectedTab = State(initialValue: DryRunScreenFixture.initialTab)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                header

                DryRunTabsView(
                    tabs: fixture.tabs,
                    selectedTab: $selectedTab,
                    countProvider: fixture.count(for:)
                )

                VStack(alignment: .leading, spacing: 6) {
                    ForEach(fixture.posts(for: selectedTab)) { post in
                        DryRunPostCardView(post: post)
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

            Label("Tổng: \(fixture.totalPostCount) bài mẫu", systemImage: "doc.text.magnifyingglass")
                .font(.callout)
                .foregroundStyle(.secondary)
                .accessibilityElement(children: .ignore)
                .accessibilityLabel("Tổng: \(fixture.totalPostCount) bài mẫu")
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }
}

#Preview {
    DryRunFixtureView()
}
