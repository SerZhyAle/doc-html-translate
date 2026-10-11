package main

import (
	"strings"
	"testing"
	"time"
)

// feed drives observe with a synthetic CPU history: each step adds cpuTicks of busy time to the
// given process over one second of wall clock.
type cpuHistory struct {
	v     *validityWatch
	now   time.Time
	ticks map[int]uint64
}

func newCPUHistory() *cpuHistory {
	t0 := time.Unix(1_700_000_000, 0)
	return &cpuHistory{v: newValidityWatch(t0), now: t0, ticks: map[int]uint64{}}
}

// step advances one second; busy maps pid -> fraction of one core used during that second.
func (h *cpuHistory) step(procs map[int]cpuCounter, busy map[int]float64) {
	h.now = h.now.Add(time.Second)
	current := map[int]cpuCounter{}
	for pid, p := range procs {
		h.ticks[pid] += uint64(busy[pid] * 1e7)
		p.Ticks = h.ticks[pid]
		current[pid] = p
	}
	h.v.observe(h.now, current)
}

func TestSustainedForeignLoadIsNamedByImage(t *testing.T) {
	h := newCPUHistory()
	procs := map[int]cpuCounter{
		100: {Image: "Code.exe", Parent: 1},
		101: {Image: "Code.exe", Parent: 1},
		102: {Image: "chrome.exe", Parent: 1},
		103: {Image: "idle.exe", Parent: 1},
	}
	busy := map[int]float64{100: 0.5, 101: 0.2, 102: 0.15, 103: 0.01}
	h.step(procs, nil) // baseline snapshot, no earlier counters to diff
	for i := 0; i < foreignCPUWindow-1; i++ {
		h.step(procs, busy)
	}
	if len(h.v.reasons) != 0 {
		t.Fatalf("a window not yet complete produced reasons: %v", h.v.reasons)
	}
	h.step(procs, busy)
	if len(h.v.reasons) != 3 {
		t.Fatalf("reasons = %v, want the three busy processes", h.v.reasons)
	}
	for _, r := range h.v.reasons {
		if strings.Contains(r, "idle.exe") {
			t.Errorf("an idle process was named: %s", r)
		}
	}
	sum := h.v.offenderSummary()
	if !strings.Contains(sum, "Code.exe x2") || !strings.Contains(sum, "chrome.exe x1") {
		t.Errorf("summary = %q", sum)
	}
	if strings.Index(sum, "Code.exe") > strings.Index(sum, "chrome.exe") {
		t.Errorf("summary must list the most numerous image first: %q", sum)
	}
}

func TestOneSecondSpikeResetsTheStreak(t *testing.T) {
	h := newCPUHistory()
	procs := map[int]cpuCounter{100: {Image: "spiky.exe", Parent: 1}}
	h.step(procs, nil)
	pattern := []float64{0.5, 0.5, 0.5, 0.5, 0.0, 0.5, 0.5, 0.5, 0.5, 0.0}
	for _, f := range pattern {
		h.step(procs, map[int]float64{100: f})
	}
	if len(h.v.reasons) != 0 {
		t.Fatalf("a process that dipped inside the window must not be named: %v", h.v.reasons)
	}
}

func TestExcludedTreeIsNotForeign(t *testing.T) {
	h := newCPUHistory()
	h.v.exclude(200) // the app under test
	procs := map[int]cpuCounter{
		200: {Image: "doc-html-translate.exe", Parent: 1},
		201: {Image: "7z.exe", Parent: 200},        // its helper
		202: {Image: "tesseract.exe", Parent: 201}, // the helper's helper
	}
	busy := map[int]float64{200: 0.9, 201: 0.9, 202: 0.9}
	h.step(procs, nil)
	for i := 0; i < foreignCPUWindow+2; i++ {
		h.step(procs, busy)
	}
	if len(h.v.reasons) != 0 {
		t.Fatalf("the app's own tree was counted as foreign load: %v", h.v.reasons)
	}
}

func TestVanishedProcessLosesItsStreak(t *testing.T) {
	h := newCPUHistory()
	procs := map[int]cpuCounter{100: {Image: "a.exe", Parent: 1}}
	h.step(procs, nil)
	for i := 0; i < foreignCPUWindow-1; i++ {
		h.step(procs, map[int]float64{100: 0.5})
	}
	h.step(map[int]cpuCounter{}, nil) // the process exits
	if _, ok := h.v.streak[100]; ok {
		t.Fatalf("a vanished process kept its streak: %v", h.v.streak)
	}
}

func TestImageIsAFileNameNotAPath(t *testing.T) {
	// The reader supplies only ExeFile from the toolhelp snapshot; this pins the contract that a
	// recorded image never carries a directory.
	h := newCPUHistory()
	procs := map[int]cpuCounter{100: {Image: "weird.exe", Parent: 1}}
	h.step(procs, nil)
	for i := 0; i < foreignCPUWindow; i++ {
		h.step(procs, map[int]float64{100: 1})
	}
	for _, r := range h.v.reasons {
		if strings.ContainsAny(r, `\/`) {
			t.Errorf("a reason carries a path separator: %s", r)
		}
	}
}
