import SwiftUI

struct WatchedGroupsView: View {
    @ObservedObject var store: WatchedGroupsStore
    @State private var isPresentingAddGroup = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                header
                watchedGroupsSection
                nextFiveSection
            }
            .padding(20)
            .frame(maxWidth: 920, alignment: .leading)
        }
        .task {
            await store.loadIfNeeded()
        }
        .sheet(isPresented: $isPresentingAddGroup) {
            AddWatchedGroupSheet(store: store, isPresented: $isPresentingAddGroup)
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text("Nhóm theo dõi")
                .font(.largeTitle)
                .fontWeight(.semibold)

            Text("Danh sách chỉ tồn tại trong phiên ứng dụng hiện tại.")
                .font(.body)
                .foregroundStyle(.secondary)
        }
    }

    private var watchedGroupsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Watched Groups")
                    .font(.title2)
                    .fontWeight(.semibold)

                Spacer()

                Button {
                    isPresentingAddGroup = true
                } label: {
                    Label("Thêm nhóm", systemImage: "plus")
                }
                .disabled(store.isBusy)
                .accessibilityLabel("Thêm nhóm theo dõi")
            }

            if store.groups.isEmpty {
                ContentUnavailableView(
                    "Chưa có nhóm theo dõi",
                    systemImage: "rectangle.stack",
                    description: Text("Thêm nhóm thủ công để chuẩn bị lượt chọn tiếp theo.")
                )
                .frame(maxWidth: .infinity, minHeight: 150)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(store.groups.enumerated()), id: \.element.id) { index, group in
                        WatchedGroupRowView(
                            group: group,
                            isBusy: store.isBusy,
                            onActiveChange: { active in
                                Task {
                                    await store.setActive(active, for: group.id)
                                }
                            }
                        )

                        if index < store.groups.count - 1 {
                            Divider()
                        }
                    }
                }
                .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            }

            if let errorMessage = store.errorMessage {
                Text(errorMessage)
                    .font(.callout)
                    .foregroundStyle(.red)
                    .accessibilityLabel("Lỗi: \(errorMessage)")
            }
        }
    }

    private var nextFiveSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Next 5 Groups")
                    .font(.title2)
                    .fontWeight(.semibold)

                Spacer()

                if store.canAdvanceSelection {
                    Button {
                        Task {
                            await store.advanceSelection()
                        }
                    } label: {
                        Label("Chuyển lượt chọn", systemImage: "arrow.forward")
                    }
                    .disabled(store.isBusy)
                }
            }

            if store.needsMoreActiveGroups {
                Text("Cần ít nhất 5 nhóm đang hoạt động.")
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, minHeight: 72, alignment: .leading)
                    .padding(14)
                    .background(.quaternary, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            } else {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(Array(store.nextFive.enumerated()), id: \.element.id) { index, group in
                        HStack(alignment: .firstTextBaseline, spacing: 10) {
                            Text("\(index + 1).")
                                .font(.body.monospacedDigit())
                                .foregroundStyle(.secondary)
                                .frame(width: 24, alignment: .trailing)

                            Text(group.name)
                                .font(.body)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
                .padding(14)
                .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
        }
    }
}
