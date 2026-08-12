import SwiftUI

struct WatchedGroupsView: View {
    @ObservedObject var store: WatchedGroupsStore
    @State private var isShowingEnrollment = false

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
        .sheet(isPresented: $isShowingEnrollment) {
            AddWatchedGroupSheet(store: store, isPresented: $isShowingEnrollment)
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text("Nhóm theo dõi")
                .font(.largeTitle)
                .fontWeight(.semibold)

            Text("Thêm một lần các nhóm Facebook bạn muốn theo dõi. ScanFB lưu cục bộ để dùng cho những lần Scan sau.")
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

                Button("Thêm nhóm theo dõi", systemImage: "plus") {
                    isShowingEnrollment = true
                }
                .disabled(store.loadState != .loaded || store.isBusy)
                .help("Thêm một nhóm và lưu cho các lần Scan sau")
            }

            if store.loadState == .loading || store.loadState == .idle {
                ProgressView("Đang tải nhóm đã lưu…")
                    .frame(maxWidth: .infinity, minHeight: 150)
            } else if store.loadState == .failed {
                ContentUnavailableView(
                    "Không thể tải nhóm đã lưu",
                    systemImage: "externaldrive.badge.xmark",
                    description: Text(store.errorMessage ?? "Không thể mở dữ liệu nhóm đã lưu.")
                )
                .frame(maxWidth: .infinity, minHeight: 150)
            } else if store.groups.isEmpty {
                ContentUnavailableView(
                    "Chưa thêm nhóm theo dõi",
                    systemImage: "rectangle.stack",
                    description: Text("Thêm các nhóm Facebook bạn muốn ScanFB theo dõi. Mỗi nhóm được lưu cục bộ và chỉ cần thêm một lần.")
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

            if store.loadState != .failed, let errorMessage = store.errorMessage {
                Text(errorMessage)
                    .font(.callout)
                    .foregroundStyle(.red)
                    .accessibilityLabel("Lỗi: \(errorMessage)")
            }
        }
    }

    private var nextFiveSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Next 5 Groups")
                .font(.title2)
                .fontWeight(.semibold)

            Text("Bản xem trước chỉ đọc, tự động lấy 5 nhóm đang hoạt động cho batch Scan tiếp theo. Xem trước không đổi lượt.")
                .font(.callout)
                .foregroundStyle(.secondary)

            if store.loadState != .loaded {
                Text("Lượt chọn sẽ hiển thị sau khi tải dữ liệu nhóm.")
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, minHeight: 72, alignment: .leading)
                    .padding(14)
                    .background(.quaternary, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            } else if store.needsMoreActiveGroups {
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
