package utils

import (
	"github.com/praserx/ipconv"
)

func IPv4ToInt(ip string) uint32 {
	if ip, version, err := ipconv.ParseIP(ip); err == nil && version == 4 {
		data, _ := ipconv.IPv4ToInt(ip)
		return data
	}
	return 0
}
