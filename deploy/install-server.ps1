# 服务端一键部署脚本（需管理员权限运行）
# 用法示例：
#   .\install-server.ps1 -DbUrl "postgres://ops:密码@127.0.0.1:5432/opsplatform?sslmode=disable" -AgentToken "你的令牌"
param(
    [Parameter(Mandatory = $true)][string]$DbUrl,
    [Parameter(Mandatory = $true)][string]$AgentToken,
    [string]$Port = "8080",
    [string]$AdminPassword = "admin123"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$deployDir = "C:\OpsPlatform"

Write-Output "== 1/4 复制文件 =="
New-Item -ItemType Directory -Force -Path $deployDir | Out-Null
Copy-Item "$root\bin\server.exe" "$deployDir\server.exe" -Force
New-Item -ItemType Directory -Force -Path "$deployDir\web" | Out-Null
Copy-Item "$root\web\dist" "$deployDir\web\dist" -Recurse -Force

Write-Output "== 2/4 生成配置 =="
$jwtSecret = -join ((48..57)+(65..90)+(97..122) | Get-Random -Count 40 | ForEach-Object {[char]$_})
$conf = "OPS_ADDR=:$Port`nOPS_DB_URL=$DbUrl`nOPS_AGENT_TOKEN=$AgentToken`nOPS_JWT_SECRET=$jwtSecret`nOPS_ADMIN_PASSWORD=$AdminPassword`n"
[System.IO.File]::WriteAllText("$deployDir\server.conf", $conf, (New-Object System.Text.UTF8Encoding($false)))

Write-Output "== 3/4 注册开机自启（计划任务 SYSTEM）=="
$action = New-ScheduledTaskAction -Execute "$deployDir\server.exe" -WorkingDirectory $deployDir
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "OpsPlatformServer" -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null

Write-Output "== 4/4 放行防火墙 + 启动 =="
New-NetFirewallRule -DisplayName "OpsPlatform" -Direction Inbound -Protocol TCP -LocalPort $Port -Action Allow -ErrorAction SilentlyContinue | Out-Null
Start-ScheduledTask -TaskName "OpsPlatformServer"

Write-Output ""
Write-Output "部署完成：$deployDir"
Write-Output "访问地址：http://服务器IP:$Port（管理员 admin / $AdminPassword）"
