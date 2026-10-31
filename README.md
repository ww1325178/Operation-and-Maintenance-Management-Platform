# 运维管理平台

内网 Windows 电脑的远程监控与管理平台（RMM）。被控端 Agent 采集信息并通过长连接上报，服务端存入 PostgreSQL 18，Web 管理端展示并支持远程查看/接管。

## 组件

| 目录 | 说明 |
|---|---|
| `cmd/server` | 服务端（Go）：HTTP API + Agent WebSocket + noVNC 中继 |
| `cmd/agent` | 被控端（Go）：采集 + 心跳上报，装为 Windows 服务 |
| `internal/protocol` | Agent ↔ 服务端的消息协议 |
| `internal/server` | 服务端实现（配置/DB/认证/路由） |
| `internal/agent` | 被控端实现（采集/连接） |
| `web/` | Web 管理端（React + Vite + Ant Design） |
| `deploy/` | 部署脚本与 Windows 服务封装（待补） |

## 快速开始

### 1. 服务端

```powershell
# 配置（环境变量，均可省略用默认值）
$env:OPS_DB_URL = "postgres://用户名:密码@localhost:5432/opsplatform?sslmode=disable"
$env:OPS_AGENT_TOKEN = "换成你的共享令牌"
$env:OPS_JWT_SECRET = "换成随机长字符串"
$env:OPS_ADMIN_PASSWORD = "初始管理员密码"

go run ./cmd/server
```

首次启动会自动建表并创建管理员（默认 `admin` / `OPS_ADMIN_PASSWORD`）。

### 2. 被控端 Agent

```powershell
go run ./cmd/agent -server ws://服务器:8080/ws/agent -token 共享令牌
```

编译成单个 exe 分发：

```powershell
go build -o bin/agent.exe ./cmd/agent
go build -o bin/server.exe ./cmd/server
```

### 3. Web 管理端

```powershell
cd web
npm install
npm run dev     # 开发模式，代理 /api 到 :8080
npm run build   # 产物输出到 web/dist，由服务端直接托管
```

## 被控端系统支持

| 系统 | 支持 |
|---|---|
| Windows 10 / 11（64 位） | ✅ `bin/agent.exe` |
| Windows 10 / 11（32 位） | ✅ `bin/agent-x86.exe` |
| Windows 7 / 8 | ✅ `bin/agent-win7.exe`（64位）/ `bin/agent-win7-x86.exe`（32位），Go 1.20 编译 |
| Windows XP / Vista | ❌ 需 Go 1.10（2018），现代依赖无法兼容 |

远程画质：高清(原分辨率) / 标清(1/2) / 流畅(1/4)，在远程窗口内切换，鼠标坐标自动换算回真实屏幕。

> 📄 完整部署文档（编译 → 数据库 → 服务端 → 被控端 → 验证 → 排障）见 [docs/deploy.md](docs/deploy.md)。

## 服务端部署（一键）

```powershell
# 需管理员权限；PostgreSQL 已安装并建好库/账号
.\deploy\install-server.ps1 -DbUrl "postgres://ops:密码@127.0.0.1:5432/opsplatform?sslmode=disable" -AgentToken "你的令牌"
```

- 部署到 `C:\OpsPlatform\`，注册计划任务 `OpsPlatformServer`（开机自启、崩溃自动重启）。
- 自动放行防火墙 TCP 端口（默认 8080）。
- 配置写在 `C:\OpsPlatform\server.conf`；服务端配置优先级：默认值 < server.conf < 环境变量。
- 卸载：`.\deploy\uninstall-server.ps1`。

## 被控端部署（静默安装）

Agent 支持自部署：`-install` 会复制自身到 `%ProgramData%\opsagent\`、写配置、注册登录自启（HKCU Run 键）并启动。

```powershell
# 1. 编译静默版（无窗口）
.\deploy\build-agent.ps1          # Win10 版 → bin\agent.exe
# Win7 版见 win7-agent\build.ps1 → bin\agent-win7.exe / agent-win7-x86.exe

# 2. 部署（在被控机上执行一次）
agent.exe -install -server ws://服务器:8080/ws/agent -token 令牌
```

- 双击 `deploy\install.bat`（改好 server/token）即可静默安装，弹「安装成功」提示。
- 卸载：`agent.exe -uninstall`。
- 配置存于 `%ProgramData%\opsagent\agent.conf`，日志存于 `%ProgramData%\opsagent\agent.log`。
- 自启用「用户登录自启」（非 SYSTEM 服务），因为屏幕采集/远程控制需运行在登录用户会话。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `OPS_ADDR` | `:8080` | 服务端监听地址 |
| `OPS_DB_URL` | `postgres://postgres:postgres@localhost:5432/opsplatform?sslmode=disable` | PostgreSQL 连接串 |
| `OPS_JWT_SECRET` | dev 值 | JWT 签名密钥，生产必须改 |
| `OPS_AGENT_TOKEN` | 空（拒绝连接） | Agent 共享令牌，生产必须设 |
| `OPS_ADMIN_USER` / `OPS_ADMIN_PASSWORD` | `你的user` / `你的密码` | 初始管理员 |
| `OPS_HEARTBEAT_TIMEOUT_SEC` | `90` | 心跳超时判离线（秒） |


## 已知限制（v1）

- VNC 引擎为自研 Go 实现：raw 编码 + 脏矩形增量，仅主显示器，键鼠映射覆盖基础键位
- 无剪贴板同步；VNC 握手无密码（靠回环端口 + 随机会话 token + 管理员 JWT + 审计兜底）
- 屏幕采集用 GDI BitBlt，抓不到 UAC 安全桌面 / 硬件加速受保护内容
- 隧道与中继层已与引擎解耦，后续可按需换成 UltraVNC/TightVNC 引擎
- 仅作为科研使用，严禁非法用途
- 因涉及到网络安全，固未对agent做任何套壳、防杀等
