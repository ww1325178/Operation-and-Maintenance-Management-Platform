package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	configName = "agent.conf"
	exeName    = "agent.exe"
)

// DeployDir 返回部署目录 %ProgramData%\opsagent
func DeployDir() string {
	base := os.Getenv("PROGRAMDATA")
	if base == "" {
		base = filepath.Join(os.Getenv("SystemDrive"), "ProgramData")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "opsagent")
}

// LoadConfig 从部署目录的 agent.conf 读取 server/token
func LoadConfig() (server, token string) {
	data, err := os.ReadFile(filepath.Join(DeployDir(), configName))
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.IndexByte(line, '='); i > 0 {
			k := strings.TrimSpace(line[:i])
			v := strings.TrimSpace(line[i+1:])
			switch k {
			case "server":
				server = v
			case "token":
				token = v
			}
		}
	}
	return server, token
}

// Install 部署：复制自身到部署目录、写配置、注册登录自启、启动
func Install(server, token string) error {
	dir := DeployDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, exeName)
	if !samePath(exe, dst) {
		if err := copyFile(exe, dst); err != nil {
			return err
		}
	}

	conf := fmt.Sprintf("server=%s\ntoken=%s\n", server, token)
	if err := os.WriteFile(filepath.Join(dir, configName), []byte(conf), 0o644); err != nil {
		return err
	}

	if err := setRunKey(dst); err != nil {
		return err
	}

	_ = exec.Command(dst).Start()
	return nil
}

// Uninstall 卸载：移除自启、配置与部署副本
func Uninstall() {
	removeRunKey()
	dir := DeployDir()
	_ = os.Remove(filepath.Join(dir, configName))
	_ = os.Remove(filepath.Join(dir, exeName))
}

func samePath(a, b string) bool {
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return strings.EqualFold(aa, bb)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func setRunKey(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("OpsAgent", `"`+exePath+`"`)
}

func removeRunKey() {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	_ = k.DeleteValue("OpsAgent")
}

// MessageBox 弹窗提示（windowsgui 无控制台，用弹窗反馈安装/卸载结果）
func MessageBox(title, text string) {
	u32 := syscall.NewLazyDLL("user32.dll")
	proc := u32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	ti, _ := syscall.UTF16PtrFromString(title)
	proc.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(ti)), 0x40) // MB_ICONINFORMATION
}
