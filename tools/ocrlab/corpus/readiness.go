package corpus

import "strings"

// RightsReviewed explicitly rejects the legacy automatic stamp. Synthetic provenance is not
// a third-party rights decision; synthetic scenes still cannot supply independent holdout.
func (s *Scene) RightsReviewed() bool {
	if s.Licence == LicenceSynthetic {
		return true
	}
	v := strings.TrimSpace(s.LicenceVerifiedBy)
	return v != "" && v != "commons-api" && !strings.HasPrefix(v, "tools/")
}

// FamilyProblems follows parents as well as hashes and source-item links, including indirect
// chains. A source family cannot cross splits or count as independent holdout.
func FamilyProblems(m *Manifest) []Problem {
	var out []Problem
	for i := range m.Scenes {
		a := &m.Scenes[i]
		seen := map[string]bool{a.ID: true}
		p := a
		for p.DerivedFrom != "" {
			if seen[p.DerivedFrom] {
				out = append(out, Problem{RuleBadDerivedFrom, a.ID, "cyclic source family"})
				break
			}
			seen[p.DerivedFrom] = true
			p = m.Find(p.DerivedFrom)
			if p == nil {
				break
			}
			if p.Split != a.Split {
				out = append(out, Problem{RuleBadSplit, a.ID, "source family crosses dev and holdout"})
				break
			}
		}
		for j := 0; j < i; j++ {
			b := &m.Scenes[j]
			same := a.SHA256 != "" && a.SHA256 == b.SHA256 || a.SourceItem != "" && a.SourceItem == b.SourceItem || a.SourceURL != "" && a.SourceURL == b.SourceURL
			if same && a.Split != b.Split {
				out = append(out, Problem{RuleBadSplit, a.ID, "duplicate bytes or source crosses splits: " + b.ID})
			}
		}
		if a.Split == SplitHoldout && (a.DerivedFrom != "" || a.Licence == LicenceSynthetic) {
			out = append(out, Problem{RuleBadSplit, a.ID, "derived or synthetic scene is not independent holdout"})
		}
	}
	return out
}
