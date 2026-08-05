import SwiftUI

enum AppSection: String, CaseIterable, Identifiable {
    case overview
    case leads
    case dryRun
    case groups
    case blocklist
    case settings

    static let defaultSection: AppSection = .overview

    var id: String {
        rawValue
    }

    var title: String {
        switch self {
        case .overview:
            "Tổng quan"
        case .leads:
            "Leads"
        case .dryRun:
            "Dry Run"
        case .groups:
            "Nhóm"
        case .blocklist:
            "Blocklist"
        case .settings:
            "Cài đặt"
        }
    }

    var symbolName: String {
        switch self {
        case .overview:
            "rectangle.grid.2x2"
        case .leads:
            "person.2"
        case .dryRun:
            "doc.text.magnifyingglass"
        case .groups:
            "rectangle.stack"
        case .blocklist:
            "hand.raised"
        case .settings:
            "gearshape"
        }
    }

    var placeholderSentence: String {
        switch self {
        case .overview:
            "Tổng quan batch sẽ được triển khai trong một milestone sau."
        case .leads:
            "Danh sách lead sẽ được triển khai trong một milestone sau."
        case .dryRun:
            "Màn hình Dry Run sẽ được triển khai trong một milestone sau."
        case .groups:
            "Quản lý nhóm sẽ được triển khai trong một milestone sau."
        case .blocklist:
            "Blocklist sẽ được triển khai trong một milestone sau."
        case .settings:
            "Cài đặt sẽ được triển khai trong một milestone sau."
        }
    }
}
