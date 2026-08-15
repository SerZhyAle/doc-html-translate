package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ProbeElementID is the element the probe writes its findings into. --dump-dom serializes it,
// which is how the runner reads a rendered page's real geometry without a WebSocket client or a
// browser-automation dependency.
const ProbeElementID = "ocrlab-evidence"

// probeTemplate is injected at the end of the converted page's body, after the app's own
// re-fit script, and observes rather than participates.
//
// Why a DOM probe and not the DevTools protocol: the repository already drives headless Chrome
// this way (scripts/verify-html.ps1), it needs no dependency and no open port, and everything
// the measurement wants - the laid-out rect of each plate, whether the browser says its content
// overflows, the colours it actually painted - is readable from the page itself. What it costs
// is one browser launch per state instead of one session for all of them, which at corpus scale
// is minutes rather than seconds and buys a runner with no moving parts.
//
// Two entry points, selected by the URL fragment:
//
//	#ocrlab-collect               apply every stress case in turn and emit all plate sets
//	#ocrlab-stress=<name>         apply one case and stop, so a screenshot captures that state
//
// %s is the JSON stress table.
const probeTemplate = `<script id="ocrlab-probe">
(function(){
  var CASES = %s;

  // Wait for the shipped re-fit to finish and for layout to settle.
  //
  // Timers only, deliberately. The obvious implementation waits on two requestAnimationFrame
  // callbacks, and under --virtual-time-budget that hangs forever: --dump-dom runs with no
  // compositor, so no frame is ever produced, and with nothing but a pending rAF the virtual
  // clock never advances - so the budget never expires and the browser never exits. Reading
  // layout forces a synchronous reflow, which is the part rAF was wanted for anyway.
  function settle(){
    return new Promise(function(r){
      setTimeout(function(){
        void document.body.offsetHeight; // force layout before anything is measured
        setTimeout(r, 30);
      }, 30);
    });
  }

  function figures(){
    var out = [];
    var imgs = document.querySelectorAll(".ocr-fig > img, .ocr-overlay-img");
    for (var i = 0; i < imgs.length; i++) {
      var img = imgs[i];
      var fig = img.closest(".ocr-fig") || img.parentElement;
      if (!fig) continue;
      out.push({ img: img, plates: fig.querySelectorAll(".ocr-box, .ocr-plate") });
    }
    return out;
  }

  function repeatTo(text, targetLen){
    if (!text) return "";
    var out = text;
    while (out.length < targetLen) out += text;
    return out.slice(0, Math.max(targetLen, text.length)).trim();
  }

  function applyCase(c){
    figures().forEach(function(f){
      for (var i = 0; i < f.plates.length; i++) {
        var p = f.plates[i];
        if (p.dataset.ocrlabOriginal === undefined) p.dataset.ocrlabOriginal = p.textContent;
        var original = p.dataset.ocrlabOriginal;
        if (!c.Text) { p.textContent = original; }
        else { p.textContent = repeatTo(c.Text, Math.max(1, Math.round(original.length * c.Factor))); }
        p.style.direction = c.Dir === "rtl" ? "rtl" : "";
      }
    });
  }

  // Map a plate's laid-out rect back into the source image's own pixels. Every number the lab
  // stores is in that space, so a run at three viewports produces directly comparable geometry.
  function readPlates(caseName){
    var out = [];
    figures().forEach(function(f){
      var ir = f.img.getBoundingClientRect();
      var nw = f.img.naturalWidth || ir.width, nh = f.img.naturalHeight || ir.height;
      if (!ir.width || !ir.height) return;
      var sx = nw / ir.width, sy = nh / ir.height;
      for (var i = 0; i < f.plates.length; i++) {
        var p = f.plates[i], r = p.getBoundingClientRect(), cs = getComputedStyle(p);
        out.push({
          text: p.textContent,
          rect: {
            x0: Math.round((r.left - ir.left) * sx),
            y0: Math.round((r.top - ir.top) * sy),
            x1: Math.round((r.right - ir.left) * sx),
            y1: Math.round((r.bottom - ir.top) * sy)
          },
          stressCase: caseName,
          fontPx: parseFloat(cs.fontSize) || 0,
          background: cs.backgroundColor,
          ink: cs.color,
          mode: p.dataset.ocrMode || "fill",
          modeConfidence: parseFloat(p.dataset.ocrModeConf || "0") || 0,
          scrollHeight: p.scrollHeight,
          clientHeight: p.clientHeight
        });
      }
    });
    return out;
  }

  // The scene's image, whether or not an overlay was drawn over it.
  //
  // figures() finds it through .ocr-fig, which only exists once the overlay has plates to hold.
  // A page where the recognizer found nothing has no .ocr-fig - and that is precisely the page
  // whose concealment measurement matters most, because nothing is concealed on it. Without a
  // rect the runner cannot map its screenshot into image space, stores no rendered shot, and the
  // scorer then reports residual 0 - "concealed perfectly" - for a page showing every original
  // word. So fall back to the largest image on the page, which on a lab scene is the scene.
  function sceneImage(){
    var f = figures()[0];
    if (f) return f.img;
    var imgs = document.querySelectorAll("img"), best = null, bestArea = 0;
    for (var i = 0; i < imgs.length; i++) {
      var r = imgs[i].getBoundingClientRect(), area = r.width * r.height;
      if (area > bestArea) { best = imgs[i]; bestArea = area; }
    }
    return best;
  }

  function imageRect(){
    var img = sceneImage();
    if (!img) return null;
    var r = img.getBoundingClientRect();
    if (!r.width || !r.height) return null;
    return {
      left: r.left, top: r.top, width: r.width, height: r.height,
      naturalWidth: img.naturalWidth, naturalHeight: img.naturalHeight
    };
  }

  function emit(payload){
    var el = document.createElement("script");
    el.type = "application/json";
    el.id = %q;
    el.textContent = JSON.stringify(payload);
    document.body.appendChild(el);
  }

  async function collect(){
    await settle();
    var plates = [], errors = [];
    for (var i = 0; i < CASES.length; i++) {
      var c = CASES[i];
      try { applyCase(c); await settle(); plates = plates.concat(readPlates(c.Name)); }
      catch (e) { errors.push(c.Name + ": " + e); }
    }
    applyCase(CASES[0]);
    await settle();
    emit({ ok: errors.length === 0, errors: errors, imageRect: imageRect(), plates: plates });
  }

  async function stressOnly(name){
    var c = CASES.filter(function(x){ return x.Name === name; })[0];
    await settle();
    if (c) { applyCase(c); await settle(); }
    emit({ ok: !!c, errors: c ? [] : ["unknown stress case " + name], imageRect: imageRect(), plates: [] });
  }

  function start(){
    var h = decodeURIComponent(location.hash || "");
    if (h.indexOf("#ocrlab-stress=") === 0) stressOnly(h.slice("#ocrlab-stress=".length));
    else collect();
  }
  if (document.readyState === "complete") start();
  else window.addEventListener("load", start);
})();
</script>`

// probeScript renders the probe with the current stress table baked in.
func probeScript() (string, error) {
	table, err := json.Marshal(StressCases)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(probeTemplate, table, ProbeElementID), nil
}

// probeResult is what the probe emits.
type probeResult struct {
	OK        bool         `json:"ok"`
	Errors    []string     `json:"errors"`
	ImageRect *probeRect   `json:"imageRect"`
	Plates    []probePlate `json:"plates"`
}

type probeRect struct {
	Left          float64 `json:"left"`
	Top           float64 `json:"top"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	NaturalWidth  int     `json:"naturalWidth"`
	NaturalHeight int     `json:"naturalHeight"`
}

type probePlate struct {
	Text string `json:"text"`
	Rect struct {
		X0 int `json:"x0"`
		Y0 int `json:"y0"`
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
	} `json:"rect"`
	StressCase     string  `json:"stressCase"`
	FontPx         float64 `json:"fontPx"`
	Background     string  `json:"background"`
	Ink            string  `json:"ink"`
	Mode           string  `json:"mode"`
	ModeConfidence float64 `json:"modeConfidence"`
	ScrollHeight   int     `json:"scrollHeight"`
	ClientHeight   int     `json:"clientHeight"`
}

// injectProbe writes a probe-carrying copy of a converted page next to the original, so the
// page's relative image sources still resolve. Returns the new file's path.
func injectProbe(pagePath string) (string, error) {
	data, err := os.ReadFile(pagePath)
	if err != nil {
		return "", err
	}
	script, err := probeScript()
	if err != nil {
		return "", err
	}
	html := string(data)
	idx := strings.LastIndex(strings.ToLower(html), "</body>")
	if idx < 0 {
		html += script
	} else {
		html = html[:idx] + script + html[idx:]
	}
	out := strings.TrimSuffix(pagePath, ".html") + ".ocrlab.html"
	if err := os.WriteFile(out, []byte(html), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// extractProbeResult pulls the probe's JSON out of a --dump-dom capture.
func extractProbeResult(dom string) (*probeResult, error) {
	marker := `id="` + ProbeElementID + `"`
	i := strings.Index(dom, marker)
	if i < 0 {
		return nil, fmt.Errorf("the page produced no %s element - the probe did not run", ProbeElementID)
	}
	open := strings.Index(dom[i:], ">")
	if open < 0 {
		return nil, fmt.Errorf("malformed %s element", ProbeElementID)
	}
	start := i + open + 1
	end := strings.Index(dom[start:], "</script>")
	if end < 0 {
		return nil, fmt.Errorf("unterminated %s element", ProbeElementID)
	}
	var res probeResult
	if err := json.Unmarshal([]byte(dom[start:start+end]), &res); err != nil {
		return nil, fmt.Errorf("parse probe output: %w", err)
	}
	return &res, nil
}
