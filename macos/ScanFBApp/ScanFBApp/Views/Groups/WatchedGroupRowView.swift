import SwiftUI

struct WatchedGroupRowView: View {
    let group: WatchedGroupBridgeValue
    let isBusy: Bool
    let onActiveChange: (Bool) -> Void

    var body: some View {
        HStack(spacing: 14) {
            Text(group.name)
                .font(.body)
                .frame(maxWidth: .infinity, alignment: .leading)

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
