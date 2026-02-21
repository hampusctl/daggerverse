# JSON → Markdown flow

This document describes how Grant’s JSON report is turned into Markdown: where it happens, what structures are used, and the order of steps in code. The converter lives in **`grant/internal/report`**; **`grant/main.go`** only calls it from `Check()`.

---

## 1. Where the flow runs

Conversion runs **on the host** (your machine), not inside the Grant container.

1. **Check()** runs Grant in a container with `--output json --output-file /tmp/report.json`.
2. The host reads that file: `ctr.File("/tmp/report.json").Contents(ctx)` (Dagger SDK).
3. **report.ToMarkdown(jsonBytes)** parses the JSON, builds template data, and renders Markdown.
4. The host writes the result: `ctr.Directory("/tmp").WithNewFile("report.md", md)`.
5. The returned directory contains both `report.json` (from Grant) and `report.md` (generated).

The converter lives in `grant/internal/report` (exported `ToMarkdown`). `main.go` only calls `report.ToMarkdown(jsonBytes)` after reading the file from the container.

---

## 2. Input: Grant JSON shape

Grant’s `--output json` looks like this (simplified):

```json
{
  "tool": "grant",
  "version": "0.6.2",
  "run": {
    "targets": [
      {
        "source": { "type": "file", "ref": "sbom.json" },
        "evaluation": {
          "status": "noncompliant",
          "summary": {
            "packages": { "total", "allowed", "denied", "ignored", "unlicensed" },
            "licenses": { "unique", "allowed", "denied", "nonSPDX" }
          },
          "findings": {
            "packages": [
              {
                "id", "name", "type", "version",
                "decision": "allow" | "deny" | ...,
                "licenses": [ { "id", "riskCategory" } ]
              }
            ]
          }
        }
      }
    ]
  }
}
```

We only use **the first target**: `run.targets[0]`. Multiple SBOMs/targets are not merged.

---

## 3. Code structures

### 3.1 grantReport (`internal/report/report.go`)

- **Role:** Exact mirror of the JSON so `encoding/json` can unmarshal into it.
- **Used in:** `ToMarkdown` → `json.Unmarshal(data, &r)`.
- **Fields:** `Tool`, `Version`, `Run.Targets[].Source`, `Run.Targets[].Evaluation.Status`, `Summary` (packages + licenses), `Findings.Packages[]` (each with `ID`, `Name`, `Type`, `Version`, `Decision`, `Licenses[]` with `ID`, `RiskCategory`).

No validation beyond “targets non-empty”; unknown fields are ignored.

### 3.2 defaultGrantReportTmpl (`internal/report/report.go`)

- **Role:** Single Go string with `html/template` syntax. Renders the final Markdown.
- **Sections:**
  1. Title and meta (Tool, Version, Status, Target).
  2. **Summary** – `<details>` with a table: packages (Total, Allowed, Denied, Ignored, Unlicensed) and licenses (Unique, Allowed, Denied, NonSPDX).
  3. **Licenses (summary)** – `<details>` with table: License, Risk, Denied packages (count).
  4. **Denied / non-compliant packages** – `<details>` with table: Name, Version, Type, Licenses (text list).
  5. **Unlicensed packages** – `<details>` with table: Name, Version, Type.

All template data comes from `grantReportTmplData`; the raw `grantReport` is never passed to the template.

### 3.3 grantReportTmplData (`internal/report/report.go`)

- **Role:** View model for the template. Flat, pre-aggregated and sorted.
- **Fields:**
  - `Tool`, `Version`, `Status`, `TargetRef` – from first target.
  - `SummaryPkgs` – Total, Allowed, Denied, Ignored, Unlicensed (ints).
  - `SummaryLics` – Unique, Allowed, Denied, NonSPDX (ints).
  - `LicenseSummary` – slice of `{ ID, RiskCategory, Count }` (one per unique license in denied packages).
  - `LicenseSummaryCount` – `len(LicenseSummary)`.
  - `DeniedCount`, `DeniedPackages` – slice of `{ Name, Version, Type, LicenseList, LicenseSortKey }`.
  - `UnlicensedCount`, `UnlicensedPackages` – slice of `{ Name, Version, Type }`.

---

## 4. ToMarkdown step by step

Function: **ToMarkdown(data []byte) (string, error)** in `internal/report/report.go`.

### Step 1: Parse JSON

- `json.Unmarshal(data, &r)` into a `grantReport`.
- On error: `return "", fmt.Errorf("parse grant report json: %w", err)`.

### Step 2: Parse template

- `template.New("grant").Parse(defaultGrantReportTmpl)`.
- On error: `return "", fmt.Errorf("parse template: %w", err)`.

### Step 3: Copy run metadata and summary

- Take first target: `tgt := &r.Run.Targets[0]` (error if no targets).
- Copy into `td`: Tool, Version, Status, TargetRef.
- Copy `tgt.Evaluation.Summary.Packages` → `td.SummaryPkgs` (field-by-field, no shared type).
- Copy `tgt.Evaluation.Summary.Licenses` → `td.SummaryLics`.

### Step 4: One pass over findings.packages

We iterate **all** `tgt.Evaluation.Findings.Packages` once.

- **Unlicensed:** If `len(p.Licenses) == 0`, append `{ Name, Version, Type }` to `unlicensedPkgs` and `continue`. No license data is used.
- **Not denied:** If `p.Decision` is not `"deny"` or `"denied"`, skip (no append to denied list, no license aggregation).
- **Denied:** For each package with decision deny/denied:
  - For **each** license in `p.Licenses`:
    - Update `licenseAgg[l.ID]` (risk from first occurrence, count incremented for every occurrence).
    - Append to `listParts` and `sortIDs` for this package.
  - Build `LicenseList`: comma-separated `"ID (RiskCategory)"` for display.
  - Build `LicenseSortKey`: sorted license IDs joined by space (for stable sort).
  - Append to `deniedPkgs`: Name, Version, Type, LicenseList, LicenseSortKey.

So: every license on a denied package is counted in `licenseAgg`; packages with multiple licenses get one row in the denied table with all licenses in `LicenseList`.

### Step 5: License summary slice

- From `licenseAgg`, build `td.LicenseSummary`: one entry per license ID with `ID`, `RiskCategory` (risk), `Count` (number of denied packages that have that license).
- Sort by `ID`.
- Set `td.LicenseSummaryCount = len(td.LicenseSummary)`.

### Step 6: Sort denied packages

- Sort `deniedPkgs` by `LicenseSortKey` (primary), then `Name` (secondary).
- Assign to `td.DeniedPackages` and set `td.DeniedCount`.

### Step 7: Sort unlicensed packages

- Sort `unlicensedPkgs` by `Type`, then `Name`.
- Assign to `td.UnlicensedPackages` and set `td.UnlicensedCount`.

### Step 8: Execute template

- `tpl.Execute(&buf, td)` writes the final Markdown into `buf`.
- On error: `return "", fmt.Errorf("execute template: %w", err)`.
- `return buf.String(), nil`.

---

## 5. How Check() uses it (`main.go`)

1. After `WithExec`, read JSON: `jsonBytes, err := ctr.File("/tmp/report.json").Contents(ctx)`.
2. Convert: `md, err := report.ToMarkdown([]byte(jsonBytes))`.
3. Return directory: `ctr.Directory("/tmp").WithNewFile("report.md", md)`.

So the pipeline is: **container writes report.json → host reads it → host produces report.md → host attaches report.md to the same directory → caller gets both files.**

---

## 6. Summary diagram

```
Grant container
  WithExec("grant check ... --output-file /tmp/report.json")
  → /tmp/report.json

Host (report.ToMarkdown)
  json.Unmarshal → grantReport
  → copy metadata + summary to grantReportTmplData
  → one loop over findings.packages:
       unlicensed (licenses==[]) → unlicensedPkgs
       denied (decision deny/denied) → licenseAgg + deniedPkgs
  → build LicenseSummary from licenseAgg, sort
  → sort deniedPkgs (license, name), unlicensedPkgs (type, name)
  → template.Execute(grantReportTmplData) → markdown string

Host (Check)
  ctr.Directory("/tmp").WithNewFile("report.md", md)
  → *dagger.Directory with report.json + report.md
```

This is the full JSON → Markdown flow as implemented in the code.

**Tests:** `internal/report/report_test.go` tests `ToMarkdown` with minimal JSON (summary, denied + unlicensed packages), invalid JSON, empty targets, and multiple licenses per package. Run with `go test ./internal/report/`.
