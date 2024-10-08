package gxxljob

import "sync"

type taskList struct {
	mu   sync.RWMutex
	data map[string]*Task
}

// Set 设置数据
func (t *taskList) Set(key string, val *Task) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data[key] = val
}

// Get 获取数据
func (t *taskList) Get(key string) *Task {
	return t.data[key]
}

// Del 设置数据
func (t *taskList) Del(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.data, key)
}

// Len 长度
func (t *taskList) Len() int {
	return len(t.data)
}

// Exists Key是否存在
func (t *taskList) Exists(key string) bool {
	_, ok := t.data[key]
	return ok
}
