//go:build windows

package platform

import "golang.org/x/sys/windows"

// isJunctionPlatform 在 Windows 上判断路径是否是 Junction（重解析点）
// 通过读取文件属性中的 FILE_ATTRIBUTE_REPARSE_POINT 标志判断。
func isJunctionPlatform(path string) bool {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		return false
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
