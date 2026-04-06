package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func GetRemoteHead(repo, branch string) (string, error) {
	cmd := exec.Command("git", "ls-remote", repo, fmt.Sprintf("refs/heads/%s", branch))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return "", fmt.Errorf("no commits found for branch %s", branch)
	}
	return fields[0], nil
}

func CloneRepo(repo, branch, dir string) error {
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", "--depth", "1", repo, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s: %w", string(out), err)
	}
	return nil
}

func PullRepo(dir, branch string) error {
	cmd := exec.Command("git", "-C", dir, "pull", "--ff-only", "origin", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull: %s: %w", string(out), err)
	}
	return nil
}

func GetLocalHead(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
