package http

import (
	"fmt"
	"net"
	neturl "net/url"
	"strconv"
	"strings"
)

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty url")
	}

	u, err := neturl.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported url scheme: %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty host in url")
	}

	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}

		dangerousPorts := []int{22, 23, 25, 3306, 3389, 5432, 6379, 9200, 27017}
		for _, dangerous := range dangerousPorts {
			if port == dangerous {
				return fmt.Errorf("refuse dangerous port: %d", port)
			}
		}
	}

	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "::1" || lower == "127.0.0.1" {
		return fmt.Errorf("refuse loopback host: %s", host)
	}

	if ip := net.ParseIP(host); ip != nil {
		if err := validateIP(ip, host); err != nil {
			return err
		}
	}

	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if err := validateIP(ip, fmt.Sprintf("%s -> %s", host, ip.String())); err != nil {
				return fmt.Errorf("resolved to unsafe ip: %w", err)
			}
		}
	}

	return nil
}

func validateIP(ip net.IP, context string) error {

	if ip.IsLoopback() {
		return fmt.Errorf("refuse loopback ip: %s", context)
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("refuse link-local ip: %s", context)
	}

	if ip.IsMulticast() {
		return fmt.Errorf("refuse multicast ip: %s", context)
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("refuse unspecified ip: %s", context)
	}

	if ip4 := ip.To4(); ip4 != nil {

		if ip4[0] == 10 {
			return fmt.Errorf("refuse private ip range 10.0.0.0/8: %s", context)
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return fmt.Errorf("refuse private ip range 172.16.0.0/12: %s", context)
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return fmt.Errorf("refuse private ip range 192.168.0.0/16: %s", context)
		}

		if ip4[0] == 169 && ip4[1] == 254 {
			return fmt.Errorf("refuse link-local/metadata ip 169.254.0.0/16: %s", context)
		}

		if ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 {
			return fmt.Errorf("refuse broadcast address: %s", context)
		}

		if ip4[0] == 0 {
			return fmt.Errorf("refuse 0.0.0.0/8 range: %s", context)
		}

		if ip4[0] == 127 {
			return fmt.Errorf("refuse loopback range 127.0.0.0/8: %s", context)
		}
	}

	if ip.To4() == nil && len(ip) == net.IPv6len {

		if ip[0]&0xfe == 0xfc {
			return fmt.Errorf("refuse ipv6 unique local address fc00::/7: %s", context)
		}

		if ip[0] == 0xfe && (ip[1]&0xc0) == 0xc0 {
			return fmt.Errorf("refuse ipv6 site-local address fec0::/10: %s", context)
		}
	}

	return nil
}
