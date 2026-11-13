package audio

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
)

//go:embed nature/*.mp3
var audioFiles embed.FS

func Handler() http.Handler {
	subFS, err := fs.Sub(audioFiles, "nature")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(subFS))
}

func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("getting network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if !isInterfaceValid(iface) {
			continue
		}

		if ip, err := findPrivateIPv4(iface); err == nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("no private IPv4 address found")
}

func isInterfaceValid(iface net.Interface) bool {
	isUp := iface.Flags&net.FlagUp != 0
	isLoopback := iface.Flags&net.FlagLoopback != 0
	return isUp && !isLoopback
}

func findPrivateIPv4(iface net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP.To4()
		if ip != nil && ip.IsPrivate() {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no private IPv4 found on interface %s", iface.Name)
}
