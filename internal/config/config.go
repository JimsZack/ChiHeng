// Package config 集中管理应用标识与用户数据目录，避免路径与常量散落各处。
package config

import (
	"os"
	"path/filepath"
)

// 应用标识（技术元数据统一使用 ChiHeng，展示名使用「持衡 ChiHeng」）。
const (
	AppName        = "chiheng"
	AppDisplayName = "持衡 ChiHeng"
	Tagline        = "看清持仓，理性权衡。"
	Version        = "0.1.0"
	DBFileName     = "chiheng.db"
)

// DataDir 返回用户数据目录（os.UserConfigDir()/chiheng），不存在则创建。
func DataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// DBPath 返回 SQLite 数据库文件绝对路径。
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DBFileName), nil
}

// LogDir 返回日志目录，不存在则创建。
func LogDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	return logDir, nil
}
