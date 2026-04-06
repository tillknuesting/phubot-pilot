package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Reconciler struct {
	cfg       *Config
	stateFile string
}

func NewReconciler(cfg *Config) *Reconciler {
	return &Reconciler{
		cfg:       cfg,
		stateFile: filepath.Join(cfg.DeployDir, ".pilot-state.json"),
	}
}

func (r *Reconciler) Run() {
	for {
		if err := r.reconcile(); err != nil {
			log.Printf("[reconciler] error: %v", err)
		}
		time.Sleep(r.cfg.PollInterval)
	}
}

func (r *Reconciler) ReconcileOnce() error {
	return r.reconcile()
}

func (r *Reconciler) reconcile() error {
	state, err := LoadState(r.stateFile)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	remoteHead, err := GetRemoteHead(r.cfg.Repo, r.cfg.Branch)
	if err != nil {
		state.Status = "error"
		SaveState(r.stateFile, state)
		return fmt.Errorf("get remote head: %w", err)
	}

	if remoteHead == state.CurrentCommit {
		if !IsServiceActive(r.cfg.ServiceName) {
			log.Printf("[reconciler] service %s not running, healing", r.cfg.ServiceName)
			if err := restartService(r.cfg.ServiceName); err != nil {
				state.Status = "error"
				SaveState(r.stateFile, state)
				return fmt.Errorf("heal restart: %w", err)
			}
			state.Status = "healed"
		}
		state.PilotVersion = version
		if state.PhubotVersion == "" && state.CurrentCommit != "" {
			state.PhubotVersion = state.CurrentCommit[:8]
		}
		SaveState(r.stateFile, state)
		return nil
	}

	if state.CurrentCommit == "" {
		log.Printf("[reconciler] initial deploy: remote=%s", remoteHead[:8])
	} else {
		log.Printf("[reconciler] drift detected: deployed=%s remote=%s", state.CurrentCommit[:8], remoteHead[:8])
	}

	if err := r.syncAndBuild(remoteHead, state); err != nil {
		state.Status = "build_failed"
		SaveState(r.stateFile, state)
		return fmt.Errorf("sync and build: %w", err)
	}

	return nil
}

func (r *Reconciler) syncAndBuild(commit string, state *State) error {
	if _, err := os.Stat(r.cfg.SrcDir); os.IsNotExist(err) {
		log.Printf("[reconciler] cloning repo into %s", r.cfg.SrcDir)
		if err := CloneRepo(r.cfg.Repo, r.cfg.Branch, r.cfg.SrcDir); err != nil {
			return err
		}
	} else {
		log.Printf("[reconciler] pulling latest in %s", r.cfg.SrcDir)
		if err := PullRepo(r.cfg.SrcDir, r.cfg.Branch); err != nil {
			return err
		}
	}

	protectDir := filepath.Join(r.cfg.DeployDir, ".pilot-protect")
	os.RemoveAll(protectDir)
	os.MkdirAll(protectDir, 0755)
	for _, f := range r.cfg.ProtectFiles {
		src := filepath.Join(r.cfg.DeployDir, f)
		dst := filepath.Join(protectDir, f)
		if _, err := os.Stat(src); err == nil {
			copyPath(src, dst)
		}
	}

	buildDur, err := Build(r.cfg.SrcDir, r.cfg.DeployDir, r.cfg.BinaryName, commit, r.cfg.BuildTimeout)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}

	if err := Deploy(r.cfg); err != nil {
		log.Printf("[reconciler] deploy failed, rolling back")
		Rollback(r.cfg)
		return fmt.Errorf("deploy: %w", err)
	}

	for _, f := range r.cfg.ProtectFiles {
		src := filepath.Join(protectDir, f)
		dst := filepath.Join(r.cfg.DeployDir, f)
		if _, err := os.Stat(src); err == nil {
			copyPath(src, dst)
		}
	}
	os.RemoveAll(protectDir)

	time.Sleep(3 * time.Second)
	if !IsServiceActive(r.cfg.ServiceName) {
		log.Printf("[reconciler] service failed to start, rolling back")
		Rollback(r.cfg)
		return fmt.Errorf("service failed after deploy")
	}

	state.CurrentCommit = commit
	state.DeployedAt = time.Now().UTC()
	state.BuildDurationMs = buildDur.Milliseconds()
	state.Status = "healthy"
	state.PilotVersion = version
	state.PhubotVersion = commit[:8]
	if len(state.RollbackCommits) > r.cfg.RollbackVersions {
		state.RollbackCommits = state.RollbackCommits[len(state.RollbackCommits)-r.cfg.RollbackVersions:]
	}
	return SaveState(r.stateFile, state)
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		os.MkdirAll(dst, info.Mode())
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFile(src, dst)
}
