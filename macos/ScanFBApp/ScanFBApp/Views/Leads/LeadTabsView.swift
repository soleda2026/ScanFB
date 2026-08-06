import SwiftUI

struct LeadTabsView: View {
    let tabs: [LeadPresentationTab]
    @Binding var selectedTab: LeadPresentationTab
    let countProvider: (LeadPresentationTab) -> Int

    var body: some View {
        Picker("Bộ lọc lead", selection: $selectedTab) {
            ForEach(tabs) { tab in
                Text("\(tab.title) (\(countProvider(tab)))")
                    .tag(tab)
                    .accessibilityLabel("\(tab.title): \(countProvider(tab)) lead")
            }
        }
        .pickerStyle(.segmented)
    }
}
