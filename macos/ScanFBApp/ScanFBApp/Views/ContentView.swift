import SwiftUI

struct ContentView: View {
    @State private var selection: AppSection? = AppSection.defaultSection

    var body: some View {
        NavigationSplitView {
            SidebarView(selection: $selection)
        } detail: {
            switch selection ?? AppSection.defaultSection {
            case .overview:
                OverviewDashboardView()
            case .leads:
                LeadsFixtureView()
            case .dryRun, .groups, .blocklist, .settings:
                PlaceholderDetailView(section: selection ?? AppSection.defaultSection)
            }
        }
        .navigationTitle("ScanFB")
    }
}

#Preview {
    ContentView()
}
