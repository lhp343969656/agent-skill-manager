//go:build !windows

package platform

// isJunctionPlatform 在非 Windows 平台始终返回 false
func isJunctionPlatform(path string) bool {
	return false
}
