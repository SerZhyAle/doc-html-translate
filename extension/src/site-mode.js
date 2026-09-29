// The site mode and membership decision shared by the popup, shortcut and badge.
export function allowlistMode(options) {
  return options.siteMode === "allowlist";
}

// DNR domain conditions include subdomains, so the controls mirror that meaning.
export function hostInList(hosts, host) {
  return (hosts || []).some((entry) => host === entry || host.endsWith(`.${entry}`));
}

export function siteEnabled(options, host) {
  if (!host || !options.enabledByDefault) return false;
  return allowlistMode(options)
    ? hostInList(options.allowedHosts, host)
    : !hostInList(options.disabledHosts, host);
}

export function setSiteEnabled(options, host, enabled) {
  const key = allowlistMode(options) ? "allowedHosts" : "disabledHosts";
  const set = new Set(options[key] || []);
  if (enabled === allowlistMode(options)) set.add(host);
  else for (const entry of set) if (host === entry || host.endsWith(`.${entry}`)) set.delete(entry);
  options[key] = [...set];
}
