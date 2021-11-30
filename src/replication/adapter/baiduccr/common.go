package baiduccr

import (
	"os"
	"strings"
)

const (
	endpoint       = "ccr.gz.baidubce.com"
	credentialPath = "/var/run/secrets/ccr/credential"
)

func getEndpoint(region string) string {
	ep := os.Getenv("CCR_ENDPOINT")
	if ep == "" {
		return "ccr." + region + ".baidubce.com"
	}
	return ep
}

func getClusterIDFromHost(host string) string {
	if strings.HasPrefix(host, "http://") {
		host = strings.TrimPrefix(host, "http://")
	}

	if strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(host, "https://")
	}

	return host[:12]
}

// TODO: 根据域名来获取区域
// ccr-xxxx-xxx.cnc.gz.baidubce.com
func getRegionFromHost(host string) string {
	parts := strings.Split(host, ".")
	return parts[2]
}
