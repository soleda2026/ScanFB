import SwiftUI

struct DryRunTabsView: View {
    let tabs: [DryRunPresentationTab]
    @Binding var selectedTab: DryRunPresentationTab
    let countProvider: (DryRunPresentationTab) -> Int

    var body: some View {
        Picker("Bộ lọc Dry Run", selection: $selectedTab) {
            ForEach(tabs) { tab in
                Text("\(tab.title) (\(countProvider(tab)))")
                    .tag(tab)
                    .accessibilityLabel("\(tab.title): \(countProvider(tab)) bài")
            }
        }
        .pickerStyle(.segmented)
    }
}
