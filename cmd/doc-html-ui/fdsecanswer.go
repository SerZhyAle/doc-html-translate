package main

import (
	"encoding/json"
	"net/http"
)

// handleSecret delivers the password the user typed in the window's masked dialog to the
// converter that asked for it (a FileDO secret file). The value goes from the request body to
// the child's stdin and nowhere else: it is not logged, not echoed, not kept - the local copy
// is dropped after the write (best effort: a Go string cannot be wiped). A cancel is the same call with no value.
//
//	POST {"input":"..","output":"..","value":"..","cancel":bool} → {"ok":bool}
func handleSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
		Value  string `json:"value"`
		Cancel bool   `json:"cancel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ok := answerRunSecret(runKey(req.Input, req.Output), req.Value, req.Cancel)
	req.Value = ""
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": ok})
}
