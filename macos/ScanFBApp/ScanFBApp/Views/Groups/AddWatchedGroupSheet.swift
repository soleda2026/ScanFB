import SwiftUI

struct AddWatchedGroupSheet: View {
    @ObservedObject var store: WatchedGroupsStore
    @Binding var isPresented: Bool
    @State private var name = ""
    @State private var canonicalURL = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Thêm nhóm theo dõi")
                .font(.title2)
                .fontWeight(.semibold)

            Text("Thêm nhóm một lần bằng URL Facebook. ScanFB lưu nhóm cục bộ cho các lần Scan sau.")
                .font(.callout)
                .foregroundStyle(.secondary)

            Form {
                TextField("Tên nhóm", text: $name)
                TextField("URL nhóm", text: $canonicalURL)
            }
            .formStyle(.grouped)

            if let errorMessage = store.errorMessage {
                Text(errorMessage)
                    .font(.callout)
                    .foregroundStyle(.red)
            }

            HStack {
                Spacer()

                Button("Hủy", role: .cancel) {
                    isPresented = false
                }

                Button("Thêm") {
                    Task {
                        if await store.addGroup(name: name, canonicalURL: canonicalURL) {
                            isPresented = false
                        }
                    }
                }
                .keyboardShortcut(.defaultAction)
                .disabled(store.isBusy)
            }
        }
        .padding(22)
        .frame(width: 460)
    }
}
