import SwiftUI

struct LeadsFixtureView: View {
    let fixture: LeadsScreenFixture
    @State private var selectedTab: LeadPresentationTab = .all
    @State private var interactionState: LeadInteractionStateModel

    init(fixture: LeadsScreenFixture = .sample) {
        self.fixture = fixture
        _interactionState = State(initialValue: LeadInteractionStateModel(leadIDs: fixture.leads.map(\.id)))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 10) {
                header

                LeadTabsView(
                    tabs: fixture.tabs,
                    selectedTab: $selectedTab,
                    countProvider: fixture.count(for:)
                )

                VStack(alignment: .leading, spacing: 6) {
                    ForEach(fixture.leads(for: selectedTab)) { lead in
                        LeadCardView(
                            lead: lead,
                            interactionState: interactionState.state(for: lead.id),
                            onMarkViewed: {
                                interactionState.markViewed(lead.id)
                            },
                            onMarkContacted: {
                                interactionState.markContacted(lead.id)
                            },
                            onMarkIgnored: {
                                interactionState.markIgnored(lead.id)
                            }
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
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }
}

#Preview {
    LeadsFixtureView()
}
