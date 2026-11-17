package commands

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/internal/i18n"
)

// SchemaVersion is the wire schema version of APP-COMMANDS rule 7.
const SchemaVersion = 1

// MaxPipeLineBytes caps a single JSON Lines request or response over the command pipe (APP-COMMANDS rule 3, 0.2 item L).
const MaxPipeLineBytes = 64 << 10

// DeliveryBound is the connect and delivery deadline of APP-ACTIVATION rule 3 / APP-COMMANDS rule 8.
const DeliveryBound = 1000 * time.Millisecond

// Additional exit codes of APP-COMMANDS rule 11.
const (
	ExitNotProven  = 2
	ExitPermission = 12
)

// Closed reason tokens of APP-COMMANDS rule 12 and 0.2 item M.
const (
	ReasonBusy             = "busy"
	ReasonNotRunning       = "not-running"
	ReasonElevation        = "elevation"
	ReasonSchema           = "schema"
	ReasonTimeout          = "timeout"
	ReasonError            = "error"
	ReasonResponseTooLarge = "response-too-large"
)

// Response statuses of APP-COMMANDS rule 7.
const (
	StatusDone    = "done"
	StatusRefused = "refused"
	StatusFailed  = "failed"
	StatusBusy    = "busy"
)

// Exported i18n message keys shared with the launcher's InstanceHandler.
const (
	MsgLauncherBusy = msgLauncherBusy
	MsgQuitBusy     = msgQuitBusy
	MsgStatusReady  = msgStatusReady
	MsgShowDone     = msgShowDone
	MsgQuitDone     = msgQuitDone
	MsgOpenDone     = msgOpenDone
)

// Request is the JSON Lines request of APP-COMMANDS rule 7.
type Request struct {
	SchemaVersion int      `json:"schemaVersion"`
	Command       string   `json:"command"`
	Args          []string `json:"args"`
	WorkingDir    string   `json:"workingDir"`
	Origin        string   `json:"origin"`
	RequestID     string   `json:"requestId"`
	Wait          string   `json:"wait"`
	ResultPath    string   `json:"resultPath,omitempty"`
}

// Response is the JSON Lines response of APP-COMMANDS rule 7.
type Response struct {
	SchemaVersion int             `json:"schemaVersion"`
	RequestID     string          `json:"requestId"`
	Status        string          `json:"status"`
	ExitCode      int             `json:"exitCode"`
	Reason        string          `json:"reason,omitempty"`
	Message       string          `json:"message"`
	Data          json.RawMessage `json:"data,omitempty"`
}

// StatusData is the data payload of `status`.
type StatusData struct {
	Version                   string `json:"version"`
	PID                       int    `json:"pid"`
	WindowOpen                bool   `json:"windowOpen"`
	ActiveRuns                int    `json:"activeRuns"`
	AcceptsRestrictedCommands bool   `json:"acceptsRestrictedCommands"`
}

// ShowData is the data payload of `show`.
type ShowData struct {
	Focused  bool `json:"focused"`
	Reopened bool `json:"reopened"`
}

// QuitData is the data payload of `quit`.
type QuitData struct {
	CancelledRuns int `json:"cancelledRuns"`
}

// OpenData is the data payload of `open`.
type OpenData struct {
	Queued   int  `json:"queued"`
	Reopened bool `json:"reopened"`
}

// InstanceLock is an acquired single-instance mutex handle held by the first instance.
type InstanceLock interface {
	Release()
}

// LauncherHooks connects commands.Front to the launcher executable (doc-html-ui.exe).
type LauncherHooks struct {
	// BypassSingleInstance skips the mutex and pipe steps (DOCHT_GUI_NO_WINDOW).
	BypassSingleInstance bool
	// PipeName overrides the session pipe name (empty uses DefaultPipeName()).
	PipeName string
	// MutexName overrides the session mutex name (empty uses DefaultMutexName()).
	MutexName string
	// StartFirstInstance runs the launcher when this process becomes the first instance.
	// onReady, when non-nil, is called right after the HTTP server and pipe listener are up.
	StartFirstInstance func(lock InstanceLock, initialFiles []string, onReady func()) int
}

// InstanceHandler executes forwarded launcher verbs inside the running first instance.
type InstanceHandler interface {
	Status(ctx context.Context) (data StatusData, exitCode int, reason, msgKey string)
	Show(ctx context.Context) (data ShowData, exitCode int, reason, msgKey string)
	Quit(ctx context.Context, cancelRuns bool) (data QuitData, exitCode int, reason, msgKey string, afterSend func())
	Open(ctx context.Context, paths []string) (data OpenData, exitCode int, reason, msgKey string)
}

// HandlePipeRequest validates one request line from the pipe, executes at most one forwarded verb through h,
// and returns the JSON Lines response (capped at MaxPipeLineBytes) plus an optional post-reply hook (used by quit).
func HandlePipeRequest(ctx context.Context, m *Manifest, h InstanceHandler, lang string, line []byte) (reply []byte, afterSend func()) {
	if lang == "" {
		lang = i18n.Resolve("", "", "")
	}
	trimmed := bytes.TrimRight(line, "\r\n")
	if len(trimmed) > MaxPipeLineBytes {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     "",
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonTooLarge,
			Message:       i18n.T(lang, msgTooLarge),
		}), nil
	}
	if len(bytes.TrimSpace(trimmed)) == 0 {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     "",
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonBadArgument,
			Message:       i18n.T(lang, msgPipeFailed),
		}), nil
	}
	var req Request
	if err := json.Unmarshal(trimmed, &req); err != nil || strings.TrimSpace(req.RequestID) == "" {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     "",
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonBadArgument,
			Message:       i18n.T(lang, msgPipeFailed),
		}), nil
	}
	if req.SchemaVersion < 1 || req.SchemaVersion > SchemaVersion {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitNotProven,
			Reason:        ReasonSchema,
			Message:       i18n.T(lang, msgOlderInstance),
		}), nil
	}
	if req.Wait != "" && req.Wait != "done" {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonBadArgument,
			Message:       i18n.T(lang, msgUnexpected, req.Wait, "wait"),
		}), nil
	}
	cmd := m.Find(strings.Fields(req.Command)...)
	if cmd == nil || !launcherGroup[cmd.Name] {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonUnknownVerb,
			Message:       i18n.T(lang, msgUnknownCommand, req.Command),
		}), nil
	}
	if uerr := CheckBounds(req.Args); uerr != nil {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        uerr.Reason,
			Message:       uerr.Message(lang),
		}), nil
	}
	inv, uerr := Parse(cmd, req.Args)
	if uerr != nil {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        uerr.Reason,
			Message:       uerr.Message(lang),
		}), nil
	}
	// Validate --json and --result types if supplied in args.
	if _, uerr := inv.Bool("json"); uerr != nil {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        uerr.Reason,
			Message:       uerr.Message(lang),
		}), nil
	}
	if _, uerr := inv.String("result"); uerr != nil {
		return encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        uerr.Reason,
			Message:       uerr.Message(lang),
		}), nil
	}

	resp := Response{
		SchemaVersion: SchemaVersion,
		RequestID:     req.RequestID,
	}
	switch cmd.Name {
	case "status":
		d, code, reason, key := h.Status(ctx)
		resp.ExitCode = code
		resp.Reason = reason
		resp.Status = statusForExit(code, reason)
		resp.Message = i18n.T(lang, key)
		if code == ExitDone {
			resp.Data = encodeCompact(d)
		}
	case "show":
		d, code, reason, key := h.Show(ctx)
		resp.ExitCode = code
		resp.Reason = reason
		resp.Status = statusForExit(code, reason)
		resp.Message = i18n.T(lang, key)
		if code == ExitDone {
			resp.Data = encodeCompact(d)
		}
	case "quit":
		cancelRuns, uerr := inv.Bool("cancel-runs")
		if uerr != nil {
			return encodePipeResponse(lang, Response{
				SchemaVersion: SchemaVersion,
				RequestID:     req.RequestID,
				Status:        StatusRefused,
				ExitCode:      ExitUsage,
				Reason:        uerr.Reason,
				Message:       uerr.Message(lang),
			}), nil
		}
		d, code, reason, key, hook := h.Quit(ctx, cancelRuns)
		afterSend = hook
		resp.ExitCode = code
		resp.Reason = reason
		resp.Status = statusForExit(code, reason)
		resp.Message = i18n.T(lang, key)
		if code == ExitDone {
			resp.Data = encodeCompact(d)
		}
	case "open":
		paths, missing := resolveExistingPaths(req.WorkingDir, inv.Strings("in"))
		if missing {
			resp.ExitCode = ExitNotFound
			resp.Reason = ReasonTargetMissing
			resp.Status = StatusFailed
			resp.Message = i18n.T(lang, msgNoSuchDocument)
			break
		}
		d, code, reason, key := h.Open(ctx, paths)
		resp.ExitCode = code
		resp.Reason = reason
		resp.Status = statusForExit(code, reason)
		if code == ExitDone {
			resp.Message = i18n.T(lang, key, d.Queued)
			resp.Data = encodeCompact(d)
		} else {
			resp.Message = i18n.T(lang, key)
		}
	}
	return encodePipeResponse(lang, resp), afterSend
}

func statusForExit(code int, reason string) string {
	switch {
	case code == ExitDone:
		return StatusDone
	case reason == ReasonBusy:
		return StatusBusy
	case code == ExitUsage || code == ExitPermission || reason == ReasonNotRunning || reason == ReasonSchema:
		return StatusRefused
	default:
		return StatusFailed
	}
}

func encodePipeResponse(lang string, r Response) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
	line := b.Bytes()
	if len(bytes.TrimRight(line, "\r\n")) > MaxPipeLineBytes {
		b.Reset()
		_ = enc.Encode(Response{
			SchemaVersion: SchemaVersion,
			RequestID:     r.RequestID,
			Status:        StatusFailed,
			ExitCode:      ExitFailed,
			Reason:        ReasonResponseTooLarge,
			Message:       i18n.T(lang, msgPipeFailed),
		})
		line = b.Bytes()
	}
	return line
}

func encodeCompact(v any) json.RawMessage {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimSpace(b.Bytes())
}

// resolveExistingPaths resolves relative inputs against workDir and checks that every file exists on disk.
func resolveExistingPaths(workDir string, raw []string) (resolved []string, missing bool) {
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, true
		}
		full := p
		if !filepath.IsAbs(full) && workDir != "" {
			full = filepath.Join(workDir, full)
		}
		abs, err := filepath.Abs(full)
		if err != nil {
			return nil, true
		}
		info, err := os.Stat(abs)
		if err != nil || errors.Is(err, fs.ErrNotExist) || info.IsDir() {
			return nil, true
		}
		out = append(out, abs)
	}
	return out, false
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
