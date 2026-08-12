package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

const (
	OperationWatchedGroupsList      = "watched_groups_list"
	OperationWatchedGroupsAdd       = "watched_groups_add"
	OperationWatchedGroupsSetActive = "watched_groups_set_active"
	OperationWatchedGroupsNextFive  = "watched_groups_next_five"

	WatchedGroupsStatusOK    = "ok"
	WatchedGroupsStatusError = "error"

	WatchedGroupsErrorInvalidGroup       = "invalid_group"
	WatchedGroupsErrorDuplicateGroup     = "duplicate_group"
	WatchedGroupsErrorGroupNotFound      = "group_not_found"
	WatchedGroupsErrorInsufficientActive = "insufficient_active_groups"
	WatchedGroupsErrorInvalidCursor      = "invalid_cursor"

	MaxWatchedGroupsRequestBytes  = 1024 * 1024
	MaxWatchedGroupsResponseBytes = 1024 * 1024
)

type WatchedGroupBridgeValue struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CanonicalURL string `json:"canonical_url"`
	CreatedAt    string `json:"created_at"`
	Active       bool   `json:"active"`
}

type AddWatchedGroupBridgeValue struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CanonicalURL string `json:"canonical_url"`
	CreatedAt    string `json:"created_at"`
}

type WatchedGroupsRequest struct {
	SchemaVersion int                         `json:"schema_version"`
	Operation     string                      `json:"operation"`
	Groups        []WatchedGroupBridgeValue   `json:"groups"`
	Cursor        int                         `json:"cursor"`
	NewGroup      *AddWatchedGroupBridgeValue `json:"new_group,omitempty"`
	GroupID       string                      `json:"group_id,omitempty"`
	Active        *bool                       `json:"active,omitempty"`
}

type WatchedGroupsResponse struct {
	SchemaVersion int                       `json:"schema_version"`
	Operation     string                    `json:"operation"`
	Status        string                    `json:"status"`
	Groups        []WatchedGroupBridgeValue `json:"groups"`
	Selection     []WatchedGroupBridgeValue `json:"selection,omitempty"`
	NextCursor    *int                      `json:"next_cursor,omitempty"`
	ErrorCode     string                    `json:"error_code,omitempty"`
}

func DecodeWatchedGroupsRequest(reader io.Reader) (WatchedGroupsRequest, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, MaxWatchedGroupsRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > MaxWatchedGroupsRequestBytes {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request WatchedGroupsRequest
	if err := decoder.Decode(&request); err != nil {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}
	if request.SchemaVersion != SchemaVersion {
		return WatchedGroupsRequest{}, ErrUnsupportedSchemaVersion
	}
	if !isWatchedGroupsOperation(request.Operation) {
		return WatchedGroupsRequest{}, ErrUnsupportedOperation
	}
	return request, nil
}

func HandleWatchedGroups(request WatchedGroupsRequest) WatchedGroupsResponse {
	response := WatchedGroupsResponse{
		SchemaVersion: SchemaVersion,
		Operation:     request.Operation,
		Status:        WatchedGroupsStatusOK,
		Groups:        []WatchedGroupBridgeValue{},
	}

	collection, err := collectionFromBridgeValues(request.Groups)
	if err != nil {
		return watchedGroupsErrorResponse(request.Operation, err)
	}

	switch request.Operation {
	case OperationWatchedGroupsList:
		response.Groups = bridgeValuesFromGroups(collection.Groups())
	case OperationWatchedGroupsAdd:
		if request.NewGroup == nil {
			return watchedGroupsErrorResponse(request.Operation, domain.ErrMissingWatchedGroupIdentity)
		}
		group, err := groupFromAddBridgeValue(*request.NewGroup)
		if err != nil {
			return watchedGroupsErrorResponse(request.Operation, err)
		}
		if err := collection.Add(group); err != nil {
			return watchedGroupsErrorResponse(request.Operation, err)
		}
		response.Groups = bridgeValuesFromGroups(collection.Groups())
	case OperationWatchedGroupsSetActive:
		if request.Active == nil {
			return watchedGroupsErrorResponse(request.Operation, domain.ErrInvalidWatchedGroupID)
		}
		if *request.Active {
			_, err = collection.Activate(request.GroupID)
		} else {
			_, err = collection.Deactivate(request.GroupID)
		}
		if err != nil {
			return watchedGroupsErrorResponse(request.Operation, err)
		}
		response.Groups = bridgeValuesFromGroups(collection.Groups())
	case OperationWatchedGroupsNextFive:
		cursor, err := application.NewWatchedGroupSelectionCursor(request.Cursor)
		if err != nil {
			return watchedGroupsErrorResponse(request.Operation, err)
		}
		selection, err := application.SelectNextFiveActiveGroups(collection.Groups(), cursor)
		if err != nil {
			return watchedGroupsErrorResponse(request.Operation, err)
		}
		nextCursor := selection.NextCursor().Position()
		response.Groups = bridgeValuesFromGroups(collection.Groups())
		response.Selection = bridgeValuesFromGroups(selection.Groups())
		response.NextCursor = &nextCursor
	}

	return response
}

func EncodeWatchedGroupsResponse(response WatchedGroupsResponse) ([]byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(payload)+1 > MaxWatchedGroupsResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return append(payload, '\n'), nil
}

func ServeWatchedGroups(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	request, err := DecodeWatchedGroupsRequest(stdin)
	if err != nil {
		writeDiagnostic(stderr, "watched groups request rejected")
		return 2
	}

	payload, err := EncodeWatchedGroupsResponse(HandleWatchedGroups(request))
	if err != nil {
		writeDiagnostic(stderr, "watched groups response failed")
		return 3
	}
	if _, err := stdout.Write(payload); err != nil {
		writeDiagnostic(stderr, "watched groups response write failed")
		return 3
	}
	return 0
}

func isWatchedGroupsOperation(operation string) bool {
	switch operation {
	case OperationWatchedGroupsList,
		OperationWatchedGroupsAdd,
		OperationWatchedGroupsSetActive,
		OperationWatchedGroupsNextFive:
		return true
	default:
		return false
	}
}

func collectionFromBridgeValues(values []WatchedGroupBridgeValue) (*application.WatchedGroupCollection, error) {
	collection := application.NewWatchedGroupCollection()
	for _, value := range values {
		group, err := groupFromBridgeValue(value)
		if err != nil {
			return nil, err
		}
		if err := collection.Add(group); err != nil {
			return nil, err
		}
	}
	return collection, nil
}

func groupFromBridgeValue(value WatchedGroupBridgeValue) (domain.WatchedGroup, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, value.CreatedAt)
	if err != nil {
		return domain.WatchedGroup{}, domain.ErrInvalidWatchedGroupCreatedAt
	}
	group, err := domain.NewWatchedGroup(value.ID, "", value.CanonicalURL, value.Name, createdAt)
	if err != nil {
		return domain.WatchedGroup{}, err
	}
	return group.WithActive(value.Active), nil
}

func groupFromAddBridgeValue(value AddWatchedGroupBridgeValue) (domain.WatchedGroup, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, value.CreatedAt)
	if err != nil {
		return domain.WatchedGroup{}, domain.ErrInvalidWatchedGroupCreatedAt
	}
	return domain.NewWatchedGroup(value.ID, "", value.CanonicalURL, value.Name, createdAt)
}

func bridgeValuesFromGroups(groups []domain.WatchedGroup) []WatchedGroupBridgeValue {
	values := make([]WatchedGroupBridgeValue, len(groups))
	for i, group := range groups {
		values[i] = WatchedGroupBridgeValue{
			ID:           group.ID(),
			Name:         group.Name(),
			CanonicalURL: group.CanonicalURL(),
			CreatedAt:    group.CreatedAt().Format(time.RFC3339Nano),
			Active:       group.IsActive(),
		}
	}
	return values
}

func watchedGroupsErrorResponse(operation string, err error) WatchedGroupsResponse {
	return WatchedGroupsResponse{
		SchemaVersion: SchemaVersion,
		Operation:     operation,
		Status:        WatchedGroupsStatusError,
		Groups:        []WatchedGroupBridgeValue{},
		ErrorCode:     watchedGroupsErrorCode(err),
	}
}

func watchedGroupsErrorCode(err error) string {
	switch {
	case errors.Is(err, application.ErrDuplicateWatchedGroupID),
		errors.Is(err, application.ErrDuplicateWatchedGroupIdentity),
		errors.Is(err, application.ErrDuplicateWatchedGroupSelectionGroup):
		return WatchedGroupsErrorDuplicateGroup
	case errors.Is(err, application.ErrWatchedGroupNotFound):
		return WatchedGroupsErrorGroupNotFound
	case errors.Is(err, application.ErrInsufficientActiveWatchedGroups),
		errors.Is(err, application.ErrEmptyWatchedGroupSelectionCollection):
		return WatchedGroupsErrorInsufficientActive
	case errors.Is(err, application.ErrInvalidWatchedGroupSelectionCursor):
		return WatchedGroupsErrorInvalidCursor
	default:
		return WatchedGroupsErrorInvalidGroup
	}
}
