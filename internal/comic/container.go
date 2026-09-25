package comic

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

// Container kinds, named as the user knows them.
const (
	containerZip = "ZIP"
	containerTar = "TAR"
	containerRar = "RAR"
	container7z  = "7z"
)

// extKind is what each extension promises. It is only a hint: the signature wins.
var extKind = map[string]string{
	".cbz": containerZip,
	".cbt": containerTar,
	".cbr": containerRar,
	".cb7": container7z,
}

// sniffContainer identifies an archive by its signature, or returns "" when none
// matches (an old pre-POSIX TAR has no magic, and an empty ZIP starts with its end
// record). The signatures match the extension's detectContainer (docs/PARITY.md).
func sniffContainer(head []byte) string {
	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		return containerZip
	case bytes.HasPrefix(head, []byte("Rar!\x1a\x07")):
		return containerRar
	case bytes.HasPrefix(head, []byte("7z\xbc\xaf\x27\x1c")):
		return container7z
	case len(head) >= 262 && string(head[257:262]) == "ustar":
		return containerTar
	}
	return ""
}

// containerKind reads the first bytes of path and returns its container, falling
// back to what the extension promises. A mismatch is said out loud, because a
// .cbz that needs 7-Zip is otherwise a surprise.
func containerKind(path, ext string) string {
	hint := extKind[ext]
	f, err := os.Open(path)
	if err != nil {
		return hint
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 262)
	n, _ := io.ReadFull(f, head)
	kind := sniffContainer(head[:n])
	if kind == "" {
		return hint
	}
	if kind != hint {
		logging.Printf("  %s\n", i18n.S("%s is a %s archive despite its extension, and is opened as one", filepath.Base(path), kind))
	}
	return kind
}
