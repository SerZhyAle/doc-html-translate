package pipeline

import (
	"context"
	"errors"
	"path/filepath"

	"doc-html-translate/internal/fdsec"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

// unwrapContainer is the pre-stage for a FileDO secret file (.fd-sec): the installed FileDO
// decrypts it into one private plain copy, and the extractor then reads that copy while the
// container stays the run's identity (output folder, reuse, reader key, navigation label,
// completion record). The caller removes the copy when extraction ends.
//
// Failures reuse the published exit codes - no new one (ticket 94, owner decision Q3): the
// distinction between the classes is in the words, which fdsec.Error localises. The error is
// returned as it is so the CLI prints that text; nothing here adds FileDO's output, the
// password or the sealed name to it.
func (r Runner) unwrapContainer(ctx context.Context, inputPath string) (*fdsec.Plain, int, error) {
	logging.Println(i18n.S("  Opening FileDO secret file %s with FileDO..", filepath.Base(inputPath)))
	plain, err := fdsec.Unwrap(ctx, inputPath, fdsec.NewAsker(r.cfg.FdsecPasswordEnv), Convertible)
	if err == nil {
		return plain, ExitOK, nil
	}
	var fe *fdsec.Error
	switch {
	case ctx.Err() != nil:
		return nil, ExitInterrupted, err
	case !errors.As(err, &fe):
		return nil, ExitParse, err
	}
	switch fe.Class {
	case fdsec.Cancelled:
		return nil, ExitInterrupted, err
	case fdsec.NoSource, fdsec.EmptyEnv:
		return nil, ExitArgsError, err
	case fdsec.IO:
		return nil, ExitIOError, err
	}
	return nil, ExitParse, err
}
