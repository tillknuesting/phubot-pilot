package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Deploy(cfg *Config) error {
	newBinary := filepath.Join(cfg.DeployDir, cfg.BinaryName+".new")
	activeBinary := filepath.Join(cfg.DeployDir, cfg.BinaryName)
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("new binary not found: %s", newBinary)
	}
	if err := os.Chmod(newBinary, 0755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}
	backupDir := filepath.Join(cfg.DeployDir, "rollback")
	os.MkdirAll(backupDir, 0755)
	if _, err := os.Stat(activeBinary); err == nil {
		localHead, _ := GetLocalHead(cfg.SrcDir)
		if localHead != "" {
			backupPath := filepath.Join(backupDir, cfg.BinaryName+"."+localHead[:8])
			copyFile(activeBinary, backupPath)
			cleanOldBackups(backupDir, cfg.BinaryName, cfg.RollbackVersions)
		}
	}
	if err := os.Rename(newBinary, activeBinary); err != nil {
		return fmt.Errorf("swap binary: %w", err)
	}
	if err := restartService(cfg.ServiceName); err != nil {
		return fmt.Errorf("restart service: %w", err)
	}
	return nil
}

func Rollback(cfg *Config) error {
	backupDir := filepath.Join(cfg.DeployDir, "rollback")
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no rollback available")
	}
	latest := entries[len(entries)-1]
	activeBinary := filepath.Join(cfg.DeployDir, cfg.BinaryName)
	backupPath := filepath.Join(backupDir, latest.Name())
	if err := copyFile(backupPath, activeBinary); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}
	os.Chmod(activeBinary, 0755)
	os.Remove(backupPath)
	if err := restartService(cfg.ServiceName); err != nil {
		return fmt.Errorf("restart after rollback: %w", err)
	}
	return nil
}

func IsServiceActive(name string) bool {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

func restartService(name string) error {
	out, err := exec.Command("sudo", "systemctl", "restart", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func cleanOldBackups(dir, binaryName string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var backups []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), binaryName+".") {
			backups = append(backups, e)
		}
	}
	for i := 0; i < len(backups)-keep; i++ {
		os.Remove(filepath.Join(dir, backups[i].Name()))
	}
}
