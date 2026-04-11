$ErrorActionPreference = "Stop"

Write-Host "Installing golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
if ($LASTEXITCODE -ne 0) {
    throw "failed to install golangci-lint"
}

Write-Host "Installing typos-cli..."
cargo install typos-cli
if ($LASTEXITCODE -ne 0) {
    throw "failed to install typos-cli via cargo. Install Rust/Cargo first or install typos manually"
}

Write-Host "Tool bootstrap completed"
