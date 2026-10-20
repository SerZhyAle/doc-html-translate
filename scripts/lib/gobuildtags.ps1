# The Go build tags every go command of this repo needs (ticket 115).
#
# github.com/gen2brain/jpegxl, unless built with nodynamic, calls LoadLibrary("libjxl.dll") in its package
# init: whatever the DLL search path finds - the exe's folder, the current directory, PATH - is mapped into
# every start of the app, in place of the pinned WASM decoder. internal/jxl/nodynamic_guard.go refuses to
# compile without the tag, so a go command that misses it fails instead of shipping.
#
# The tags travel in GOFLAGS rather than as a -tags argument, so every go child of a script inherits them
# too (go run, golangci-lint, a test or tool that builds the app). Every script that runs go dot-sources
# this file; it merges into a GOFLAGS already set, keeping its other flags and tags. The merge runs in a
# child scope, so dot-sourcing leaves no variable behind but $GoBuildTags.

$GoBuildTags = @('nodynamic')

& {
    $tags = [System.Collections.Generic.List[string]]::new()
    $flags = [System.Collections.Generic.List[string]]::new()
    foreach ($f in ("$env:GOFLAGS" -split '\s+')) {
        if ($f -match '^--?tags=(.*)$') { foreach ($t in ($Matches[1] -split ',')) { if ($t -and -not $tags.Contains($t)) { $tags.Add($t) } } }
        elseif ($f) { $flags.Add($f) }
    }
    foreach ($t in $GoBuildTags) { if (-not $tags.Contains($t)) { $tags.Add($t) } }
    $flags.Add("-tags=$($tags -join ',')")
    $env:GOFLAGS = $flags -join ' '
}
