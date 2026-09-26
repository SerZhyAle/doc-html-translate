# goversioninfo, pinned per repo.
#
# The binary on PATH (%USERPROFILE%\go\bin) is shared with other repos on the machine, and at least one
# of them (FileDo) pins v1.4.1 and reinstalls it - which cannot read our three-ICO IconPath (mark, verb,
# document type) and fails opening the comma-joined path. So we never call the PATH copy: `go run
# module@version` builds the pinned version once into the Go build cache and ignores both PATH and our
# go.mod. Bump the pin here; every build script dot-sources this file.

$GoVersionInfoModule = "github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0"

# Runs the pinned goversioninfo with the given arguments in the current directory. The tool is built for
# the host, so a caller's cross-compile GOOS/GOARCH/CGO_ENABLED is cleared for the call and restored after.
# A plain function on purpose: a [Parameter()] attribute would make it an advanced function, whose
# common parameters (-OutVariable, -OutBuffer, ...) swallow goversioninfo's -o.
function Invoke-GoVersionInfo {
    $Arguments = $args
    $saved = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED }
    try {
        $env:GOOS = $null; $env:GOARCH = $null; $env:CGO_ENABLED = $null
        & go run $GoVersionInfoModule @Arguments
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo ($GoVersionInfoModule) failed in $(Get-Location)" }
    } finally {
        $env:GOOS = $saved.GOOS; $env:GOARCH = $saved.GOARCH; $env:CGO_ENABLED = $saved.CGO_ENABLED
    }
}
