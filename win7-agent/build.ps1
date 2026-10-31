# 编译 Win7 兼容版 Agent（需 Go 1.20，见 README）
$go = "D:\dev\tools\go1.20\bin\go.exe"
$root = Split-Path -Parent $PSScriptRoot
$token = (Get-Content (Join-Path $root ".env") | Where-Object { $_ -like 'OPS_AGENT_TOKEN=*' }) -replace '^OPS_AGENT_TOKEN=',''
$ld = "-H=windowsgui -X ops-platform/internal/agent.DefaultToken=$token"

Push-Location $PSScriptRoot
try {
    $env:GOPROXY = "https://goproxy.cn,direct"

    & $go build -ldflags $ld -o (Join-Path $root "bin\agent-win7.exe") ./cmd/agent
    if ($LASTEXITCODE -ne 0) { throw "win7 amd64 编译失败" }

    $env:GOOS = "windows"; $env:GOARCH = "386"
    & $go build -ldflags $ld -o (Join-Path $root "bin\agent-win7-x86.exe") ./cmd/agent
    if ($LASTEXITCODE -ne 0) { throw "win7 386 编译失败" }
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
} finally {
    Pop-Location
}
Write-Output "Win7 Agent 编译完成：bin\agent-win7.exe（64位）、bin\agent-win7-x86.exe（32位）"
