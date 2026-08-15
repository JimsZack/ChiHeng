package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JimsZack/ChiHeng/internal/store"
)

// Settings 偏好与 AI 配置域。
type Settings struct {
	store *store.Store
}

func NewSettings(repository *store.Store) *Settings {
	return &Settings{store: repository}
}

// Preferences 当前生效的偏好。
type Preferences struct {
	Locale               string
	Theme                string
	RefreshInterval      int
	DisclaimerVersion    string
	LogLevel             string
	Version              int64
}

// GetPreferences 读取偏好，未设置时返回默认值。
func (s *Settings) GetPreferences(ctx context.Context) (Preferences, error) {
	values, err := s.store.ListSettings(ctx)
	if err != nil {
		return Preferences{}, err
	}
	p := Preferences{
		Locale:            "zh-CN",
		Theme:             "system",
		RefreshInterval:   60,
		DisclaimerVersion: "1.0",
		LogLevel:          "info",
	}
	if v, ok := values["pref.locale"]; ok {
		p.Locale = v
	}
	if v, ok := values["pref.theme"]; ok {
		p.Theme = v
	}
	if v, ok := values["pref.refresh_interval"]; ok {
		fmt.Sscanf(v, "%d", &p.RefreshInterval)
	}
	if v, ok := values["pref.disclaimer_version"]; ok {
		p.DisclaimerVersion = v
	}
	if v, ok := values["pref.log_level"]; ok {
		p.LogLevel = v
	}
	return p, nil
}

// SavePreferences 保存偏好（全量覆盖）。
func (s *Settings) SavePreferences(ctx context.Context, p Preferences) error {
	updates := map[string]string{
		"pref.locale":            p.Locale,
		"pref.theme":             p.Theme,
		"pref.refresh_interval":  fmt.Sprintf("%d", p.RefreshInterval),
		"pref.disclaimer_version": p.DisclaimerVersion,
		"pref.log_level":         p.LogLevel,
	}
	for key, value := range updates {
		if err := s.store.SetSetting(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// AIProfile AI 配置（密钥不回传，仅暴露 HasSecret）。
type AIProfile struct {
	Provider          string
	Model             string
	HasSecret         bool
	Available         bool
	UnavailableReason string
}

// GetAIProfile 返回 AI 配置状态。
func (s *Settings) GetAIProfile(ctx context.Context, secrets SecretReader) (AIProfile, error) {
	values, err := s.store.ListSettings(ctx)
	if err != nil {
		return AIProfile{}, err
	}
	profile := AIProfile{
		Provider:  "deepseek",
		Model:     values["ai.model"],
		Available: false,
	}
	if profile.Model == "" {
		profile.Model = "deepseek-chat"
	}
	if secrets != nil {
		_, err := secrets.Get(ctx, "ai.api_key")
		profile.HasSecret = err == nil
	}
	if profile.HasSecret && values["ai.model"] != "" {
		profile.Available = true
	} else if !profile.HasSecret {
		profile.UnavailableReason = "未配置 API Key"
	}
	return profile, nil
}

// SaveAIProfile 保存模型配置，密钥写入 SecretStore。
func (s *Settings) SaveAIProfile(ctx context.Context, provider, model, apiKey string, secrets SecretWriter) error {
	if provider == "" {
		provider = "deepseek"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	if err := s.store.SetSetting(ctx, "ai.provider", provider); err != nil {
		return err
	}
	if err := s.store.SetSetting(ctx, "ai.model", model); err != nil {
		return err
	}
	if apiKey != "" && secrets != nil {
		return secrets.Set(ctx, "ai.api_key", apiKey)
	}
	return nil
}

// SecretWriter 密钥写入接口（适配 OS keyring 与文件存储）。
type SecretWriter interface {
	Set(ctx context.Context, name, value string) error
}

// SecretReader 密钥读取接口。
type SecretReader interface {
	Get(ctx context.Context, name string) (string, error)
}

// FileSecretStore 基于文件（0600）的密钥存储，作为 OS keyring 的轻量回退。
// 文件内容不可读回明文之外的元数据，避免日志与配置泄露。
type FileSecretStore struct {
	dir string
}

// NewFileSecretStore 创建文件密钥存储（目录不存在则创建）。
func NewFileSecretStore(dir string) (*FileSecretStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create secret dir: %w", err)
	}
	return &FileSecretStore{dir: dir}, nil
}

func (f *FileSecretStore) path(name string) string {
	sum := sha256Sum(name)
	return filepath.Join(f.dir, sum+".secret")
}

func (f *FileSecretStore) Set(_ context.Context, name, value string) error {
	path := f.path(name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(value), 0o600); err != nil {
		return fmt.Errorf("write secret: %w", err)
	}
	return os.Rename(tmp, path)
}

func (f *FileSecretStore) Get(_ context.Context, name string) (string, error) {
	data, err := os.ReadFile(f.path(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", store.ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

func (f *FileSecretStore) Delete(_ context.Context, name string) error {
	err := os.Remove(f.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func sha256Sum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// FileSecretStore 同时满足读写接口。
var _ SecretReader = (*FileSecretStore)(nil)
var _ SecretWriter = (*FileSecretStore)(nil)

// isSecretLike 判断键名是否疑似密钥（供日志脱敏辅助）。
func isSecretLike(key string) bool {
	lower := strings.ToLower(key)
	for _, token := range []string{"api_key", "secret", "token", "password"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}
