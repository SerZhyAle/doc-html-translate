//go:build !windows

package bundledtools

// PDFToTextPath reports ErrNotBundled: the embedded pdftotext is a Windows executable,
// so on other systems it is looked up on PATH instead.
func PDFToTextPath() (string, error) {
	return "", ErrNotBundled
}
