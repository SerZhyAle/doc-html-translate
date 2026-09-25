// Package bundledtools embeds and extracts external binaries bundled with the app.
// Only the Windows build carries any: the embedded pdftotext is a Windows executable.
package bundledtools

import "errors"

// ErrNotBundled means this build carries no copy of the requested tool, so the caller
// should look for one installed on the system.
var ErrNotBundled = errors.New("tool is not bundled with this build")
