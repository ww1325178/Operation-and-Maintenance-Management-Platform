# 启动被控端 Agent（在 PowerShell 中运行，前台）
# 用法: .\run-agent.ps1 [服务端地址]
# 默认连接 ws://127.0.0.1:8080/ws/agent
$root = Split-Path -Parent $PSScriptRoot
$token = (Get-Content (Join-Path $root ".env") | Where-Object { $_ -like 'OPS_AGENT_TOKEN=*' }) -replace '^OPS_AGENT_TOKEN=',''

$server = if ($args.Count -gt 0) { $args[0] } else { "ws://127.0.0.1:8080/ws/agent" }

Write-Output "启动 Agent，连接 $server"
& (Join-Path $root "bin\agent.exe") -server $server -token $token
