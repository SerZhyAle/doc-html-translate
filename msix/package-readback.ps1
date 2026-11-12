<#
.SYNOPSIS
  Read-back of a packed MSIX: open the archive and compare what it says with the expected values.

.DESCRIPTION
  WINDOWS-STORE 0.4 rules 1-4, conformance rung 3. build-msix.ps1 dot-sources this file right after
  "makeappx pack", and the negative run (temp\E0134) calls the function on copies of a package.
  The file only defines the function; it runs nothing on load.
#>

Add-Type -AssemblyName System.IO.Compression.FileSystem

function Assert-MsixReadback {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$IdentityName,
        [Parameter(Mandatory)][string]$Publisher,
        [Parameter(Mandatory)][string]$PublisherDisplayName,
        [Parameter(Mandatory)][string]$Version
    )
    $problems = [System.Collections.Generic.List[string]]::new()
    $zip = [System.IO.Compression.ZipFile]::OpenRead($Path)
    try {
        $manifestEntry = $zip.GetEntry('AppxManifest.xml')
        if (-not $manifestEntry) {
            $problems.Add('AppxManifest.xml is missing from the archive')
        } else {
            $reader = [System.IO.StreamReader]::new($manifestEntry.Open())
            try { [xml]$manifest = $reader.ReadToEnd() } finally { $reader.Dispose() }
            $identity = $manifest.Package.Identity
            if ($identity.Name -cne $IdentityName) {
                $problems.Add("Identity Name is '$($identity.Name)', expected '$IdentityName'")
            }
            if ($identity.Publisher -cne $Publisher) {
                $problems.Add("Identity Publisher is '$($identity.Publisher)', expected '$Publisher'")
            }
            if ($identity.Version -cne $Version) {
                $problems.Add("Identity Version is '$($identity.Version)', expected '$Version'")
            }
            $displayName = $manifest.Package.Properties.PublisherDisplayName
            if ($displayName -cne $PublisherDisplayName) {
                $problems.Add("PublisherDisplayName is '$displayName', expected '$PublisherDisplayName'")
            }
        }
        if ($zip.GetEntry('AppxSignature.p7x')) {
            $problems.Add('AppxSignature.p7x is present: the package is already signed before the upload step')
        }
        if (-not $zip.GetEntry('THIRD-PARTY-NOTICES.txt')) {
            $problems.Add('THIRD-PARTY-NOTICES.txt is missing from the archive')
        }
    } finally {
        $zip.Dispose()
    }
    if ($problems.Count -gt 0) {
        # Fail closed: a package that does not say what the build meant is not kept for upload.
        Remove-Item -LiteralPath $Path -Force
        throw "Read-back refused $Path (deleted):`n  $($problems -join "`n  ")"
    }
    Write-Host "Read-back OK: $(Split-Path -Leaf $Path) holds Identity $IdentityName $Version, no signature, notices present" -ForegroundColor DarkGray
}
