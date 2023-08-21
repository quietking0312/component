package mnet

import (
	"net"
)

func GetLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	i := ""
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, address := range addrs {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				return ""
			}
			if ip.IsPrivate() {
				i = ip.String()
				return i // 返回的第一个网络适配器的内网ip 在特殊环境例如 wifi docker  第一个网络适配器或许并不是你想要的
			}
		}
	}

	return i
}
