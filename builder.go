package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

func Build(srcDir, deployDir, binaryName string, timeout time.Duration) (time.Duration, error) {
	binaryPath := filepath.Join(deployDir, binaryName+".new")
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	cmd.Dir = srcDir
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		return elapsed, fmt.Errorf("build failed: %s: %w", string(out), err)
	}
	return elapsed, nil
}
