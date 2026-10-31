# 编译静默版 Agent（无控制台窗口，用于部署到被控机）
$ErrorActionPreference = "Stop"
$env:GOPROXY = "https://goproxy.cn,direct"
$root = Split-Path -Parent $PSScriptRoot
# 从 .env 读取令牌，内置进 exe（首次部署时只需填服务器 IP，不用填令牌）
$token = (Get-Content (Join-Path $root ".env") | Where-Object { $_ -like 'OPS_AGENT_TOKEN=*' }) -replace '^OPS_AGENT_TOKEN=',''
Push-Location $root
try {
    go build -ldflags "-H=windowsgui -X ops-platform/internal/agent.DefaultToken=$token" -o bin\agent.exe ./cmd/agent
} finally {
    Pop-Location
}
Write-Output "已生成静默版 bin\agent.exe（部署用，已内置令牌）"
