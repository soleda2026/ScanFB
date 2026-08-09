enum LeadInteractionState: String, CaseIterable, Equatable {
    case new
    case viewed
    case ignored

    var title: String {
        switch self {
        case .new:
            "Mới"
        case .viewed:
            "Đã xem"
        case .ignored:
            "Bỏ qua"
        }
    }

    var symbolName: String {
        switch self {
        case .new:
            "sparkle"
        case .viewed:
            "eye"
        case .ignored:
            "hand.raised"
        }
    }
}

struct LeadInteractionStateModel: Equatable {
    private(set) var statesByLeadID: [String: LeadInteractionState]

    init(leadIDs: [String]) {
        statesByLeadID = Dictionary(uniqueKeysWithValues: leadIDs.map { ($0, LeadInteractionState.new) })
    }

    func state(for leadID: String) -> LeadInteractionState {
        statesByLeadID[leadID] ?? .new
    }

    mutating func markViewed(_ leadID: String) {
        guard state(for: leadID) == .new else {
            return
        }
        statesByLeadID[leadID] = .viewed
    }

    mutating func markIgnored(_ leadID: String) {
        statesByLeadID[leadID] = .ignored
    }
}
