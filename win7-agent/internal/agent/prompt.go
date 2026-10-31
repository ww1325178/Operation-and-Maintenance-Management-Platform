package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// DefaultToken 内置共享令牌；编译时可用
// -ldflags "-X ops-platform/internal/agent.DefaultToken=XXX" 覆盖。
var DefaultToken = "填写一个你自己的token"

// BuildServerURL 把用户输入的 IP 或 IP:端口 构造成完整 ws:// 地址。
// 例如 "192.168.1.100" -> "ws://192.168.1.100:8080/ws/agent"
func BuildServerURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		if !strings.Contains(addr, "/ws/agent") {
			addr = strings.TrimRight(addr, "/") + "/ws/agent"
		}
		return addr
	}
	if !strings.Contains(addr, ":") {
		addr += ":8080"
	}
	return "ws://" + addr + "/ws/agent"
}

// PromptServerIP 首次运行（无配置）时弹出一个临时控制台让用户输入服务器
// IP 或 IP:端口，返回输入内容（取消/空则返回空串）。
func PromptServerIP() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("AllocConsole").Call()
	defer kernel32.NewProc("FreeConsole").Call()

	fmt.Print("Server IP or IP:port (default port 8080): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
