import SwiftUI

struct ContentView: View {
    @State private var selection: AppSection? = AppSection.defaultSection
    @StateObject private var watchedGroupsStore = WatchedGroupsStore()

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
                WatchedGroupsView(store: watchedGroupsStore)
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
