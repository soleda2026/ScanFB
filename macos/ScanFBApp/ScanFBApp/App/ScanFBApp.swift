import SwiftUI

@main
struct ScanFBApp: App {
    var body: some Scene {
        WindowGroup("ScanFB") {
            ContentView()
                .frame(minWidth: 720, minHeight: 480)
        }
        .defaultSize(width: 1000, height: 680)
    }
}
