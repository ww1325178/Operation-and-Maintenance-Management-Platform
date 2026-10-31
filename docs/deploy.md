# 情指行一体化平台 · 部署文档

> 适用范围：内网 Windows 环境。服务端一套，被控端 50–200 台。

## 1. 架构概览

- **服务端**（Go）：REST API + Agent 长连接 + noVNC 中继 + 前端静态托管，需连接 PostgreSQL 18。
- **被控端 Agent**（Go）：装在每台被管电脑上，**主动出站**连接服务端（客户端无需开放入站端口）。
- **管理端**：浏览器访问服务端，侧边栏「情报 / 指挥 / 行动」，「情报」下是「一机两用及上网行为管控」。

## 2. 环境要求

| 角色 | 要求 |
|---|---|
| 服务端 | Windows Server（或 Win10+），能访问 PostgreSQL，内网可达 |
| 数据库 | PostgreSQL 18 |
| 被控端 | Windows 10/11（64/32 位）；Windows 7/8 用 Go 1.20 版 Agent |
| 网络 | 被控端 → 服务端 TCP 8080 端口互通 |

## 3. 编译产物

首次从源码编译（已编译好可跳过）：

```powershell
# 1. 前端
cd web; npm install; npm run build; cd ..

# 2. 服务端
go build -o bin\server.exe ./cmd/server

# 3. 被控端（Win10，静默无窗口）
.\deploy\build-agent.ps1

# 4. 被控端（Win7，静默无窗口）
.\win7-agent\build.ps1
```

产物清单：

| 文件 | 说明 |
|---|---|
| `bin\server.exe` + `web\dist\` | 服务端（两个要一起部署） |
| `bin\agent.exe` | 被控端 Win10/11 64 位 |
| `bin\agent-x86.exe` | 被控端 Win10/11 32 位 |
| `bin\agent-win7.exe` | 被控端 Win7/8 64 位 |
| `bin\agent-win7-x86.exe` | 被控端 Win7/8 32 位 |

## 4. 数据库准备

安装 PostgreSQL 18，创建专用账号和数据库：

```sql
CREATE ROLE ops WITH LOGIN PASSWORD '改成你的密码';
CREATE DATABASE opsplatform OWNER ops;
```

服务端首次启动会自动建表、自动创建管理员（`admin`）。

## 5. 服务端部署（一键）

> 前置：PostgreSQL 已就绪；`web\dist` 已构建。

在服务器上**以管理员**运行 PowerShell：

```powershell
.\deploy\install-server.ps1 `
  -DbUrl "postgres://ops:密码@127.0.0.1:5432/opsplatform?sslmode=disable" `
  -AgentToken "你的被控端令牌"
```

参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| `-DbUrl` | ✅ | PostgreSQL 连接串 |
| `-AgentToken` | ✅ | 被控端共享令牌（**自定义一个，别用默认**） |
| `-Port` | 否 | 服务端口，默认 `8080` |
| `-AdminPassword` | 否 | 管理员初始密码，默认 `admin123` |

脚本自动完成：
1. 复制 `server.exe` + 前端到 `C:\OpsPlatform\`；
2. 生成配置 `C:\OpsPlatform\server.conf`；
3. 注册计划任务 `OpsPlatformServer`（开机自启、崩溃自动重启）；
4. 放行防火墙 TCP 端口；
5. 启动服务端。

完成后浏览器访问 `http://服务器IP:8080`，账号 `admin`。

**卸载**：

```powershell
.\deploy\uninstall-server.ps1
```

## 6. 被控端部署

### 方式 A：单文件部署（推荐）

1. 把对应位数的 Agent（如 `agent.exe`）**这一个文件**拷到被控机；
2. 右键 → **以管理员身份运行**；
3. 弹出的命令行里**输入服务器 IP**（如 `192.168.1.100`，默认端口 8080；也可写 `IP:端口`）；
4. 自动安装、写配置、注册开机自启并启动，弹「安装成功」。

> 令牌已内置进 exe（编译时从 .env 读取），所以只需填 IP，不用填令牌。

### 方式 B：命令行（备选）

```powershell
agent.exe -install -server ws://服务器IP:8080/ws/agent -token 令牌
```

### 安装后

- 部署目录：`%ProgramData%\opsagent\`（`agent.conf` 配置、`agent.log` 日志）。
- 开机自启：**用户登录时自动启动**（注册表 Run 键，保证能采集屏幕）。
- 卸载：`agent.exe -uninstall`。

> Win7 机器用 `agent-win7.exe`（64 位）或 `agent-win7-x86.exe`（32 位），部署方式相同。

## 7. 验证

1. 浏览器登录管理端（`admin`）；
2. 进入「情报 → 一机两用及上网行为管控」，确认被控机显示**在线**；
3. 点「远程」，确认画面实时、键鼠可控制、鼠标指针可见。

## 8. 修改服务器地址

- **被控端**：改 `%ProgramData%\opsagent\agent.conf` 里的 `server=`，重启 agent（或重新 `-install`）。
- **服务端**：改 `C:\OpsPlatform\server.conf`，重启计划任务。

## 9. 常见问题

| 现象 | 排查 |
|---|---|
| 被控机不上线 | ① 服务端防火墙是否放行 8080；② 被控端能否 ping 通服务器；③ 看 `agent.log` 报错 |
| 远程画面灰屏/定帧 | 检查被控端版本是否匹配（Win7 必须用 win7 版）；看 `agent.log` |
| 杀软误杀/拦截 | 见第 10 节 |
| 忘了管理员密码 | 修改库里的 users 表；或改 `OPS_ADMIN_PASSWORD` 后重启（仅首次建用户时生效） |

## 10. 安全与合规建议

1. **代码签名**：给 Agent 签单位的代码签名证书，可避免被火绒/360 等误杀；
2. **加白名单**：在杀软管理端把 `%ProgramData%\opsagent\` 和 `C:\OpsPlatform\` 加入信任；
3. **合规告知**：在员工手册/入职须知注明「配发电脑装有运维 Agent」；
4. **改默认口令**：部署后立即修改 admin 密码、Agent 令牌、JWT 密钥。
