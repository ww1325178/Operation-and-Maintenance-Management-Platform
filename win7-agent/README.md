# Win7 兼容版 Agent

用 **Go 1.20** 编译的 Agent，兼容 Windows 7 / 8 / Server 2008 R2 / 2012。

## 为什么独立目录

主项目用 Go 1.27（运行时要求 Windows 10+），而 Win7 最高只支持 Go 1.20。因此这里复制了 Agent 相关源码（`protocol` / `agent` / `vnc`），用独立的 `go.mod` 和独立的 Go 1.20 工具链（`D:\dev\tools\go1.20`）编译。

## 构建

```powershell
.\build.ps1
```

产出：
- `bin\agent-win7.exe`（64 位）
- `bin\agent-win7-x86.exe`（32 位）

## 维护注意

这里的代码是主项目 `internal/{protocol,agent,vnc}` 的**快照**。若主项目这些文件有改动，需同步复制过来并重新编译。

## 已知限制

- Go 1.20 要求 Windows 7 **SP1 + 平台更新（KB2670838）**，否则个别图形 API 可能不可用。
- Windows XP / Vista 不受支持（Go 1.10 之后已移除，现代依赖无法兼容）。
