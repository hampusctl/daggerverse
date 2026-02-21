package report

import (
	"strings"
	"testing"
)

// Minimal valid Grant JSON: one target, summary, one denied and one unlicensed package.
const minimalGrantJSON = `{
  "tool": "grant",
  "version": "0.6.2",
  "run": {
    "targets": [
      {
        "source": { "type": "file", "ref": "sbom.json" },
        "evaluation": {
          "status": "noncompliant",
          "summary": {
            "packages": { "total": 2, "allowed": 0, "denied": 1, "ignored": 0, "unlicensed": 1 },
            "licenses": { "unique": 1, "allowed": 0, "denied": 1, "nonSPDX": 0 }
          },
          "findings": {
            "packages": [
              {
                "id": "apk:pkg-a@1.0",
                "name": "pkg-a",
                "type": "apk",
                "version": "1.0",
                "decision": "deny",
                "licenses": [
                  { "id": "GPL-2.0-only", "riskCategory": "Strong Copyleft (High Risk)" }
                ]
              },
              {
                "id": "go-module:foo/bar@v1.0.0",
                "name": "foo/bar",
                "type": "go-module",
                "version": "v1.0.0",
                "decision": "allow",
                "licenses": []
              }
            ]
          }
        }
      }
    ]
  }
}`

func TestToMarkdown_MinimalValidJSON(t *testing.T) {
	md, err := ToMarkdown([]byte(minimalGrantJSON))
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}

	// Must contain title and meta
	if !strings.Contains(md, "# Grant License Report") {
		t.Error("output missing title")
	}
	if !strings.Contains(md, "grant") || !strings.Contains(md, "0.6.2") {
		t.Error("output missing tool/version")
	}
	if !strings.Contains(md, "noncompliant") {
		t.Error("output missing status")
	}
	if !strings.Contains(md, "sbom.json") {
		t.Error("output missing target ref")
	}

	// Summary section
	if !strings.Contains(md, "Summary") {
		t.Error("output missing Summary section")
	}
	if !strings.Contains(md, "Total") || !strings.Contains(md, "2") {
		t.Error("output missing summary counts")
	}
	if !strings.Contains(md, "Denied") || !strings.Contains(md, "Unlicensed") {
		t.Error("output missing Denied/Unlicensed in summary")
	}

	// License summary (from denied packages)
	if !strings.Contains(md, "Licenses (summary)") {
		t.Error("output missing Licenses (summary) section")
	}
	if !strings.Contains(md, "GPL-2.0-only") {
		t.Error("output missing denied license ID")
	}

	// Denied packages table
	if !strings.Contains(md, "Denied / non-compliant packages") {
		t.Error("output missing Denied packages section")
	}
	if !strings.Contains(md, "pkg-a") || !strings.Contains(md, "1.0") || !strings.Contains(md, "apk") {
		t.Error("output missing denied package name/version/type")
	}
	if !strings.Contains(md, "Strong Copyleft") {
		t.Error("output missing risk category for denied package")
	}

	// Unlicensed packages table
	if !strings.Contains(md, "Unlicensed packages") {
		t.Error("output missing Unlicensed packages section")
	}
	if !strings.Contains(md, "foo/bar") || !strings.Contains(md, "v1.0.0") || !strings.Contains(md, "go-module") {
		t.Error("output missing unlicensed package name/version/type")
	}

	// Collapsible details
	if !strings.Contains(md, "<details>") || !strings.Contains(md, "<summary>") {
		t.Error("output missing details/summary blocks")
	}
}

func TestToMarkdown_InvalidJSON(t *testing.T) {
	_, err := ToMarkdown([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse grant report json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToMarkdown_EmptyTargets(t *testing.T) {
	emptyTargets := `{"tool":"grant","version":"0.1","run":{"targets":[]}}`
	_, err := ToMarkdown([]byte(emptyTargets))
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
	if !strings.Contains(err.Error(), "no targets") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToMarkdown_MultipleLicensesPerPackage(t *testing.T) {
	// One denied package with two licenses; both must appear in license summary and in package row
	json := `{
  "tool": "g",
  "version": "0",
  "run": {
    "targets": [{
      "source": { "type": "file", "ref": "x" },
      "evaluation": {
        "status": "noncompliant",
        "summary": {
          "packages": { "total": 1, "allowed": 0, "denied": 1, "ignored": 0, "unlicensed": 0 },
          "licenses": { "unique": 2, "allowed": 0, "denied": 2, "nonSPDX": 0 }
        },
        "findings": {
          "packages": [{
            "id": "pkg",
            "name": "pkg",
            "type": "apk",
            "version": "1",
            "decision": "deny",
            "licenses": [
              { "id": "MIT", "riskCategory": "Permissive" },
              { "id": "GPL-2.0", "riskCategory": "Copyleft" }
            ]
          }]
        }
      }
    }]
  }
}`
	md, err := ToMarkdown([]byte(json))
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(md, "MIT") || !strings.Contains(md, "GPL-2.0") {
		t.Error("both licenses must appear in output")
	}
	if !strings.Contains(md, "Permissive") || !strings.Contains(md, "Copyleft") {
		t.Error("risk categories must appear")
	}
}
