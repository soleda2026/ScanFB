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
            case .dryRun:
                DryRunFixtureView()
            case .groups:
                PlaceholderDetailView(section: selection ?? AppSection.defaultSection)
            case .blocklist:
                BlocklistFixtureView()
            case .settings:
                SettingsFixtureView()
            }
        }
        .navigationTitle("ScanFB")
    }
}

#Preview {
    ContentView()
}
