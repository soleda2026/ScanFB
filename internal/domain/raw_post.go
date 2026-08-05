package domain

import "time"

// AuthorIdentity represents author identity data supplied by the Facebook boundary.
//
// DisplayName is auxiliary data and must not be treated as a stable identity key.
type AuthorIdentity struct {
	FacebookUserID      string
	CanonicalProfileURL string
	Username            string
	DisplayName         string
}

// RawPost represents minimal normalized post input supplied by the Facebook boundary.
//
// It intentionally does not contain inferred classification or filtering results.
type RawPost struct {
	PostID     string
	GroupID    string
	GroupName  string
	PostURL    string
	Author     AuthorIdentity
	Body       string
	CreatedAt  time.Time
	CapturedAt time.Time
}
