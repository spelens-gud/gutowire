package generator

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileCache struct    文件缓存信息.
type FileCache struct {
	ModTime  time.Time `json:"mod_time"` // 文件修改时间
	Elements []Element `json:"elements"` // 解析出的元素
	Hash     string    `json:"hash"`     // 文件内容哈希
}

// CacheManager struct    缓存管理器.
type CacheManager struct {
	cacheFile string                // 缓存文件路径
	cache     map[string]*FileCache // 文件路径 -> 缓存信息
	mu        sync.RWMutex          // 读写锁
	enabled   bool                  // 是否启用缓存
}

// NewCacheManager function    创建缓存管理器.
func NewCacheManager(genPath string, enabled bool) *CacheManager {
	return &CacheManager{
		cacheFile: filepath.Join(genPath, ".gutowire.cache"),
		cache:     make(map[string]*FileCache),
		enabled:   enabled,
	}
}

// Load method    加载缓存.
func (cm *CacheManager) Load() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 缓存文件不存在是正常的
		}
		return fmt.Errorf("读取缓存文件失败: %w", err)
	}

	if err := json.Unmarshal(data, &cm.cache); err != nil {
		return fmt.Errorf("解析缓存文件失败: %w", err)
	}

	return nil
}

// Save method    保存缓存.
func (cm *CacheManager) Save() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, err := json.MarshalIndent(cm.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	//nolint:gosec
	if err := os.WriteFile(cm.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("写入缓存文件失败: %w", err)
	}

	return nil
}

// IsModified method    检查文件是否被修改.
func (cm *CacheManager) IsModified(filePath string) (bool, error) {
	if !cm.enabled {
		return true, nil // 缓存未启用，总是返回已修改
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return true, err
	}

	cm.mu.RLock()
	cached, exists := cm.cache[filePath]
	cm.mu.RUnlock()

	if !exists {
		return true, nil // 缓存中不存在
	}

	// 比较修改时间
	if !info.ModTime().Equal(cached.ModTime) {
		return true, nil
	}

	// 可选：比较文件哈希（更精确但更慢）
	hash, err := cm.calculateHash(filePath)
	if err != nil {
		return true, err
	}

	return hash != cached.Hash, nil
}

// Get method    获取缓存的元素.
func (cm *CacheManager) Get(filePath string) ([]Element, bool) {
	if !cm.enabled {
		return nil, false
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cached, exists := cm.cache[filePath]
	if !exists {
		return nil, false
	}

	return cached.Elements, true
}

// Set method    设置缓存.
func (cm *CacheManager) Set(filePath string, elements []Element) error {
	if !cm.enabled {
		return nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	hash, err := cm.calculateHash(filePath)
	if err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache[filePath] = &FileCache{
		ModTime:  info.ModTime(),
		Elements: elements,
		Hash:     hash,
	}

	return nil
}

// Clear method    清空缓存.
func (cm *CacheManager) Clear() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.Lock()
	cm.cache = make(map[string]*FileCache)
	cm.mu.Unlock()

	return os.Remove(cm.cacheFile)
}

// calculateHash method    计算文件内容哈希.
func (cm *CacheManager) calculateHash(filePath string) (string, error) {
	//nolint:gosec
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	//nolint:gosec
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}
