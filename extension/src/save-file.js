// save-file.js - hands the reader a file the extension just made (the settings file, the diagnostics
// summary). A Blob behind an anchor with the `download` attribute is the browser's own save, started by
// the reader's click: no `downloads` permission and no network access are involved.

const REVOKE_AFTER_MS = 60000;

export function saveTextFile(name, text, type) {
  const url = URL.createObjectURL(new Blob([text], { type }));
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), REVOKE_AFTER_MS);
}
