package agents

import (
	"context"
	"fmt"
	"sync"
)

// Registry 管理所有 Agent 适配器
type Registry struct {
	adapters map[string]AgentAdapter
}

var (
	once     sync.Once
	registry *Registry
)

// buildAllAdapters 汇集所有内置 Agent 适配器
func buildAllAdapters() []AgentAdapter {
	return append([]AgentAdapter{
		&CodexAdapter{},
		&OpenCodeAdapter{},
	}, buildAdditionalAdapters()...)
}

// DefaultRegistry 返回全局注册表
func DefaultRegistry() *Registry {
	once.Do(func() {
		registry = NewRegistry(buildAllAdapters()...)
	})
	return registry
}

// NewRegistry 创建一个注册表并注册给定适配器
func NewRegistry(adapters ...AgentAdapter) *Registry {
	r := &Registry{adapters: make(map[string]AgentAdapter)}
	for _, a := range adapters {
		r.Register(a)
	}
	return r
}

// Register 注册适配器
func (r *Registry) Register(a AgentAdapter) {
	if a != nil {
		r.adapters[a.ID()] = a
	}
}

// List 返回所有适配器
func (r *Registry) List() []AgentAdapter {
	list := make([]AgentAdapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		list = append(list, a)
	}
	return list
}

// Get 返回指定 ID 的适配器
func (r *Registry) Get(id string) (AgentAdapter, bool) {
	a, ok := r.adapters[id]
	return a, ok
}

// DetectAll 检测所有适配器的安装状态。
// 返回 map[id]Installation；未检测到的不包含在结果中。
func (r *Registry) DetectAll(ctx context.Context) map[string]Installation {
	result := make(map[string]Installation)
	for id, adapter := range r.adapters {
		installations, err := adapter.Detect(ctx)
		if err != nil || len(installations) == 0 {
			continue
		}
		result[id] = installations[0]
	}
	return result
}

// InvalidSkillsDirError 表示 Skill 目录无效
type InvalidSkillsDirError struct {
	Adapter string
	Reason  string
}

func (e *InvalidSkillsDirError) Error() string {
	return fmt.Sprintf("Agent %s 的 Skill 目录无效: %s", e.Adapter, e.Reason)
}
