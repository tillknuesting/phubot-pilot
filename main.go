package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "/etc/phubot-pilot.yaml", "config file path")
	flag.Parse()

	args := flag.Args()
	cmd := "daemon"
	if len(args) > 0 {
		cmd = args[0]
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	switch cmd {
	case "daemon":
		log.Println("[pilot] starting daemon")
		log.Printf("[pilot] repo=%s branch=%s poll=%s", cfg.Repo, cfg.Branch, cfg.PollInterval)
		log.Printf("[pilot] version=%s", version)
		r := NewReconciler(cfg)
		r.Run()

	case "status":
		r := NewReconciler(cfg)
		state, err := LoadState(r.stateFile)
		if err != nil {
			log.Fatalf("load state: %v", err)
		}
		fmt.Printf("Status:     %s\n", state.Status)
		fmt.Printf("Pilot:      %s\n", version)
		if state.CurrentCommit != "" {
			fmt.Printf("Commit:     %s\n", state.CurrentCommit[:8])
		}
		fmt.Printf("Deployed:   %s\n", state.DeployedAt.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("Last check: %s\n", state.LastCheck.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("Build:      %dms\n", state.BuildDurationMs)
		fmt.Printf("Rollbacks:  %d\n", len(state.RollbackCommits))
		active := IsServiceActive(cfg.ServiceName)
		if active {
			fmt.Printf("Service:    active\n")
		} else {
			fmt.Printf("Service:    inactive\n")
		}

	case "reconcile":
		log.Println("[pilot] forcing reconcile")
		r := NewReconciler(cfg)
		if err := r.ReconcileOnce(); err != nil {
			log.Fatalf("reconcile failed: %v", err)
		}
		log.Println("[pilot] reconcile complete")

	case "rollback":
		log.Println("[pilot] rolling back")
		if err := Rollback(cfg); err != nil {
			log.Fatalf("rollback failed: %v", err)
		}
		log.Println("[pilot] rollback complete")

	case "install":
		if err := Install(cfg); err != nil {
			log.Fatalf("install failed: %v", err)
		}
		log.Println("[pilot] install complete")

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: phubot-pilot [daemon|status|reconcile|rollback|install]\n", cmd)
		os.Exit(1)
	}
}
