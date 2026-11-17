//go:build !windows

package commands

import (
	"time"

	"doc-html-translate/internal/i18n"
)

// DefaultMutexName returns the single-instance lock identifier.
func DefaultMutexName() string { return "DocHtmlTranslateSingleInstance" }

// DefaultPipeName returns the local command pipe name.
func DefaultPipeName() string { return "" }

type noopLock struct{}

func (noopLock) Release() {}

// AcquireInstanceLock always acquires on non-Windows where the Windows pipe single-instance is inactive.
func AcquireInstanceLock(_ string) (InstanceLock, bool, error) { return noopLock{}, true, nil }

// PipeServer is a no-op stub on non-Windows.
type PipeServer struct{}

// StartPipeServer is a no-op on non-Windows.
func StartPipeServer(_ string, _ func([]byte) ([]byte, func()), _ string) (*PipeServer, error) {
	return &PipeServer{}, nil
}

// Close is a no-op on non-Windows.
func (*PipeServer) Close() {}

func dialAndExchange(_, _ string, req Request, _, _ time.Duration, _ bool, lang string) Response {
	return Response{
		SchemaVersion: SchemaVersion,
		RequestID:     req.RequestID,
		Status:        StatusRefused,
		ExitCode:      ExitNotFound,
		Reason:        ReasonNotRunning,
		Message:       i18n.T(lang, msgNotRunning),
	}
}
