import SwiftUI

struct OverviewDashboardView: View {
    let fixture: DashboardFixture

    init(fixture: DashboardFixture = .sample) {
        self.fixture = fixture
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                header

                LazyVGrid(columns: adaptiveColumns, spacing: 8) {
                    SummaryMetricCard(
                        title: "Thành công",
                        value: fixture.groupStatus.successfulGroups,
                        symbolName: "checkmark.circle",
                        footnote: "group",
                        tone: .success
                    )
                    SummaryMetricCard(
                        title: "Lỗi",
                        value: fixture.groupStatus.failedGroups,
                        symbolName: "exclamationmark.triangle",
                        footnote: "group",
                        tone: .failure
                    )
                    SummaryMetricCard(
                        title: "Chờ xử lý",
                        value: fixture.groupStatus.pendingGroups,
                        symbolName: "clock",
                        footnote: "group",
                        tone: .neutral
                    )
                    SummaryMetricCard(
                        title: "Tổng nhóm",
                        value: fixture.groupStatus.totalGroups,
                        symbolName: "rectangle.stack",
                        footnote: "group",
                        tone: .neutral
                    )
                }

                section("Chỉ số chính") {
                    LazyVGrid(columns: adaptiveColumns, spacing: 8) {
                        ForEach(fixture.primaryMetrics) { metric in
                            SummaryMetricCard(
                                title: metric.title,
                                value: metric.value,
                                symbolName: metric.symbolName,
                                footnote: "mẫu",
                                tone: .neutral
                            )
                        }
                    }
                }

                DecisionSummaryView(summary: fixture.decisionSummary)
                ReasonBreakdownView(items: fixture.exclusionReasons)
            }
            .padding(22)
            .frame(maxWidth: 1120, alignment: .leading)
        }
        .accessibilityElement(children: .contain)
    }

    private var adaptiveColumns: [GridItem] {
        [GridItem(.adaptive(minimum: 170), spacing: 8, alignment: .top)]
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                Text("Tổng quan")
                    .font(.largeTitle)
                    .fontWeight(.semibold)

                Text(fixture.stateLabel)
                    .font(.callout)
                    .fontWeight(.medium)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(.quaternary, in: Capsule())
                    .accessibilityLabel("Trạng thái dữ liệu: \(fixture.stateLabel)")
            }

            Text(fixture.title)
                .font(.title2)
                .fontWeight(.medium)

            Text(fixture.sampleDisclaimer)
                .font(.body)
                .foregroundStyle(.secondary)

            HStack(spacing: 10) {
                metadataLabel("Ngày", fixture.dateLabel, systemImage: "calendar")
                metadataLabel("Khu vực", fixture.geographicMode, systemImage: "mappin.and.ellipse")
                metadataLabel("Profile", fixture.searchProfile, systemImage: "laptopcomputer")
                metadataLabel("Dry Run", fixture.dryRunLabel, systemImage: "checkmark.shield")
            }
            .font(.callout)
            .foregroundStyle(.secondary)
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
    }

    private func metadataLabel(_ title: String, _ value: String, systemImage: String) -> some View {
        Label {
            Text("\(title): \(value)")
        } icon: {
            Image(systemName: systemImage)
                .accessibilityHidden(true)
        }
    }

    private func section<Content: View>(_ title: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.title3)
                .fontWeight(.semibold)
            content()
        }
    }
}

#Preview {
    OverviewDashboardView()
}
