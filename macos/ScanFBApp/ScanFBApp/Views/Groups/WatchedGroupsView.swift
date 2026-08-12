import SwiftUI

struct WatchedGroupsView: View {
    @ObservedObject var store: WatchedGroupsStore

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
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text("Nhóm theo dõi")
                .font(.largeTitle)
                .fontWeight(.semibold)

            Text("Các nhóm đã tham gia sẽ được đồng bộ từ Facebook trong Safari và lưu cục bộ trên máy này.")
                .font(.body)
                .foregroundStyle(.secondary)

            Text("Tính năng đồng bộ chưa khả dụng.")
                .font(.callout)
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

                Button("Đồng bộ nhóm đã tham gia", systemImage: "arrow.triangle.2.circlepath") {}
                    .disabled(true)
                    .help("Chưa khả dụng")
                    .accessibilityLabel("Đồng bộ nhóm đã tham gia, chưa khả dụng")
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
                    "Chưa có nhóm theo dõi",
                    systemImage: "rectangle.stack",
                    description: Text("Danh sách sẽ xuất hiện sau khi tính năng đồng bộ nhóm đã tham gia được triển khai.")
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

            Text("Bản xem trước chỉ đọc cho batch Scan tiếp theo.")
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
