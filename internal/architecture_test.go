package internal_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var skeletonPackages = []string{
	"./cmd/scanfb",
	"./internal/application",
	"./internal/blocklist",
	"./internal/dedup",
	"./internal/domain",
	"./internal/facebook",
	"./internal/persistence",
	"./internal/rules",
	"./internal/ui",
}

func TestSkeletonPackagesList(t *testing.T) {
	args := append([]string{"list"}, skeletonPackages...)
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list skeleton packages failed: %v\n%s", err, out)
	}
}

func TestScanFBBinaryBuilds(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "scanfb")
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/scanfb")
	cmd.Dir = repoRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/scanfb failed: %v\n%s", err, out)
	}
}

func TestDomainDoesNotImportAdapters(t *testing.T) {
	forbidden := []string{
		"github.com/soleda2026/ScanFB/internal/application",
		"github.com/soleda2026/ScanFB/internal/dedup",
		"github.com/soleda2026/ScanFB/internal/rules",
		"github.com/soleda2026/ScanFB/internal/facebook",
		"github.com/soleda2026/ScanFB/internal/persistence",
		"github.com/soleda2026/ScanFB/internal/ui",
	}

	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "domain", "*.go"))
	if err != nil {
		t.Fatalf("glob domain files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected domain package skeleton files")
	}

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, forbiddenPath := range forbidden {
				if importPath == forbiddenPath {
					t.Fatalf("domain imports forbidden adapter package %q in %s", importPath, path)
				}
			}
		}
	}
}

func TestRulesImportsOnlyDomainAndStandardLibrary(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "rules", "*.go"))
	if err != nil {
		t.Fatalf("glob rules files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected rules package files")
	}

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == "github.com/soleda2026/ScanFB/internal/domain" {
				continue
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("rules imports forbidden internal package %q in %s", importPath, path)
			}
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("rules imports non-standard package %q in %s", importPath, path)
			}
		}
	}
}

func TestDedupImportsOnlyDomainAndStandardLibrary(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "dedup", "*.go"))
	if err != nil {
		t.Fatalf("glob dedup files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected dedup package files")
	}

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == "github.com/soleda2026/ScanFB/internal/domain" {
				continue
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("dedup imports forbidden internal package %q in %s", importPath, path)
			}
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("dedup imports non-standard package %q in %s", importPath, path)
			}
		}
	}
}

func TestBlocklistImportsOnlyDomainAndStandardLibrary(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "blocklist", "*.go"))
	if err != nil {
		t.Fatalf("glob blocklist files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected blocklist package files")
	}

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == "github.com/soleda2026/ScanFB/internal/domain" {
				continue
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("blocklist imports forbidden internal package %q in %s", importPath, path)
			}
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("blocklist imports non-standard package %q in %s", importPath, path)
			}
		}
	}
}

func TestApplicationImportsOnlyAllowedPackages(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "application", "*.go"))
	if err != nil {
		t.Fatalf("glob application files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected application package files")
	}

	allowedInternal := map[string]struct{}{
		"github.com/soleda2026/ScanFB/internal/blocklist": {},
		"github.com/soleda2026/ScanFB/internal/dedup":     {},
		"github.com/soleda2026/ScanFB/internal/domain":    {},
		"github.com/soleda2026/ScanFB/internal/rules":     {},
	}

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if _, ok := allowedInternal[importPath]; ok {
				continue
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("application imports forbidden internal package %q in %s", importPath, path)
			}
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("application imports non-standard package %q in %s", importPath, path)
			}
		}
	}
}

func TestPersistenceImportsOnlyApplicationDomainAndStandardLibrary(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", "persistence", "*.go"))
	if err != nil {
		t.Fatalf("glob persistence files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected persistence package files")
	}

	allowedInternal := map[string]struct{}{
		"github.com/soleda2026/ScanFB/internal/application": {},
		"github.com/soleda2026/ScanFB/internal/domain":      {},
	}

	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", path, err)
		}

		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if _, ok := allowedInternal[importPath]; ok {
				continue
			}
			if strings.HasPrefix(importPath, "github.com/soleda2026/ScanFB/internal/") {
				t.Fatalf("persistence imports forbidden internal package %q in %s", importPath, path)
			}
			firstPart := strings.Split(importPath, "/")[0]
			if strings.Contains(firstPart, ".") {
				t.Fatalf("persistence imports non-standard package %q in %s", importPath, path)
			}
		}
	}
}

func TestCorePackagesDoNotImportPersistence(t *testing.T) {
	corePackages := []string{"application", "blocklist", "dedup", "domain", "rules"}
	fset := token.NewFileSet()

	for _, packageName := range corePackages {
		files, err := filepath.Glob(filepath.Join(repoRoot(), "internal", packageName, "*.go"))
		if err != nil {
			t.Fatalf("glob %s files: %v", packageName, err)
		}
		if len(files) == 0 {
			t.Fatalf("expected %s package files", packageName)
		}

		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			parsed, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse imports for %s: %v", path, err)
			}

			for _, imp := range parsed.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if importPath == "github.com/soleda2026/ScanFB/internal/persistence" {
					t.Fatalf("%s imports persistence in %s", packageName, path)
				}
			}
		}
	}
}

func repoRoot() string {
	return filepath.Clean("..")
}
