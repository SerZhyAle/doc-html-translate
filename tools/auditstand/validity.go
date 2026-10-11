package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// "Sustained" is recorded as five consecutive one-second observations. A monitor gap
// over two seconds invalidates a run even if its cause cannot be distinguished from sleep.
const foreignCPUWindow = 5

type cpuCounter struct {
	Ticks  uint64
	Parent int
}

type validityWatch struct {
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	excluded map[int]bool
	reasons  []string
}

func startValidityWatch() *validityWatch {
	v := &validityWatch{stop: make(chan struct{}), done: make(chan struct{}), excluded: map[int]bool{os.Getpid(): true}}
	go func() {
		defer close(v.done)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		previous := map[int]cpuCounter{}
		streak := map[int]int{}
		last := time.Now()
		for {
			select {
			case <-v.stop:
				return
			case now := <-tick.C:
				if now.Sub(last) > 2*time.Second {
					v.add("validity monitor gap above 2 s (sleep or probe stall)")
				}
				if err := desktopUnlocked(); err != nil {
					v.add("desktop locked or secure desktop unavailable")
				}
				current, err := processCPUTimes()
				if err != nil {
					v.add("foreign CPU probe failed: " + err.Error())
					continue
				}
				seconds := now.Sub(last).Seconds()
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
					old, ok := previous[pid]
					if excluded || !ok || total.Ticks < old.Ticks || seconds <= 0 {
						streak[pid] = 0
						continue
					}
					fraction := float64(total.Ticks-old.Ticks) / 1e7 / seconds
					if fraction > 0.10 {
						streak[pid]++
					} else {
						streak[pid] = 0
					}
					if streak[pid] == foreignCPUWindow {
						v.add(fmt.Sprintf("foreign PID %d used over 10%% of one core for %d consecutive seconds", pid, foreignCPUWindow))
					}
				}
				previous = current
				last = now
			}
		}
	}()
	return v
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
