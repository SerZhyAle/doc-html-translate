// save-file.js - hands the reader a file the extension holds (the settings file, the diagnostics
// summary, a media file read for "Save media file for translation"). A Blob behind an anchor with the
// `download` attribute is the browser's own save: no `downloads` permission is involved.

const REVOKE_AFTER_MS = 60000;

export function saveBlobFile(name, blob) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), REVOKE_AFTER_MS);
}

export function saveTextFile(name, text, type) {
  saveBlobFile(name, new Blob([text], { type }));
}
