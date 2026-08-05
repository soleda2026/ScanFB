import SwiftUI

struct ContentView: View {
    @State private var selection: AppSection? = AppSection.defaultSection

    var body: some View {
        NavigationSplitView {
            SidebarView(selection: $selection)
        } detail: {
            PlaceholderDetailView(section: selection ?? AppSection.defaultSection)
        }
        .navigationTitle("ScanFB")
    }
}

#Preview {
    ContentView()
}
