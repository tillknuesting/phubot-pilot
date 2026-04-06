package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Install(cfg *Config) error {
	log.Println("[install] creating directories")
	for _, dir := range []string{cfg.DeployDir, cfg.SrcDir, filepath.Join(cfg.DeployDir, "rollback")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	log.Println("[install] checking Go installation")
	if _, err := exec.LookPath("go"); err != nil {
		log.Println("[install] Go not found, installing...")
		if err := installGo(); err != nil {
			return fmt.Errorf("install go: %w", err)
		}
	}

	log.Println("[install] checking Chromium installation")
	if _, err := exec.LookPath("chromium-browser"); err != nil {
		if _, err := exec.LookPath("chromium"); err != nil {
			log.Println("[install] Chromium not found, installing...")
			if out, err := exec.Command("sudo", "apt-get", "install", "-y", "chromium-browser").CombinedOutput(); err != nil {
				log.Printf("[install] chromium install warning: %s: %v", string(out), err)
			}
		}
	}

	log.Println("[install] cloning phubot repo")
	if _, err := os.Stat(filepath.Join(cfg.SrcDir, ".git")); os.IsNotExist(err) {
		if err := CloneRepo(cfg.Repo, cfg.Branch, cfg.SrcDir); err != nil {
			return fmt.Errorf("clone: %w", err)
		}
	}

	log.Println("[install] writing phubot.service")
	phubotService := fmt.Sprintf(`[Unit]
Description=Phubot Personal AI Assistant
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s/%s
Restart=always
RestartSec=5
Environment=HOME=/home/pi

[Install]
WantedBy=multi-user.target
`, cfg.DeployDir, cfg.DeployDir, cfg.BinaryName)
	if err := writeSystemdUnit("phubot", phubotService); err != nil {
		return err
	}

	log.Println("[install] writing phubot-pilot.service")
	pilotBinary, _ := os.Executable()
	pilotService := fmt.Sprintf(`[Unit]
Description=Phubot Pilot - GitOps Deploy Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s daemon
Restart=always
RestartSec=5
Environment=HOME=/home/pi

[Install]
WantedBy=multi-user.target
`, pilotBinary)
	if err := writeSystemdUnit("phubot-pilot", pilotService); err != nil {
		return err
	}

	log.Println("[install] reloading systemd")
	if out, err := exec.Command("sudo", "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %s: %w", string(out), err)
	}

	log.Println("[install] building phubot")
	if _, err := Build(cfg.SrcDir, cfg.DeployDir, cfg.BinaryName, version, cfg.BuildTimeout); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	activeBinary := filepath.Join(cfg.DeployDir, cfg.BinaryName)
	newBinary := filepath.Join(cfg.DeployDir, cfg.BinaryName+".new")
	if err := os.Rename(newBinary, activeBinary); err != nil {
		return fmt.Errorf("rename binary: %w", err)
	}

	log.Println("[install] done. Next steps:")
	log.Println("  1. Create /opt/phubot/config.json with your credentials")
	log.Println("  2. sudo systemctl enable --now phubot phubot-pilot")
	return nil
}

func writeSystemdUnit(name, content string) error {
	tmpFile := filepath.Join(os.TempDir(), name+".service")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return err
	}
	out, err := exec.Command("sudo", "cp", tmpFile, filepath.Join("/etc/systemd/system", name+".service")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("write %s.service: %s: %w", name, string(out), err)
	}
	return nil
}

func installGo() error {
	arch := runtime.GOARCH
	goURL := fmt.Sprintf("https://go.dev/dl/go1.24.4.linux-%s.tar.gz", arch)
	log.Printf("[install] downloading Go from %s", goURL)
	out, err := exec.Command("curl", "-sSL", goURL, "-o", "/tmp/go.tar.gz").CombinedOutput()
	if err != nil {
		return fmt.Errorf("download go: %s: %w", string(out), err)
	}
	out, err = exec.Command("sudo", "tar", "-C", "/usr/local", "-xzf", "/tmp/go.tar.gz").CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract go: %s: %w", string(out), err)
	}
	os.Remove("/tmp/go.tar.gz")

	profile := "/etc/profile.d/go.sh"
	content := `export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
`
	if err := os.WriteFile("/tmp/go.sh", []byte(content), 0644); err != nil {
		return err
	}
	exec.Command("sudo", "cp", "/tmp/go.sh", profile).Run()
	log.Println("[install] Go installed. You may need to re-login or run: source /etc/profile.d/go.sh")
	return nil
}
