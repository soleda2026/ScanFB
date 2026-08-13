import SwiftUI

struct WatchedGroupRowView: View {
    let group: WatchedGroupBridgeValue
    let isBusy: Bool
    let onActiveChange: (Bool) -> Void
    let onPrepareScan: () -> Void

    var body: some View {
        HStack(spacing: 14) {
            Text(group.name)
                .font(.body)
                .frame(maxWidth: .infinity, alignment: .leading)

            Button("Nhập dữ liệu quét", systemImage: "square.and.pencil") {
                onPrepareScan()
            }
            .disabled(isBusy || !group.active)
            .help(group.active ? "Nhập bài viết cho nhóm này" : "Bật nhóm trước khi nhập dữ liệu quét")

            Toggle(
                "Hoạt động",
                isOn: Binding(
                    get: { group.active },
                    set: onActiveChange
                )
            )
            .toggleStyle(.switch)
            .fixedSize()
            .disabled(isBusy)
            .accessibilityLabel("\(group.name), hoạt động")
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
    }
}
