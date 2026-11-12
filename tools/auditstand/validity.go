package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// "Sustained" is recorded as five consecutive one-second observations. A monitor gap
// over two seconds invalidates a run even if its cause cannot be distinguished from sleep.
const foreignCPUWindow = 5

// foreignCPUFraction is the strategic section 6 threshold: one foreign process above this share
// of a single core for the whole window makes the run invalid.
const foreignCPUFraction = 0.10

type cpuCounter struct {
	Ticks  uint64 // kernel + user time in 100 ns units
	Parent int
	Image  string // executable file name only; never a command line or a full path
}

// cpuSource reads the per-process CPU counters. It is a variable so the sustained-load logic is
// testable without waiting five real seconds; production uses the platform reader.
var cpuSource = processCPUTimes

type validityWatch struct {
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	excluded map[int]bool
	reasons  []string
	// offenders groups the processes that crossed the window by image name, so the record can
	// say what to quiet instead of leaving the owner a list of PIDs that no longer exist.
	offenders map[string]map[int]bool
	// declared is the named background load the operator declared on -Control (lower-case image
	// file names). A declared image that crosses the window is tolerated - grouped in tolerated
	// and reported as an advisory - instead of invalidating the run (owner decision 2026-10-30).
	declared  map[string]bool
	tolerated map[string]map[int]bool

	// Sampling state; owned by observe and its single caller.
	previous map[int]cpuCounter
	streak   map[int]int
	last     time.Time
}

func newValidityWatch(now time.Time, declared []string) *validityWatch {
	v := &validityWatch{
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		excluded:  map[int]bool{os.Getpid(): true},
		offenders: map[string]map[int]bool{},
		declared:  map[string]bool{},
		tolerated: map[string]map[int]bool{},
		previous:  map[int]cpuCounter{},
		streak:    map[int]int{},
		last:      now,
	}
	for _, image := range declared {
		v.declared[strings.ToLower(image)] = true
	}
	return v
}

func startValidityWatch(declared []string) *validityWatch {
	v := newValidityWatch(time.Now(), declared)
	go func() {
		defer close(v.done)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-v.stop:
				return
			case now := <-tick.C:
				if now.Sub(v.last) > 2*time.Second {
					v.add("validity monitor gap above 2 s (sleep or probe stall)")
				}
				if err := desktopUnlocked(); err != nil {
					v.add("desktop locked or secure desktop unavailable")
				}
				current, err := cpuSource()
				if err != nil {
					v.add("foreign CPU probe failed: " + err.Error())
					continue
				}
				v.observe(now, current)
			}
		}
	}()
	return v
}

// observe folds one CPU snapshot into the per-process streaks. A process counts only when it is
// neither the stand nor a descendant of an excluded process (the app under test and its helpers
// are the subject, not the noise).
func (v *validityWatch) observe(now time.Time, current map[int]cpuCounter) {
	seconds := now.Sub(v.last).Seconds()
	for pid, total := range current {
		v.mu.Lock()
		excluded := v.excluded[pid]
		seen := map[int]bool{}
		for parent := total.Parent; parent != 0 && !seen[parent]; parent = current[parent].Parent {
			seen[parent] = true
			if v.excluded[parent] {
				excluded = true
				break
			}
		}
		v.mu.Unlock()
		old, ok := v.previous[pid]
		if excluded || !ok || total.Ticks < old.Ticks || seconds <= 0 {
			v.streak[pid] = 0
			continue
		}
		fraction := float64(total.Ticks-old.Ticks) / 1e7 / seconds
		if fraction > foreignCPUFraction {
			v.streak[pid]++
		} else {
			v.streak[pid] = 0
		}
		if v.streak[pid] == foreignCPUWindow {
			image := total.Image
			if image == "" {
				image = "unknown image"
			}
			if v.declared[strings.ToLower(image)] {
				v.recordTolerated(pid, image)
			} else {
				v.recordOffender(pid, image)
			}
		}
	}
	// A process that left the snapshot starts from zero if its PID is ever reused.
	for pid := range v.streak {
		if _, alive := current[pid]; !alive {
			delete(v.streak, pid)
		}
	}
	v.previous = current
	v.last = now
}

func (v *validityWatch) recordOffender(pid int, image string) {
	v.add(fmt.Sprintf("foreign PID %d (%s) used over %.0f%% of one core for %d consecutive seconds",
		pid, image, foreignCPUFraction*100, foreignCPUWindow))
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.offenders[image] == nil {
		v.offenders[image] = map[int]bool{}
	}
	v.offenders[image][pid] = true
}

func (v *validityWatch) recordTolerated(pid int, image string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.tolerated[image] == nil {
		v.tolerated[image] = map[int]bool{}
	}
	v.tolerated[image][pid] = true
}

func (v *validityWatch) add(reason string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, r := range v.reasons {
		if r == reason {
			return
		}
	}
	v.reasons = append(v.reasons, reason)
}

func (v *validityWatch) exclude(pid int) { v.mu.Lock(); v.excluded[pid] = true; v.mu.Unlock() }

func (v *validityWatch) finish() []string {
	close(v.stop)
	<-v.done
	return append([]string(nil), v.reasons...)
}

// offenderSummary names the foreign images that crossed the window, most numerous first, so a
// rejected control tells the owner what to close before the next attempt. Image file names
// only: a command line or a window title could carry something private.
func (v *validityWatch) offenderSummary() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.offenders) == 0 {
		return ""
	}
	return "foreign CPU load above the validity threshold, by image: " + byImage(v.offenders)
}

// declaredLoadSummary is the advisory line of a run under a declaration: what was declared and
// which declared images actually crossed the window. Empty when nothing was declared.
func (v *validityWatch) declaredLoadSummary() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.declared) == 0 {
		return ""
	}
	declared := make([]string, 0, len(v.declared))
	for image := range v.declared {
		declared = append(declared, image)
	}
	sort.Strings(declared)
	crossed := "none crossed the window"
	if len(v.tolerated) > 0 {
		crossed = "tolerated above the threshold: " + byImage(v.tolerated)
	}
	return fmt.Sprintf("%s; %s", declaredLoadAdvisory(declared), crossed)
}

// declaredLoadAdvisory is the one wording every record under a declaration carries.
func declaredLoadAdvisory(declared []string) string {
	return "declared background load " + strings.Join(declared, ", ") +
		" is not counted as foreign CPU load (weaker baseline)"
}

// byImage renders image -> PIDs groups most numerous first, then by name.
func byImage(groups map[string]map[int]bool) string {
	type row struct {
		image string
		n     int
	}
	rows := make([]row, 0, len(groups))
	for image, pids := range groups {
		rows = append(rows, row{image, len(pids)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].image < rows[j].image
	})
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%s x%d", r.image, r.n))
	}
	return strings.Join(parts, ", ")
}

// parseDeclaredLoad reads -declared-load: comma-separated executable file names, lower-cased,
// ".exe" added when no extension is given, deduplicated and sorted. A path, a wildcard or an
// image of the app under test is refused: a declaration names one foreign program, and the
// app's own tree is the subject of the audit, never tolerated noise.
func parseDeclaredLoad(s string, appImages ...string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	app := map[string]bool{}
	for _, image := range appImages {
		if image != "" && image != "." {
			app[strings.ToLower(image)] = true
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(s, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			return nil, fmt.Errorf("-declared-load has an empty entry in %q", s)
		}
		if strings.ContainsAny(name, `\/:*?"<>|`) {
			return nil, fmt.Errorf("-declared-load entry %q is not a bare executable file name", strings.TrimSpace(part))
		}
		if filepath.Ext(name) == "" {
			name += ".exe"
		}
		if app[name] {
			return nil, fmt.Errorf("-declared-load entry %q is the app under test; its load is the subject, not noise", name)
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}
