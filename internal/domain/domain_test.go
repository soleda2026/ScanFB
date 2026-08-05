package domain

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRawPostAndAuthorIdentityCanRepresentValidValues(t *testing.T) {
	createdAt := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC)
	capturedAt := time.Date(2026, 8, 5, 9, 31, 0, 0, time.UTC)

	post := RawPost{
		PostID:    "post-1",
		GroupID:   "group-1",
		GroupName: "MacBook Buyers",
		PostURL:   "https://example.test/posts/post-1",
		Author: AuthorIdentity{
			FacebookUserID:      "user-1",
			CanonicalProfileURL: "https://example.test/user-1",
			Username:            "buyer.one",
			DisplayName:         "Buyer One",
		},
		Body:       "can mua MacBook",
		CreatedAt:  createdAt,
		CapturedAt: capturedAt,
	}

	if post.PostID != "post-1" || post.Author.DisplayName != "Buyer One" {
		t.Fatalf("RawPost did not preserve supplied values: %#v", post)
	}
	if !post.CreatedAt.Equal(createdAt) || !post.CapturedAt.Equal(capturedAt) {
		t.Fatalf("RawPost timestamps were not preserved: %#v", post)
	}
}

func TestRawPostAndAuthorIdentityDoNotDefineCredentialFields(t *testing.T) {
	for _, structName := range []string{"RawPost", "AuthorIdentity"} {
		fields := structFieldNames(t, "raw_post.go", structName)
		for _, field := range fields {
			normalized := strings.ToLower(field)
			for _, forbidden := range []string{"password", "cookie", "token", "email", "phone", "profileimage"} {
				if strings.Contains(normalized, forbidden) {
					t.Fatalf("%s defines credential or sensitive field %q", structName, field)
				}
			}
		}
	}
}

func TestMacBookSearchProfile(t *testing.T) {
	profile := MacBookSearchProfile()

	if profile.ID() != macBookSearchProfileID {
		t.Fatalf("MacBook profile ID = %q, want %q", profile.ID(), macBookSearchProfileID)
	}
	if profile.DisplayName() != "MacBook" {
		t.Fatalf("MacBook profile display name = %q", profile.DisplayName())
	}
	if len(profile.ProductTerms()) == 0 {
		t.Fatal("MacBook profile must contain at least one product term")
	}
	if !profile.IsEnabled() {
		t.Fatal("MacBook profile should be enabled")
	}
}

func TestSearchProfileCopiesInputAndReturnedSlices(t *testing.T) {
	productTerms := []string{"MacBook"}
	buyerTerms := []string{"can mua"}
	noiseTerms := []string{"can tien nen ban"}

	profile, err := NewSearchProfile("custom", "Custom", productTerms, buyerTerms, noiseTerms, true)
	if err != nil {
		t.Fatalf("NewSearchProfile returned error: %v", err)
	}

	productTerms[0] = "iPhone"
	buyerTerms[0] = "changed"
	noiseTerms[0] = "changed"

	if got := profile.ProductTerms()[0]; got != "MacBook" {
		t.Fatalf("ProductTerms changed through input slice mutation: %q", got)
	}
	if got := profile.BuyerIntentTerms()[0]; got != "can mua" {
		t.Fatalf("BuyerIntentTerms changed through input slice mutation: %q", got)
	}
	if got := profile.NoiseTerms()[0]; got != "can tien nen ban" {
		t.Fatalf("NoiseTerms changed through input slice mutation: %q", got)
	}

	returned := profile.ProductTerms()
	returned[0] = "iPhone"
	if got := profile.ProductTerms()[0]; got != "MacBook" {
		t.Fatalf("ProductTerms changed through returned slice mutation: %q", got)
	}
}

func TestGeographicModeValidation(t *testing.T) {
	validModes := []GeographicMode{
		GeographicModeHoChiMinhCity,
		GeographicModeOutsideHoChiMinhCityVN,
		GeographicModeAllVietnam,
	}

	for _, mode := range validModes {
		got, err := NewGeographicMode(string(mode))
		if err != nil {
			t.Fatalf("NewGeographicMode(%q) returned error: %v", mode, err)
		}
		if got != mode {
			t.Fatalf("NewGeographicMode(%q) = %q", mode, got)
		}
	}

	if _, err := NewGeographicMode("outside_vietnam"); !errors.Is(err, ErrInvalidGeographicMode) {
		t.Fatalf("NewGeographicMode invalid error = %v, want %v", err, ErrInvalidGeographicMode)
	}
}

func TestScanWindowValidation(t *testing.T) {
	loc := mustHoChiMinhLocation(t)
	scanDate := time.Date(2026, 8, 5, 0, 0, 0, 0, loc)
	startOfDay := time.Date(2026, 8, 5, 0, 0, 0, 0, loc)
	scanStarted := time.Date(2026, 8, 5, 11, 45, 0, 0, loc)

	window, err := NewScanWindow(scanDate, startOfDay, scanStarted)
	if err != nil {
		t.Fatalf("NewScanWindow valid returned error: %v", err)
	}
	if window.Timezone() != RequiredTimezone {
		t.Fatalf("Timezone = %q, want %q", window.Timezone(), RequiredTimezone)
	}
	if !window.StartOfDay().Equal(startOfDay) || !window.ScanStarted().Equal(scanStarted) {
		t.Fatalf("ScanWindow did not preserve supplied bounds: %#v", window)
	}

	utcDate := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	if _, err := NewScanWindow(utcDate, startOfDay, scanStarted); !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("NewScanWindow UTC error = %v, want %v", err, ErrInvalidTimezone)
	}

	lateStartOfDay := time.Date(2026, 8, 5, 12, 0, 0, 0, loc)
	if _, err := NewScanWindow(scanDate, lateStartOfDay, scanStarted); !errors.Is(err, ErrStartOfDayAfterScanStart) {
		t.Fatalf("NewScanWindow late start error = %v, want %v", err, ErrStartOfDayAfterScanStart)
	}

	previousDayStart := time.Date(2026, 8, 4, 0, 0, 0, 0, loc)
	if _, err := NewScanWindow(scanDate, previousDayStart, scanStarted); !errors.Is(err, ErrScanWindowCrossesDay) {
		t.Fatalf("NewScanWindow cross-day error = %v, want %v", err, ErrScanWindowCrossesDay)
	}
}

func TestScanRequestGroupInvariants(t *testing.T) {
	window := validScanWindow(t)
	profile := MacBookSearchProfile()
	mode := GeographicModeHoChiMinhCity

	tests := []struct {
		name     string
		groupIDs []string
		wantErr  error
	}{
		{name: "one group", groupIDs: []string{"group-1"}},
		{name: "five groups", groupIDs: []string{"group-1", "group-2", "group-3", "group-4", "group-5"}},
		{name: "zero groups", groupIDs: nil, wantErr: ErrNoScanGroups},
		{name: "six groups", groupIDs: []string{"group-1", "group-2", "group-3", "group-4", "group-5", "group-6"}, wantErr: ErrTooManyScanGroups},
		{name: "empty group", groupIDs: []string{"group-1", " "}, wantErr: ErrEmptyScanGroupID},
		{name: "duplicate group", groupIDs: []string{"group-1", "group-1"}, wantErr: ErrDuplicateScanGroupID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScanRequest(profile, mode, window, tt.groupIDs, true)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewScanRequest error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestScanRequestCopiesInputAndReturnedGroupIDs(t *testing.T) {
	input := []string{" group-1 "}
	request, err := NewScanRequest(MacBookSearchProfile(), GeographicModeHoChiMinhCity, validScanWindow(t), input, false)
	if err != nil {
		t.Fatalf("NewScanRequest returned error: %v", err)
	}

	input[0] = "group-2"
	if got := request.GroupIDs()[0]; got != "group-1" {
		t.Fatalf("GroupIDs changed through input slice mutation: %q", got)
	}

	returned := request.GroupIDs()
	returned[0] = "group-3"
	if got := request.GroupIDs()[0]; got != "group-1" {
		t.Fatalf("GroupIDs changed through returned slice mutation: %q", got)
	}

	if request.DryRun() {
		t.Fatal("DryRun should preserve explicit false value")
	}
}

func TestDomainPackageImportsOnlyStandardLibrary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob domain files: %v", err)
	}

	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("domain file %s imports non-standard package %q", path, importPath)
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("domain file %s imports internal package %q", path, importPath)
			}
		}
	}
}

func structFieldNames(t *testing.T, path string, structName string) []string {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var fields []string
	for _, decl := range parsed.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s is not a struct", structName)
			}
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					fields = append(fields, name.Name)
				}
			}
			return fields
		}
	}

	t.Fatalf("struct %s not found in %s", structName, path)
	return nil
}

func validScanWindow(t *testing.T) ScanWindow {
	t.Helper()

	loc := mustHoChiMinhLocation(t)
	window, err := NewScanWindow(
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 10, 0, 0, 0, loc),
	)
	if err != nil {
		t.Fatalf("validScanWindow setup failed: %v", err)
	}
	return window
}

func mustHoChiMinhLocation(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(RequiredTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", RequiredTimezone, err)
	}
	return loc
}
