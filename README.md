# phubot-pilot

Lightweight GitOps agent that keeps [phubot](https://github.com/tillknuesting/phubot) deployed and running on a Raspberry Pi. Polls GitHub for new commits, builds from source, and restarts the service automatically.

## How it works

```
Every 60s:
  1. git ls-remote → get latest commit hash
  2. Compare with last deployed commit (state file)
  3. If drift detected → reconcile:
     a. git pull
     b. go build
     c. swap binary (keep last 3 for rollback)
     d. systemctl restart phubot
     e. health check (service active?)
     f. fail? → rollback to previous binary
  4. If no drift:
     a. check service is running
     b. crashed? → auto restart (self-healing)
```

## Install

```bash
# Build and install
make install

# Or cross-compile for Raspberry Pi
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o phubot-pilot .
sudo cp phubot-pilot /usr/local/bin/
sudo cp phubot-pilot.yaml /etc/phubot-pilot.yaml
```

## Config

`/etc/phubot-pilot.yaml`:

```yaml
repo: https://github.com/tillknuesting/phubot.git
branch: main
poll_interval: 60s
deploy_dir: /opt/phubot
src_dir: /opt/phubot/src
build_timeout: 120s
rollback_versions: 3
binary_name: phubot
service_name: phubot
protect_files:
  - config.json
  - .phubot
```

`protect_files` are never overwritten during deploys — your config and data survive updates.

## Commands

```
phubot-pilot daemon       # run daemon (polls forever)
phubot-pilot status       # show deployed commit, build time, service state
phubot-pilot reconcile    # force one reconcile cycle now
phubot-pilot rollback     # revert to previous binary
phubot-pilot install      # one-time setup: dirs, Go, Chromium, systemd units
```

## systemd

```
sudo systemctl enable --now phubot-pilot
sudo systemctl status phubot-pilot
sudo journalctl -u phubot-pilot -f
```

## State file

`/opt/phubot/.pilot-state.json`:

```json
{
  "current_commit": "abc1234",
  "deployed_at": "2026-04-06T10:00:00Z",
  "build_duration_ms": 45000,
  "status": "healthy",
  "last_check": "2026-04-06T10:01:00Z",
  "rollback_commits": ["def5678"]
}
```

## Directory layout on Pi

```
/opt/phubot/
├── phubot              # active binary
├── config.json         # bot config (protected)
├── .phubot/            # bot data (protected)
├── .pilot-state.json   # pilot state
├── src/                # git checkout
└── rollback/           # last 3 backup binaries
```

## Requirements

- Go 1.24+ (on Pi, for building phubot from source)
- git
- systemctl
- Target service (phubot) with a systemd unit

## License

MIT
