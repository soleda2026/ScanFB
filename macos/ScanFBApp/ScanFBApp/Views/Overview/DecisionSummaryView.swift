import SwiftUI

struct DecisionSummaryView: View {
    let summary: DecisionSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Tóm tắt quyết định")
                .font(.title3)
                .fontWeight(.semibold)

            LazyVGrid(columns: columns, spacing: 8) {
                SummaryMetricCard(
                    title: "Giữ lại",
                    value: summary.included,
                    symbolName: "checkmark.seal",
                    footnote: "bài",
                    tone: .success
                )
                SummaryMetricCard(
                    title: "Cần xem lại",
                    value: summary.review,
                    symbolName: "questionmark.circle",
                    footnote: "bài",
                    tone: .review
                )
                SummaryMetricCard(
                    title: "Đã loại",
                    value: summary.excluded,
                    symbolName: "xmark.octagon",
                    footnote: "bài",
                    tone: .failure
                )
            }
        }
    }

    private var columns: [GridItem] {
        [GridItem(.adaptive(minimum: 180), spacing: 8, alignment: .top)]
    }
}
