# 启动运维平台服务端（在 PowerShell 中运行，前台）
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Get-Content (Join-Path $root ".env") | ForEach-Object {
  if ($_ -match '^([^#=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2], "Process")
  }
}

Write-Output "启动服务端（配置来自 .env，浏览器访问 http://localhost:8080）"
& (Join-Path $root "bin\server.exe")
