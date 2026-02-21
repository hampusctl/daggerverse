// Package report converts Grant JSON report output to Markdown.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"
)

// grantReport is the structure of Grant check --output json.
type grantReport struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
	Run     struct {
		Targets []struct {
			Source struct {
				Type string `json:"type"`
				Ref  string `json:"ref"`
			} `json:"source"`
			Evaluation struct {
				Status  string `json:"status"`
				Summary struct {
					Packages struct {
						Total      int `json:"total"`
						Allowed    int `json:"allowed"`
						Denied     int `json:"denied"`
						Ignored    int `json:"ignored"`
						Unlicensed int `json:"unlicensed"`
					} `json:"packages"`
					Licenses struct {
						Unique  int `json:"unique"`
						Allowed int `json:"allowed"`
						Denied  int `json:"denied"`
						NonSPDX int `json:"nonSPDX"`
					} `json:"licenses"`
				} `json:"summary"`
				Findings struct {
					Packages []struct {
						ID       string `json:"id"`
						Name     string `json:"name"`
						Type     string `json:"type"`
						Version  string `json:"version"`
						Decision string `json:"decision"`
						Licenses []struct {
							ID           string `json:"id"`
							RiskCategory string `json:"riskCategory"`
						} `json:"licenses"`
					} `json:"packages"`
				} `json:"findings"`
			} `json:"evaluation"`
		} `json:"targets"`
	} `json:"run"`
}

const defaultGrantReportTmpl = `# Grant License Report

- **Tool:** {{.Tool}} {{.Version}}
- **Status:** {{.Status}}
- **Target:** {{.TargetRef}}

<details>
<summary><strong>Summary</strong> – packages & licenses counts</summary>

| | Packages | Licenses |
|--|----------|----------|
| Total | {{.SummaryPkgs.Total}} | {{.SummaryLics.Unique}} unique |
| Allowed | {{.SummaryPkgs.Allowed}} | {{.SummaryLics.Allowed}} |
| Denied | {{.SummaryPkgs.Denied}} | {{.SummaryLics.Denied}} |
| Ignored | {{.SummaryPkgs.Ignored}} | - |
| Unlicensed | {{.SummaryPkgs.Unlicensed}} | {{.SummaryLics.NonSPDX}} non-SPDX |

</details>

<details>
<summary><strong>Licenses (summary)</strong> – {{.LicenseSummaryCount}} unique licenses in denied packages</summary>

| License | Risk | Denied packages |
|---------|------|-----------------|
{{range .LicenseSummary}}| {{.ID}} | {{.RiskCategory}} | {{.Count}} |
{{end}}

</details>

<details>
<summary><strong>Denied / non-compliant packages</strong> ({{.DeniedCount}}), sorted by license</summary>

| Name | Version | Type | Licenses |
|------|---------|------|----------|
{{range .DeniedPackages}}| {{.Name}} | {{.Version}} | {{.Type}} | {{.LicenseList}} |
{{end}}

</details>

<details>
<summary><strong>Unlicensed packages</strong> ({{.UnlicensedCount}}) – no license info; review or add to policy</summary>

| Name | Version | Type |
|------|---------|------|
{{range .UnlicensedPackages}}| {{.Name}} | {{.Version}} | {{.Type}} |
{{end}}

</details>
`

type grantReportTmplData struct {
	Tool           string
	Version        string
	Status         string
	TargetRef      string
	SummaryPkgs    struct{ Total, Allowed, Denied, Ignored, Unlicensed int }
	SummaryLics    struct{ Unique, Allowed, Denied, NonSPDX int }
	LicenseSummary []struct {
		ID, RiskCategory string
		Count            int
	}
	LicenseSummaryCount int
	DeniedCount         int
	DeniedPackages      []struct{ Name, Version, Type, LicenseList, LicenseSortKey string }
	UnlicensedCount     int
	UnlicensedPackages  []struct{ Name, Version, Type string }
}

// ToMarkdown converts Grant JSON report (--output json) to Markdown.
// It uses the first target only. Returns an error on parse failure or empty targets.
func ToMarkdown(data []byte) (string, error) {
	var r grantReport
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("parse grant report json: %w", err)
	}
	tpl, err := template.New("grant").Parse(defaultGrantReportTmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	td := grantReportTmplData{}
	td.Tool = r.Tool
	td.Version = r.Version
	if len(r.Run.Targets) == 0 {
		return "", fmt.Errorf("grant report has no targets")
	}
	tgt := &r.Run.Targets[0]
	td.Status = tgt.Evaluation.Status
	td.TargetRef = tgt.Source.Ref
	pkgs := tgt.Evaluation.Summary.Packages
	td.SummaryPkgs.Total, td.SummaryPkgs.Allowed, td.SummaryPkgs.Denied = pkgs.Total, pkgs.Allowed, pkgs.Denied
	td.SummaryPkgs.Ignored, td.SummaryPkgs.Unlicensed = pkgs.Ignored, pkgs.Unlicensed
	lics := tgt.Evaluation.Summary.Licenses
	td.SummaryLics.Unique, td.SummaryLics.Allowed, td.SummaryLics.Denied, td.SummaryLics.NonSPDX = lics.Unique, lics.Allowed, lics.Denied, lics.NonSPDX

	// Aggregate licenses from denied packages: id -> { risk, count }
	licenseAgg := make(map[string]struct {
		Risk  string
		Count int
	})
	var deniedPkgs []struct{ Name, Version, Type, LicenseList, LicenseSortKey string }
	var unlicensedPkgs []struct{ Name, Version, Type string }

	for i := range tgt.Evaluation.Findings.Packages {
		p := &tgt.Evaluation.Findings.Packages[i]
		// Unlicensed: no license info at all
		if len(p.Licenses) == 0 {
			unlicensedPkgs = append(unlicensedPkgs, struct{ Name, Version, Type string }{
				Name: p.Name, Version: p.Version, Type: p.Type,
			})
			continue
		}
		if p.Decision != "deny" && p.Decision != "denied" {
			continue
		}
		var listParts []string
		var sortIDs []string
		for _, l := range p.Licenses {
			risk := l.RiskCategory
			if risk == "" {
				risk = "-"
			}
			e, ok := licenseAgg[l.ID]
			if !ok {
				e = struct {
					Risk  string
					Count int
				}{Risk: risk, Count: 0}
			}
			e.Count++
			licenseAgg[l.ID] = e
			listParts = append(listParts, l.ID+" ("+risk+")")
			sortIDs = append(sortIDs, l.ID)
		}
		sort.Strings(sortIDs)
		licenseList := strings.Join(listParts, ", ")
		if licenseList == "" {
			licenseList = "-"
		}
		sortKey := strings.Join(sortIDs, " ")
		if sortKey == "" {
			sortKey = " "
		}
		deniedPkgs = append(deniedPkgs, struct{ Name, Version, Type, LicenseList, LicenseSortKey string }{
			Name: p.Name, Version: p.Version, Type: p.Type, LicenseList: licenseList, LicenseSortKey: sortKey,
		})
	}

	// License summary slice, sorted by license ID
	for id, v := range licenseAgg {
		td.LicenseSummary = append(td.LicenseSummary, struct {
			ID, RiskCategory string
			Count            int
		}{
			ID: id, RiskCategory: v.Risk, Count: v.Count,
		})
	}
	sort.Slice(td.LicenseSummary, func(i, j int) bool { return td.LicenseSummary[i].ID < td.LicenseSummary[j].ID })
	td.LicenseSummaryCount = len(td.LicenseSummary)

	// Sort denied packages by license(s) then name
	sort.Slice(deniedPkgs, func(i, j int) bool {
		if deniedPkgs[i].LicenseSortKey != deniedPkgs[j].LicenseSortKey {
			return deniedPkgs[i].LicenseSortKey < deniedPkgs[j].LicenseSortKey
		}
		return deniedPkgs[i].Name < deniedPkgs[j].Name
	})
	td.DeniedPackages = deniedPkgs
	td.DeniedCount = len(td.DeniedPackages)

	// Sort unlicensed by type then name
	sort.Slice(unlicensedPkgs, func(i, j int) bool {
		if unlicensedPkgs[i].Type != unlicensedPkgs[j].Type {
			return unlicensedPkgs[i].Type < unlicensedPkgs[j].Type
		}
		return unlicensedPkgs[i].Name < unlicensedPkgs[j].Name
	})
	td.UnlicensedPackages = unlicensedPkgs
	td.UnlicensedCount = len(td.UnlicensedPackages)

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
