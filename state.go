package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type State struct {
	CurrentCommit   string    `json:"current_commit"`
	DeployedAt      time.Time `json:"deployed_at"`
	BuildDurationMs int64     `json:"build_duration_ms"`
	Status          string    `json:"status"`
	LastCheck       time.Time `json:"last_check"`
	RollbackCommits []string  `json:"rollback_commits"`
	PilotVersion    string    `json:"pilot_version"`
	PhubotVersion   string    `json:"phubot_version"`
}

func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Status: "none"}, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

func SaveState(path string, s *State) error {
	s.LastCheck = time.Now().UTC()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
