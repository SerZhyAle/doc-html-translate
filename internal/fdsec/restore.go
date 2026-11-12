package fdsec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"

	"doc-html-translate/internal/procrun"
)

// SecretEnv is the variable that carries the password to FileDO. It is set in the FileDO
// child's environment only; FileDO removes it from its own environment the moment it reads it
// (FileDO's pe: credential source), so nothing it starts inherits it.
const SecretEnv = "DOCHT_FDSEC_SECRET"

// bannerBuild picks the build out of FileDO's first output line, which is how a "damaged" or
// "unsupported" answer is told apart from an outdated FileDO. Besides the closed line below,
// nothing else of FileDO's output is read: a successful restore prints the sealed true name there.
var bannerBuild = regexp.MustCompile(`sza@ukr\.net (\d{10})`)

// reasonDestinationName is the closed token of FDSEC-BEHAVIOUR 1.7 section 19: the one stderr line
// FileDO prints beside exit 5 when the destination cannot create the name sealed in a container
// that authenticated. It carries no name, path or credential. An older FileDO prints nothing.
const reasonDestinationName = "reason=destination-name"

// hasReasonLine reports whether stderr holds a line that equals want, byte for byte; only the
// line terminator (\n or \r\n) is stripped. This is the one thing read from FileDO's stderr, and
// no other byte of it leaves this function (FDSEC-BEHAVIOUR section 8.2).
func hasReasonLine(stderr []byte, want string) bool {
	for len(stderr) > 0 {
		line := stderr
		if i := bytes.IndexByte(stderr, '\n'); i >= 0 {
			line, stderr = stderr[:i], stderr[i+1:]
		} else {
			stderr = nil
		}
		if string(bytes.TrimSuffix(line, []byte{'\r'})) == want {
			return true
		}
	}
	return false
}

// restore runs FileDO's restore verb into dir with the password in the child's environment.
// --no-history is FileDO's global flag that keeps it from dropping a history.json (the
// container's path, the work folder) into whatever folder this process runs in.
// An empty password carries no secret, so it uses the visible empty form (FileDO refuses an
// empty variable). The returned error is built here and never wraps the helper's own error,
// whose text carries FileDO's stderr; FileDO's output is never copied anywhere, and the only
// thing taken from its stderr is whether the closed destination-name line is there (exit 5 only).
// secret is cleared before returning, best effort.
func restore(ctx context.Context, filedo, container, dir string, secret []byte) (string, error) {
	name := filepath.Base(container)
	cred := "p:"
	var env []string
	var envEntry []byte
	if len(secret) > 0 {
		cred = "pe:" + SecretEnv
		envEntry = append([]byte(SecretEnv+"="), secret...)
		env = []string{string(envEntry)}
	}
	res, err := procrun.Run(ctx, procrun.Cmd{
		Tool:      "FileDO",
		Path:      filedo,
		Args:      []string{"--no-history", container, "unsecure", "to", dir, cred, "-y"},
		Env:       env,
		Timeout:   procrun.FileDO.ForFile(container),
		MaxStdout: 64 << 10,
		MaxStderr: 4 << 10,
	})
	clear(secret)
	clear(envEntry)

	build := ""
	if m := bannerBuild.FindSubmatch(res.Stdout); m != nil {
		build = string(m[1])
	}
	if err == nil {
		return build, nil
	}
	if ctx.Err() != nil {
		return build, ctx.Err()
	}
	class := Failed
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		switch exit.ExitCode() {
		case 2:
			class = RefusedCall
		case 3:
			class = Credential
		case 4:
			class = Damaged
		case 5:
			class = IO
			if hasReasonLine(res.Stderr, reasonDestinationName) {
				class = DestinationName
			}
		case 6:
			class = Unsupported
		}
	}
	return build, &Error{Class: class, Name: name, Build: build}
}
