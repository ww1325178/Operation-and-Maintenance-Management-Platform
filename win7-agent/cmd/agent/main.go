package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ops-platform/internal/agent"
)

func main() {
	var (
		serverURL = flag.String("server", "", "服务端 WebSocket 地址")
		token     = flag.String("token", "", "Agent 令牌")
		interval  = flag.Duration("interval", 30*time.Second, "心跳间隔")
		deviceID  = flag.String("device", "", "设备 ID（默认自动生成并保存）")
		install   = flag.Bool("install", false, "部署安装：复制自身、写配置、注册开机自启、启动")
		uninstall = flag.Bool("uninstall", false, "卸载：移除自启与部署文件")
	)
	flag.Parse()

	// 日志写入文件（windowsgui 无控制台，文件是主要输出；文件优先避免控制台句柄无效）
	if base := os.Getenv("PROGRAMDATA"); base != "" {
		logDir := filepath.Join(base, "opsagent")
		_ = os.MkdirAll(logDir, 0o755)
		if f, err := os.OpenFile(filepath.Join(logDir, "agent.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			defer f.Close()
			log.SetOutput(io.MultiWriter(f, os.Stderr))
		}
	}

	// 安装 / 卸载
	if *install {
		if *serverURL == "" || *token == "" {
			agent.MessageBox("安装失败", "缺少 -server 或 -token 参数")
			os.Exit(1)
		}
		if err := agent.Install(*serverURL, *token); err != nil {
			log.Printf("安装失败: %v", err)
			agent.MessageBox("安装失败", err.Error())
			os.Exit(1)
		}
		log.Printf("安装成功: server=%s", *serverURL)
		agent.MessageBox("安装成功", "OpsAgent 已安装并启动")
		return
	}
	if *uninstall {
		agent.Uninstall()
		log.Printf("已卸载")
		agent.MessageBox("卸载完成", "OpsAgent 已卸载")
		return
	}

	// 未提供 server/token 时，从部署配置读取
	if *serverURL == "" || *token == "" {
		srv, tok := agent.LoadConfig()
		if *serverURL == "" {
			*serverURL = srv
		}
		if *token == "" {
			*token = tok
		}
	}

	if *token == "" || *serverURL == "" {
		// 首次运行：无配置，弹出提示输入服务器 IP，自动安装
		log.Println("未检测到配置，进入首次部署流程")
		ip := agent.PromptServerIP()
		if ip == "" {
			agent.MessageBox("已取消", "未输入服务器地址，部署已取消")
			os.Exit(1)
		}
		server := agent.BuildServerURL(ip)
		tok := *token
		if tok == "" {
			tok = agent.DefaultToken
		}
		if err := agent.Install(server, tok); err != nil {
			log.Printf("安装失败: %v", err)
			agent.MessageBox("安装失败", err.Error())
			os.Exit(1)
		}
		log.Printf("安装成功: server=%s", server)
		agent.MessageBox("安装成功", "OpsAgent 已安装并启动")
		return
	}

	id, err := agent.LoadOrCreateDeviceID(*deviceID)
	if err != nil {
		log.Fatalf("初始化设备 ID 失败: %v", err)
	}
	log.Printf("设备 ID: %s", id)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent.Run(ctx, agent.Config{
		ServerURL: *serverURL,
		Token:     *token,
		DeviceID:  id,
		Interval:  *interval,
	})
}
