import Foundation

struct LeadSourceURLHandoff: Equatable {
    let sourceURLString: String

    var validatedURL: URL? {
        Self.validatedHTTPSURL(from: sourceURLString)
    }

    var canOpen: Bool {
        validatedURL != nil
    }

    @discardableResult
    func handoff(open: (URL) -> Void) -> Bool {
        guard let url = validatedURL else {
            return false
        }

        open(url)
        return true
    }

    static func validatedHTTPSURL(from rawValue: String) -> URL? {
        let trimmedValue = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard
            let components = URLComponents(string: trimmedValue),
            components.scheme?.lowercased() == "https",
            components.host?.isEmpty == false,
            components.user == nil,
            components.password == nil,
            let url = components.url
        else {
            return nil
        }

        return url
    }
}
