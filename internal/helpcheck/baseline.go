package helpcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"doc-html-translate/internal/i18n"
)

// ── the baseline: locales the GUI ships and the help does not yet have ──

// Options carries what the judging cannot read from the tree itself.
type Options struct {
	// Committed returns the content of a repository file as HEAD holds it. found is false when HEAD has no such
	// file; err is for a result that cannot be had at all (no git, not a checkout), which ends the run as COULD NOT
	// VERIFY. Nil means the run has no committed state to compare with (a scratch copy of the tree), and the
	// baseline is judged by its own rules only.
	Committed func(rel string) (raw []byte, found bool, err error)

	// StrictFreshness makes a set older than its inputs a failure instead of an advisory: the full gate and the
	// release pre-flight set it (ticket E0125).
	StrictFreshness bool
}

// baseline is configs/help-baseline.txt read apart: the comment lines, in order, and the identifiers, in file order.
type baseline struct {
	comments []string
	ids      []string
}

func parseBaseline(raw []byte) baseline {
	var b baseline
	for _, l := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		switch l = strings.TrimSpace(l); {
		case l == "":
		case strings.HasPrefix(l, "#"):
			b.comments = append(b.comments, l)
		default:
			b.ids = append(b.ids, l)
		}
	}
	return b
}

// acceptedRe is the journal line the update path writes above the identifiers: which identifier was accepted, when,
// and why (CHECK-BASELINE rule 6). A line without a reason is no record.
var acceptedRe = regexp.MustCompile(`^#\s*accepted (\S+) on \d{4}-\d{2}-\d{2}: (\S.*)$`)

func (b baseline) reasons() map[string]string {
	out := map[string]string{}
	for _, c := range b.comments {
		if m := acceptedRe.FindStringSubmatch(c); m != nil {
			out[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return out
}

// A set baseline (CHECK-BASELINE rule 3): one identifier per line, sorted, "locale:<code>". The set may fall and
// never rise: a GUI locale with no pages that the file does not list is new debt and fails; a listed locale that now
// has pages is named so the line can be pruned. Against the committed file (rule 4) a line the file did not hold
// before is an absorption: it fails unless the update path recorded a reason for it.
func checkBaseline(r *Report, root string, m *Map, opt Options) error {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(BaselinePath)))
	if err != nil {
		r.fail("%s is missing: the locales without pages are accepted there, one per line (CHECK-BASELINE)", BaselinePath)
		return nil
	}
	cur := parseBaseline(raw)
	if !sort.StringsAreSorted(cur.ids) {
		r.fail("%s is not sorted; keep it sorted so a change reads in a diff (CHECK-BASELINE rule 3)", BaselinePath)
	}
	accepted := map[string]bool{}
	for _, l := range cur.ids {
		accepted[l] = true
	}
	missing := map[string]bool{}
	for _, code := range i18n.Codes {
		if !contains(m.Locales, code) {
			id := "locale:" + code
			missing[id] = true
			if !accepted[id] {
				r.fail("the GUI ships %s but the help has no pages in it and %s does not accept that: new debt may not be added (CHECK-BASELINE rule 2)", code, BaselinePath)
			}
		}
	}
	for _, l := range cur.ids {
		if !missing[l] {
			r.advise("%s lists %s, which is no longer a finding: prune the line", BaselinePath, l)
		} else {
			r.Debt++
		}
	}
	if opt.Committed == nil {
		return nil
	}
	snap, found, err := opt.Committed(BaselinePath)
	if err != nil {
		return fmt.Errorf("the committed %s cannot be read: %w", BaselinePath, err)
	}
	if !found {
		r.advise("%s is not in HEAD yet, so there is no committed state to compare it with (CHECK-BASELINE rule 4)", BaselinePath)
		return nil
	}
	before := map[string]bool{}
	for _, id := range parseBaseline(snap).ids {
		before[id] = true
	}
	reasons := cur.reasons()
	var withReason, without []string
	for _, id := range cur.ids {
		switch {
		case before[id]:
		case reasons[id] != "":
			withReason = append(withReason, id)
		default:
			without = append(without, id)
		}
	}
	if len(without) > 0 {
		r.fail("%s gains %d line(s) over the committed file with no recorded reason: %s - accepting debt is done on purpose with `go run ./tools/helpcheck -accept-baseline \"<reason>\"` (CHECK-BASELINE rules 4 and 6)",
			BaselinePath, len(without), strings.Join(without, ", "))
	}
	if len(withReason) > 0 {
		r.advise("%s gains %d accepted line(s) over the committed file: %s - review the reason in the diff before committing", BaselinePath, len(withReason), strings.Join(withReason, ", "))
	}
	return nil
}

// Accept is the update path of the baseline (CHECK-BASELINE rule 6). It finds the locales the GUI ships, the help has
// no pages in and the file does not accept yet, writes each as a line of the set with a dated journal comment that
// carries the reason, and returns the identifiers it accepted, for the caller to print. The reason is required.
func Accept(root, reason string, now time.Time) ([]string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || strings.ContainsAny(reason, "\r\n") {
		return nil, fmt.Errorf("the reason is required and is one line")
	}
	mapRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(MapPath)))
	if err != nil {
		return nil, err
	}
	var m Map
	if err := json.NewDecoder(bytes.NewReader(mapRaw)).Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", MapPath, err)
	}
	path := filepath.Join(root, filepath.FromSlash(BaselinePath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cur := parseBaseline(raw)
	have := map[string]bool{}
	for _, id := range cur.ids {
		have[id] = true
	}
	var added []string
	for _, code := range i18n.Codes {
		if id := "locale:" + code; !contains(m.Locales, code) && !have[id] {
			added = append(added, id)
		}
	}
	if len(added) == 0 {
		return nil, nil
	}
	day := now.Format("2006-01-02")
	var out strings.Builder
	for _, c := range cur.comments {
		out.WriteString(c + "\n")
	}
	for _, id := range added {
		out.WriteString(fmt.Sprintf("# accepted %s on %s: %s\n", id, day, reason))
	}
	ids := append(append([]string{}, cur.ids...), added...)
	sort.Strings(ids)
	for _, id := range ids {
		out.WriteString(id + "\n")
	}
	return added, os.WriteFile(path, []byte(out.String()), 0o644)
}
