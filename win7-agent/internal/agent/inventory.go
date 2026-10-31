package agent

import (
	"net"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"

	"ops-platform/internal/protocol"
)

// CollectInventory gathers hostname, OS version and network interfaces. This
// is the data behind requirement #1: IP / MAC / hostname / system version.
func CollectInventory() protocol.Hello {
	hostname, _ := os.Hostname()
	osName, osVersion, osBuild := readOSInfo()

	return protocol.Hello{
		Hostname:   hostname,
		OSName:     osName,
		OSVersion:  osVersion,
		OSBuild:    osBuild,
		Interfaces: listInterfaces(),
	}
}

func readOSInfo() (name, version, build string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows", "", ""
	}
	defer k.Close()

	name, _, _ = k.GetStringValue("ProductName")
	version, _, _ = k.GetStringValue("DisplayVersion")
	if version == "" {
		version, _, _ = k.GetStringValue("ReleaseId")
	}
	build, _, _ = k.GetStringValue("CurrentBuildNumber")
	return name, version, build
}

func listInterfaces() []protocol.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []protocol.Interface
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := ifc.HardwareAddr.String()
		if mac == "" {
			continue
		}
		addrs, _ := ifc.Addrs()
		var ips, ips6 []string
		for _, a := range addrs {
			ip := a.String()
			if i := strings.IndexByte(ip, '/'); i >= 0 {
				ip = ip[:i]
			}
			if ip == "" {
				continue
			}
			if strings.Contains(ip, ":") {
				ips6 = append(ips6, ip)
			} else {
				ips = append(ips, ip)
			}
		}
		// IPv4 first so the "primary IP" shown in the UI is the useful one.
		ips = append(ips, ips6...)
		out = append(out, protocol.Interface{Name: ifc.Name, MAC: mac, IPs: ips})
	}
	return out
}
