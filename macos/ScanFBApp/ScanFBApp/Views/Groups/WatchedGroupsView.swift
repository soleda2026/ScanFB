import SwiftUI

struct WatchedGroupsView: View {
    @ObservedObject var store: WatchedGroupsStore
    @State private var isShowingEnrollment = false
    @State private var selectedScanGroup: WatchedGroupBridgeValue?
    @StateObject private var preparedScanStore = PreparedGroupScanStore()

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
        .sheet(item: $selectedScanGroup) { group in
            PreparedGroupScanSheet(group: group, store: preparedScanStore)
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
                            },
                            onPrepareScan: {
                                preparedScanStore.beginSession()
                                selectedScanGroup = group
                            }
                        )

                        if index < store.groups.count - 1 {
                            Divider()
                        }
                    }
                }
                .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))

                if !PreparedGroupScanStore.hasActiveGroup(store.groups) {
                    Text("Bật ít nhất một nhóm để nhập dữ liệu quét.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
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

private struct PreparedGroupScanSheet: View {
    let group: WatchedGroupBridgeValue
    @ObservedObject var store: PreparedGroupScanStore
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Nhập dữ liệu quét")
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text(group.name)
                        .font(.body)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Đóng", systemImage: "xmark") {
                    dismiss()
                }
                .labelStyle(.iconOnly)
                .help("Đóng")
            }
            .padding(20)
            .onChange(of: store.posts) {
                store.formDidChange(group: group)
            }

            Divider()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    ForEach($store.posts) { $post in
                        PreparedPostEditor(
                            post: $post,
                            number: store.posts.firstIndex(where: { $0.id == post.id }).map { $0 + 1 } ?? 1,
                            canRemove: store.posts.count > PreparedGroupScanStore.minimumPostCount,
                            onRemove: { store.removePost(id: post.id) }
                        )
                    }

                    HStack {
                        Button("Thêm bài", systemImage: "plus") {
                            store.addPost()
                        }
                        .disabled(store.posts.count >= PreparedGroupScanStore.maximumPostCount || store.isSubmitting)

                        Spacer()

                        Text("\(store.posts.count)/\(PreparedGroupScanStore.maximumPostCount)")
                            .font(.callout.monospacedDigit())
                            .foregroundStyle(.secondary)
                    }

                    if let errorMessage = store.errorMessage {
                        Label(errorMessage, systemImage: "exclamationmark.triangle")
                            .font(.callout)
                            .foregroundStyle(.red)
                            .accessibilityLabel("Lỗi: \(errorMessage)")
                    }

                    if let result = store.result {
                        PreparedGroupScanResultView(result: result)
                    }
                }
                .padding(20)
            }

            Divider()

            HStack {
                Text("Dữ liệu nhập và kết quả chỉ tồn tại trong phiên này.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
                Spacer()
                Button("Quét dữ liệu đã nhập", systemImage: "play.fill") {
                    Task { await store.submit(group: group) }
                }
                .buttonStyle(.borderedProminent)
                .disabled(store.isSubmitting)
            }
            .padding(20)
        }
        .frame(minWidth: 680, minHeight: 620)
    }
}

private struct PreparedPostEditor: View {
    @Binding var post: PreparedPostDraft
    let number: Int
    let canRemove: Bool
    let onRemove: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Bài \(number)")
                    .font(.headline)
                Spacer()
                Button("Xóa bài", systemImage: "trash", role: .destructive) {
                    onRemove()
                }
                .labelStyle(.iconOnly)
                .disabled(!canRemove)
                .help("Xóa bài \(number)")
            }

            Text("Nội dung")
                .font(.subheadline)
                .fontWeight(.medium)
            TextEditor(text: $post.body)
                .frame(minHeight: 88)
                .padding(6)
                .background(.background, in: RoundedRectangle(cornerRadius: 6, style: .continuous))

            TextField("Tên hiển thị tác giả", text: $post.authorDisplayName)

            DatePicker(
                "Thời gian tạo",
                selection: $post.createdAt,
                displayedComponents: [.date, .hourAndMinute]
            )
            .environment(\.timeZone, PreparedGroupScanStore.hoChiMinhTimeZone)

            HStack {
                TextField("Post URL (tùy chọn)", text: $post.postURL)
                TextField("Post ID (tùy chọn)", text: $post.postID)
            }

            DisclosureGroup("Định danh tác giả khác (tùy chọn)") {
                VStack(spacing: 10) {
                    TextField("Username", text: $post.authorUsername)
                    TextField("Facebook User ID", text: $post.authorFacebookUserID)
                    TextField("Canonical Profile URL", text: $post.authorCanonicalProfileURL)
                }
                .padding(.top, 8)
            }
        }
        .padding(14)
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }
}

private struct PreparedGroupScanResultView: View {
    let result: PreparedGroupScanBridgeResponse

    private let columns = Array(repeating: GridItem(.flexible(), spacing: 10), count: 3)

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Quét hoàn tất", systemImage: "checkmark.circle.fill")
                .font(.headline)
                .foregroundStyle(.green)

            Text(result.groupName ?? "")
                .font(.body)

            LazyVGrid(columns: columns, spacing: 10) {
                resultMetric("Đã thu thập", result.collectedPostCount)
                resultMetric("Đã đánh giá", result.evaluatedPostCount)
                resultMetric("Được chọn", result.includedPostCount)
                resultMetric("Cần xem xét", result.reviewPostCount)
                resultMetric("Đã loại", result.excludedPostCount)
                resultMetric("Lead phù hợp", result.allowedLeadCount)
            }
        }
        .padding(14)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }

    private func resultMetric(_ label: String, _ value: Int) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text("\(value)")
                .font(.title3.monospacedDigit())
                .fontWeight(.semibold)
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(.background, in: RoundedRectangle(cornerRadius: 6, style: .continuous))
    }
}
