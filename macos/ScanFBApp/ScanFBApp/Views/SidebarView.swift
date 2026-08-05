import SwiftUI

struct SidebarView: View {
    @Binding var selection: AppSection?

    var body: some View {
        List(AppSection.allCases, selection: $selection) { section in
            Label(section.title, systemImage: section.symbolName)
                .tag(section)
        }
        .listStyle(.sidebar)
        .navigationTitle("ScanFB")
    }
}

#Preview {
    SidebarView(selection: .constant(AppSection.defaultSection))
}
