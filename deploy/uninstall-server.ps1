# 卸载服务端（需管理员权限）
Unregister-ScheduledTask -TaskName "OpsPlatformServer" -Confirm:$false -ErrorAction SilentlyContinue
Remove-NetFirewallRule -DisplayName "OpsPlatform" -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force "C:\OpsPlatform" -ErrorAction SilentlyContinue
Write-Output "服务端已卸载"
