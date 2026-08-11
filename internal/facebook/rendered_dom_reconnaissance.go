package facebook

import (
	"errors"
	"net/url"
	"sort"
	"strings"
)

const (
	RenderedDOMReconnaissanceMaxMarkerCategories = 16
	RenderedDOMReconnaissanceMaxMarkerLength     = 64
	renderedDOMReconnaissanceMaxPageURLBytes     = 4 << 10
)

var (
	ErrEmptyRenderedDOMReconnaissanceInput     = errors.New("facebook: rendered DOM reconnaissance input is empty")
	ErrOversizedRenderedDOMReconnaissanceInput = errors.New("facebook: rendered DOM reconnaissance input exceeds the limit")
)

// RenderedDOMReconnaissanceConfidence describes only the strength of
// structural evidence in one analyzed document.
type RenderedDOMReconnaissanceConfidence string

const (
	RenderedDOMReconnaissanceStrong    RenderedDOMReconnaissanceConfidence = "STRONG"
	RenderedDOMReconnaissanceTentative RenderedDOMReconnaissanceConfidence = "TENTATIVE"
	RenderedDOMReconnaissanceNotFound  RenderedDOMReconnaissanceConfidence = "NOT_FOUND"
)

// RenderedDOMReconnaissanceReport contains bounded, redacted structural
// evidence. It intentionally contains no matched values or raw page content.
type RenderedDOMReconnaissanceReport struct {
	AnalyzedDOMBytes                 int                                 `json:"analyzed_dom_bytes"`
	CandidatePostContainerCount      int                                 `json:"candidate_post_container_count"`
	CandidatePermalinkBearingCount   int                                 `json:"candidate_permalink_bearing_count"`
	CandidateBodyBearingCount        int                                 `json:"candidate_body_bearing_count"`
	CandidateAuthorBearingCount      int                                 `json:"candidate_author_bearing_count"`
	CandidateMachineTimestampCount   int                                 `json:"candidate_machine_timestamp_count"`
	CandidateRelativeTimeOnlyCount   int                                 `json:"candidate_relative_time_only_count"`
	CandidateCompleteEvidenceCount   int                                 `json:"candidate_complete_evidence_count"`
	GroupConsistentPermalinkCount    int                                 `json:"group_consistent_permalink_count"`
	GroupPageURLShapeValid           bool                                `json:"group_page_url_shape_valid"`
	DeterministicTraversalCount      int                                 `json:"deterministic_traversal_count"`
	PostContainerConfidence          RenderedDOMReconnaissanceConfidence `json:"post_container_confidence"`
	PermalinkConfidence              RenderedDOMReconnaissanceConfidence `json:"permalink_confidence"`
	BodyConfidence                   RenderedDOMReconnaissanceConfidence `json:"body_confidence"`
	AuthorConfidence                 RenderedDOMReconnaissanceConfidence `json:"author_confidence"`
	MachineTimestampConfidence       RenderedDOMReconnaissanceConfidence `json:"machine_timestamp_confidence"`
	GroupIdentityConfidence          RenderedDOMReconnaissanceConfidence `json:"group_identity_confidence"`
	TraversalConfidence              RenderedDOMReconnaissanceConfidence `json:"traversal_confidence"`
	MarkerCategories                 []string                            `json:"marker_categories"`
	RejectedUnstableMarkerCategories []string                            `json:"rejected_unstable_marker_categories"`
}

type renderedDOMReconnaissanceNode struct {
	tag    string
	attrs  map[string]string
	parent int
}

type renderedDOMReconnaissanceCandidate struct {
	permalink       bool
	body            bool
	author          bool
	machineTime     bool
	relativeTime    bool
	groupConsistent bool
}

type renderedDOMReconnaissanceHrefEvidence struct {
	marker    string
	groupKey  string
	permalink bool
	author    bool
}

var renderedDOMReconnaissanceRejectedMarkers = []string{
	"arbitrary-depth",
	"broad-text-search",
	"generated-or-obfuscated-class",
	"generic-react-comet-symbol",
	"localized-visible-text",
	"nth-child-position",
	"relative-time-text",
	"title-wording",
	"transient-internal-id",
}

// AnalyzeRenderedDOMStructure analyzes one already-acquired rendered document
// without browser, filesystem, network, clock, or product-runtime access.
func AnalyzeRenderedDOMStructure(renderedDOM, pageURL string) (RenderedDOMReconnaissanceReport, error) {
	if strings.TrimSpace(renderedDOM) == "" {
		return RenderedDOMReconnaissanceReport{}, ErrEmptyRenderedDOMReconnaissanceInput
	}
	if len([]byte(renderedDOM)) > SafariRenderedDOMMaxBytes {
		return RenderedDOMReconnaissanceReport{}, ErrOversizedRenderedDOMReconnaissanceInput
	}

	pageGroupKey, pageURLValid := renderedDOMReconnaissanceGroupKey(pageURL)
	report := RenderedDOMReconnaissanceReport{
		AnalyzedDOMBytes:                 len([]byte(renderedDOM)),
		GroupPageURLShapeValid:           pageURLValid,
		PostContainerConfidence:          RenderedDOMReconnaissanceNotFound,
		PermalinkConfidence:              RenderedDOMReconnaissanceNotFound,
		BodyConfidence:                   RenderedDOMReconnaissanceNotFound,
		AuthorConfidence:                 RenderedDOMReconnaissanceNotFound,
		MachineTimestampConfidence:       RenderedDOMReconnaissanceNotFound,
		GroupIdentityConfidence:          RenderedDOMReconnaissanceNotFound,
		TraversalConfidence:              RenderedDOMReconnaissanceNotFound,
		RejectedUnstableMarkerCategories: boundedRenderedDOMReconnaissanceMarkers(renderedDOMReconnaissanceRejectedMarkers),
	}
	if pageURLValid {
		report.GroupIdentityConfidence = RenderedDOMReconnaissanceStrong
	}

	nodes := parseRenderedDOMReconnaissanceNodes(renderedDOM)
	candidateByNode := make(map[int]int)
	candidates := make([]renderedDOMReconnaissanceCandidate, 0)
	markers := make(map[string]struct{})

	for index, node := range nodes {
		if !renderedDOMReconnaissanceCandidateNode(node) || renderedDOMReconnaissanceHasCandidateAncestor(nodes, node.parent) {
			continue
		}
		candidateByNode[index] = len(candidates)
		candidates = append(candidates, renderedDOMReconnaissanceCandidate{})
		if node.tag == "article" {
			markers["tag=article"] = struct{}{}
		}
		if strings.EqualFold(node.attrs["role"], "article") {
			markers["role=article"] = struct{}{}
		}
	}

	report.CandidatePostContainerCount = len(candidates)
	report.DeterministicTraversalCount = len(candidates)
	if len(candidates) > 0 {
		report.PostContainerConfidence = RenderedDOMReconnaissanceStrong
		markers["dom-source-order"] = struct{}{}
	}
	switch len(candidates) {
	case 0:
		report.TraversalConfidence = RenderedDOMReconnaissanceNotFound
	case 1:
		report.TraversalConfidence = RenderedDOMReconnaissanceTentative
	default:
		report.TraversalConfidence = RenderedDOMReconnaissanceStrong
	}

	for index, node := range nodes {
		candidateIndex, ok := renderedDOMReconnaissanceNearestCandidate(nodes, candidateByNode, index)
		if !ok {
			continue
		}
		candidate := &candidates[candidateIndex]

		if node.tag == "a" {
			evidence := renderedDOMReconnaissanceClassifyHref(node.attrs["href"])
			if evidence.marker != "" {
				markers[evidence.marker] = struct{}{}
			}
			if evidence.permalink {
				candidate.permalink = true
				if pageURLValid && evidence.groupKey == pageGroupKey {
					candidate.groupConsistent = true
				}
			}
			if evidence.author {
				candidate.author = true
			}
		}

		if strings.EqualFold(node.attrs["data-ad-preview"], "message") {
			candidate.body = true
			markers["data-ad-preview=message"] = struct{}{}
		}
		if strings.EqualFold(node.attrs["data-ad-comet-preview"], "message") {
			candidate.body = true
			markers["data-ad-comet-preview=message"] = struct{}{}
		}
		if (node.tag == "time" && node.attrs["datetime"] != "") || node.attrs["data-utime"] != "" {
			candidate.machineTime = true
			if node.tag == "time" && node.attrs["datetime"] != "" {
				markers["time[datetime]"] = struct{}{}
			}
			if node.attrs["data-utime"] != "" {
				markers["data-utime"] = struct{}{}
			}
		} else if node.tag == "time" {
			candidate.relativeTime = true
		}
	}

	for _, candidate := range candidates {
		if candidate.permalink {
			report.CandidatePermalinkBearingCount++
		}
		if candidate.body {
			report.CandidateBodyBearingCount++
		}
		if candidate.author {
			report.CandidateAuthorBearingCount++
		}
		if candidate.machineTime {
			report.CandidateMachineTimestampCount++
		} else if candidate.relativeTime {
			report.CandidateRelativeTimeOnlyCount++
		}
		if candidate.groupConsistent {
			report.GroupConsistentPermalinkCount++
		}
		if candidate.permalink && candidate.body && candidate.author && candidate.machineTime {
			report.CandidateCompleteEvidenceCount++
		}
	}

	report.PermalinkConfidence = renderedDOMReconnaissanceCoverageConfidence(len(candidates), report.CandidatePermalinkBearingCount)
	report.BodyConfidence = renderedDOMReconnaissanceCoverageConfidence(len(candidates), report.CandidateBodyBearingCount)
	report.AuthorConfidence = renderedDOMReconnaissanceCoverageConfidence(len(candidates), report.CandidateAuthorBearingCount)
	report.MachineTimestampConfidence = renderedDOMReconnaissanceCoverageConfidence(len(candidates), report.CandidateMachineTimestampCount)
	if report.CandidateMachineTimestampCount == 0 && report.CandidateRelativeTimeOnlyCount > 0 {
		report.MachineTimestampConfidence = RenderedDOMReconnaissanceTentative
	}
	report.MarkerCategories = renderedDOMReconnaissanceMarkerSet(markers)
	return report, nil
}

func renderedDOMReconnaissanceCoverageConfidence(total, covered int) RenderedDOMReconnaissanceConfidence {
	if covered == 0 {
		return RenderedDOMReconnaissanceNotFound
	}
	if total > 0 && covered == total {
		return RenderedDOMReconnaissanceStrong
	}
	return RenderedDOMReconnaissanceTentative
}

func renderedDOMReconnaissanceCandidateNode(node renderedDOMReconnaissanceNode) bool {
	return node.tag == "article" || strings.EqualFold(node.attrs["role"], "article")
}

func renderedDOMReconnaissanceHasCandidateAncestor(nodes []renderedDOMReconnaissanceNode, parent int) bool {
	for parent >= 0 {
		if renderedDOMReconnaissanceCandidateNode(nodes[parent]) {
			return true
		}
		parent = nodes[parent].parent
	}
	return false
}

func renderedDOMReconnaissanceNearestCandidate(nodes []renderedDOMReconnaissanceNode, candidates map[int]int, node int) (int, bool) {
	for node >= 0 {
		if candidate, ok := candidates[node]; ok {
			return candidate, true
		}
		node = nodes[node].parent
	}
	return 0, false
}

func renderedDOMReconnaissanceGroupKey(value string) (string, bool) {
	if value == "" || len([]byte(value)) > renderedDOMReconnaissanceMaxPageURLBytes || value != strings.TrimSpace(value) {
		return "", false
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || !renderedDOMReconnaissanceFacebookHost(parsed.Hostname()) {
		return "", false
	}
	segments := renderedDOMReconnaissancePathSegments(parsed.Path)
	if len(segments) < 2 || segments[0] != "groups" || segments[1] == "" {
		return "", false
	}
	return segments[1], true
}

func renderedDOMReconnaissanceClassifyHref(value string) renderedDOMReconnaissanceHrefEvidence {
	if value == "" || len([]byte(value)) > renderedDOMReconnaissanceMaxPageURLBytes {
		return renderedDOMReconnaissanceHrefEvidence{}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || (parsed.IsAbs() && (!strings.EqualFold(parsed.Scheme, "https") || !renderedDOMReconnaissanceFacebookHost(parsed.Hostname()))) {
		return renderedDOMReconnaissanceHrefEvidence{}
	}
	segments := renderedDOMReconnaissancePathSegments(parsed.Path)
	if len(segments) >= 4 && segments[0] == "groups" && segments[2] == "posts" && segments[1] != "" && segments[3] != "" {
		return renderedDOMReconnaissanceHrefEvidence{
			marker:    "href=/groups/<group>/posts/<post>/",
			groupKey:  segments[1],
			permalink: true,
		}
	}
	if len(segments) >= 4 && segments[0] == "groups" && segments[2] == "user" && segments[1] != "" && segments[3] != "" {
		return renderedDOMReconnaissanceHrefEvidence{
			marker:   "href=/groups/<group>/user/<author>/",
			groupKey: segments[1],
			author:   true,
		}
	}
	if parsed.Path == "/profile.php" && parsed.Query().Has("id") {
		return renderedDOMReconnaissanceHrefEvidence{marker: "href=/profile.php?id=<author>", author: true}
	}
	if parsed.Query().Has("story_fbid") {
		return renderedDOMReconnaissanceHrefEvidence{marker: "href=?story_fbid=<post>", permalink: true}
	}
	return renderedDOMReconnaissanceHrefEvidence{}
}

func renderedDOMReconnaissanceFacebookHost(host string) bool {
	switch strings.ToLower(host) {
	case "facebook.com", "www.facebook.com", "m.facebook.com":
		return true
	default:
		return false
	}
}

func renderedDOMReconnaissancePathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := parts[:0]
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func renderedDOMReconnaissanceMarkerSet(markers map[string]struct{}) []string {
	values := make([]string, 0, len(markers))
	for marker := range markers {
		values = append(values, marker)
	}
	return boundedRenderedDOMReconnaissanceMarkers(values)
}

func boundedRenderedDOMReconnaissanceMarkers(markers []string) []string {
	bounded := make([]string, 0, len(markers))
	seen := make(map[string]struct{})
	for _, marker := range markers {
		if marker == "" || len(marker) > RenderedDOMReconnaissanceMaxMarkerLength {
			continue
		}
		if _, exists := seen[marker]; exists {
			continue
		}
		seen[marker] = struct{}{}
		bounded = append(bounded, marker)
	}
	sort.Strings(bounded)
	if len(bounded) > RenderedDOMReconnaissanceMaxMarkerCategories {
		bounded = bounded[:RenderedDOMReconnaissanceMaxMarkerCategories]
	}
	return bounded
}

func parseRenderedDOMReconnaissanceNodes(input string) []renderedDOMReconnaissanceNode {
	lower := strings.ToLower(input)
	nodes := make([]renderedDOMReconnaissanceNode, 0, 4096)
	stack := make([]int, 0, 64)
	for index := 0; index < len(input); {
		if len(stack) > 0 {
			tag := nodes[stack[len(stack)-1]].tag
			if tag == "script" || tag == "style" {
				closing := strings.Index(lower[index:], "</"+tag)
				if closing < 0 {
					break
				}
				index += closing
			}
		}
		opening := strings.IndexByte(input[index:], '<')
		if opening < 0 {
			break
		}
		index += opening
		if strings.HasPrefix(input[index:], "<!--") {
			end := strings.Index(input[index+4:], "-->")
			if end < 0 {
				break
			}
			index += end + 7
			continue
		}
		end := renderedDOMReconnaissanceTagEnd(input, index+1)
		if end < 0 {
			break
		}
		token := strings.TrimSpace(input[index+1 : end])
		index = end + 1
		if token == "" || token[0] == '!' || token[0] == '?' {
			continue
		}
		if token[0] == '/' {
			fields := strings.Fields(strings.TrimSpace(token[1:]))
			if len(fields) == 0 {
				continue
			}
			closingTag := strings.ToLower(fields[0])
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if nodes[top].tag == closingTag {
					break
				}
			}
			continue
		}

		tag, attrs, selfClosing := renderedDOMReconnaissanceStartTag(token)
		if tag == "" {
			continue
		}
		parent := -1
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		nodes = append(nodes, renderedDOMReconnaissanceNode{tag: tag, attrs: attrs, parent: parent})
		if !selfClosing && !renderedDOMReconnaissanceVoidTag(tag) {
			stack = append(stack, len(nodes)-1)
		}
	}
	return nodes
}

func renderedDOMReconnaissanceTagEnd(input string, start int) int {
	var quote byte
	for index := start; index < len(input); index++ {
		current := input[index]
		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		if current == '>' {
			return index
		}
	}
	return -1
}

func renderedDOMReconnaissanceStartTag(token string) (string, map[string]string, bool) {
	selfClosing := strings.HasSuffix(strings.TrimSpace(token), "/")
	position := 0
	for position < len(token) && !renderedDOMReconnaissanceSpace(token[position]) && token[position] != '/' {
		position++
	}
	if position == 0 {
		return "", nil, false
	}
	tag := strings.ToLower(token[:position])
	attrs := make(map[string]string)
	for position < len(token) {
		for position < len(token) && (renderedDOMReconnaissanceSpace(token[position]) || token[position] == '/') {
			position++
		}
		start := position
		for position < len(token) && !renderedDOMReconnaissanceSpace(token[position]) && token[position] != '=' && token[position] != '/' {
			position++
		}
		if start == position {
			break
		}
		name := strings.ToLower(token[start:position])
		for position < len(token) && renderedDOMReconnaissanceSpace(token[position]) {
			position++
		}
		value := ""
		if position < len(token) && token[position] == '=' {
			position++
			for position < len(token) && renderedDOMReconnaissanceSpace(token[position]) {
				position++
			}
			value, position = renderedDOMReconnaissanceAttributeValue(token, position)
		}
		switch name {
		case "role", "href", "data-ad-preview", "data-ad-comet-preview", "datetime", "data-utime":
			attrs[name] = value
		}
	}
	return tag, attrs, selfClosing
}

func renderedDOMReconnaissanceAttributeValue(token string, position int) (string, int) {
	if position >= len(token) {
		return "", position
	}
	if token[position] == '\'' || token[position] == '"' {
		quote := token[position]
		position++
		start := position
		for position < len(token) && token[position] != quote {
			position++
		}
		value := token[start:position]
		if position < len(token) {
			position++
		}
		return value, position
	}
	start := position
	for position < len(token) && !renderedDOMReconnaissanceSpace(token[position]) && token[position] != '/' {
		position++
	}
	return token[start:position], position
}

func renderedDOMReconnaissanceSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t' || value == '\f'
}

func renderedDOMReconnaissanceVoidTag(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
