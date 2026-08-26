package platform

// LinkMode 表示创建的链接类型
type LinkMode string

const (
	// LinkModeJunction Windows 目录联接（无需管理员权限）
	LinkModeJunction LinkMode = "junction"
	// LinkModeSymlink 符号链接（Windows 需开发者模式或管理员；macOS 原生支持）
	LinkModeSymlink LinkMode = "symlink"
	// LinkModeCopy 托管复制（无法创建链接时回退：复制一份内容并登记）
	LinkModeCopy LinkMode = "copy"
)

// LinkInfo 描述一个已存在链接的检查结果
type LinkInfo struct {
	Exists       bool     // 目标是否存在
	Mode         LinkMode // 检测到的链接模式
	SourcePath   string   // 解析到的来源路径（若可解析）
	IsManaged    bool     // 是否是本管理器创建（由调用方通过数据库判断）
	TargetExists bool     // 目标路径本身是否存在
}

// Linker 是跨平台文件链接统一接口（设计文档第 10 节）
type Linker interface {
	// Create 在 target 处创建一个指向 source 的链接
	// 若 target 已存在非空内容，返回冲突错误，不覆盖任何内容。
	Create(source, target string) (LinkMode, error)
	// Inspect 检查 target 路径当前的链接状态
	Inspect(target string) (LinkInfo, error)
	// RemoveManaged 移除一个由管理器创建的链接（target -> expectedSource）
	// 删除前必须确认 target 指向 expectedSource，避免误删用户内容。
	RemoveManaged(target, expectedSource string) error
}

// ConflictError 表示目标位置存在同名冲突
type ConflictError struct {
	Target string
}

func (e *ConflictError) Error() string {
	return "目标位置已存在同名内容，已跳过以免覆盖: " + e.Target
}
