// browsercheck.go is step 00.08's own verification (phase 0): pre-launch the instrumented
// browser, observe the marker page through the debugging port, time an injected key when the
// owner opted in, cross-check the capture-diff fallback on the same window, then kill the
// instance and verify the port is gone. It is the step's exit gate for the GUI scenes: until
// it passes, the DOM-level cells cannot close and the fallback is the only timing.

package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type browserCheckResult struct {
	verdict verdict
	detail  string
}

func runBrowserCheck(env *runEnv, allowKeys bool) (browserCheckResult, error) {
	exe, err := pickAppBrowser()
	if err != nil {
		return browserCheckResult{}, err
	}
	marker, err := startMarkerServer()
	if err != nil {
		return browserCheckResult{}, err
	}
	defer marker.close()
	port, err := freeLoopbackPort()
	if err != nil {
		return browserCheckResult{}, err
	}
	session, err := prelaunchBrowser(exe, port, marker.url, marker.title)
	if err != nil {
		_ = session.close()
		if errors.Is(err, errBrowserPortUnavailable) {
			return browserCheckResult{}, fmt.Errorf("%w; close the running browser of this profile before instrumented runs, or the GUI scenes' DOM-level cells stay n/a and the capture-diff fallback carries the timing", err)
		}
		return browserCheckResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	target, err := session.markerTarget(ctx)
	if err != nil {
		_ = session.close()
		return browserCheckResult{}, err
	}
	probe, err := connectBrowserProbe(ctx, fmt.Sprintf("http://127.0.0.1:%d", port), target.ID)
	if err != nil {
		_ = session.close()
		return browserCheckResult{}, err
	}
	defer probe.conn.Close()

	fields, err := probe.sampleDOM(ctx)
	if err != nil {
		_ = session.close()
		return browserCheckResult{}, fmt.Errorf("the marker page could not be sampled through the port: %w", err)
	}
	fmt.Printf("  marker page observed through the port: dom_nodes=%v js_heap_bytes=%v\n", fields["dom_nodes"], fields["js_heap_bytes"])

	advisories := []string{}
	keyMs, paintMs := -1.0, -1.0
	if !allowKeys {
		advisories = append(advisories, "key-to-reaction timing not measured: -allow-key-inject is required (the injected-key probe never types uninvited)")
	} else {
		wins, err := findWindowsByTitle(marker.title)
		if err != nil {
			_ = session.close()
			return browserCheckResult{}, err
		}
		if len(wins) == 0 {
			advisories = append(advisories, "the marker window was not found by title; the key reaction was not measured")
		} else if err := bringWindowToFront(wins[0]); err != nil {
			advisories = append(advisories, err.Error())
		} else {
			keyMs, err = probe.keyReaction(ctx, wins[0])
			if err != nil {
				_ = session.close()
				return browserCheckResult{}, fmt.Errorf("the injected key was not observed through the port: %w", err)
			}
			fmt.Printf("  P3 injected key to visible reaction (through the port): %.1f ms\n", keyMs)
			// The same window now cross-validates the capture-diff fallback P3 uses when
			// no port can be opened: one more injected key, timed by pixel change.
			if _, gridErr := captureWindowGrid(wins[0]); gridErr == nil {
				go func() { _, _ = probe.keyReaction(ctx, wins[0]) }()
				var changed bool
				paintMs, changed, err = waitWindowPaint(wins[0], 3*time.Second)
				if err != nil {
					advisories = append(advisories, "capture-diff fallback failed: "+err.Error())
				} else if !changed {
					advisories = append(advisories, "capture-diff fallback saw no visible change; the fallback timing is unproven")
				} else {
					fmt.Printf("  P3 injected key to visible change (capture-diff fallback): %.1f ms\n", paintMs)
				}
			} else {
				advisories = append(advisories, "capture-diff fallback unavailable on this window: "+err.Error())
			}
		}
	}

	if err := session.close(); err != nil {
		return browserCheckResult{verdict: verdictFail, detail: err.Error()}, nil
	}
	fmt.Printf("  instrumented instance closed; debugging port %d verified closed\n", port)

	if len(advisories) > 0 {
		detail := fmt.Sprintf("instrumented browser verified; %s", advisories[0])
		return browserCheckResult{verdict: verdictAdvisory, detail: detail}, nil
	}
	return browserCheckResult{verdict: verdictPass, detail: fmt.Sprintf("marker observed, injected key %.1f ms, capture-diff %.1f ms, port closed", keyMs, paintMs)}, nil
}
