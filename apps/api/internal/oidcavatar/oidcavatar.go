package oidcavatar

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Profile struct {
	Name             string
	PreferredUsername string
	GivenName        string
	FamilyName       string
	Picture          string
}

func (p Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.PreferredUsername != "" {
		return p.PreferredUsername
	}
	full := strings.TrimSpace(p.GivenName + " " + p.FamilyName)
	if full != "" {
		return full
	}
	return ""
}

const maxAvatarSize = 5 << 20

func FetchAvatar(pictureURL, storageDir string, userID int64, logger *slog.Logger) (string, error) {
	parsed, err := url.Parse(pictureURL)
	if err != nil {
		return "", fmt.Errorf("invalid picture URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("picture URL must use HTTPS")
	}

	host := parsed.Hostname()
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return "", fmt.Errorf("picture URL resolves to private IP")
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Get(pictureURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch picture: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("picture URL returned status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	ext, ok := avatarExtension(ct)
	if !ok {
		return "", fmt.Errorf("unsupported content type: %s", ct)
	}

	filename := fmt.Sprintf("oidc-%d-%d%s", userID, time.Now().UnixNano(), ext)
	relativePath := filepath.Join("avatars", filename)
	absolutePath := filepath.Join(storageDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", fmt.Errorf("failed to prepare avatar directory: %w", err)
	}

	file, err := os.Create(absolutePath)
	if err != nil {
		return "", fmt.Errorf("failed to create avatar file: %w", err)
	}

	limited := io.LimitReader(resp.Body, maxAvatarSize+1)
	n, err := io.Copy(file, limited)
	if cerr := file.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("failed to write avatar file: %w", err)
	}
	if n > maxAvatarSize {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("avatar exceeds 5MB limit")
	}

	logger.Info("synced OIDC avatar", slog.Int64("user_id", userID), slog.String("path", relativePath))
	return strings.ReplaceAll(relativePath, string(filepath.Separator), "/"), nil
}

func RemoveFile(storageDir, relativePath string) {
	if relativePath == "" {
		return
	}
	abs := filepath.Join(storageDir, filepath.Clean(relativePath))
	avatarsDir := filepath.Clean(filepath.Join(storageDir, "avatars"))
	if !strings.HasPrefix(abs, avatarsDir) {
		return
	}
	_ = os.Remove(abs)
}

func isPrivateIP(ip net.IP) bool {
	private := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)},
	}
	for _, cidr := range private {
		if cidr.Contains(ip) {
			return true
		}
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

func avatarExtension(contentType string) (string, bool) {
	ct := strings.SplitN(contentType, ";", 2)[0]
	ct = strings.TrimSpace(ct)
	switch ct {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}
