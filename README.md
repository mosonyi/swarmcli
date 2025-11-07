#SwarmCLI

Simple CLI for managing Docker Swarm clusters similar to k9s.

![CI](https://github.com/mosonyi/swarmcli/actions/workflows/ci.yml/badge.svg)


# 🐳 swarmcli

A **terminal UI** for managing Docker Swarm clusters, inspired by [k9s](https://k9scli.io/).

---

## 👋 Welcome to swarmcli

**swarmcli** is a command-line interface tool that brings a powerful terminal-based UI to Docker Swarm, much like what [k9s](https://k9scli.io/) does for Kubernetes. Our mission is to empower Swarm users with a fast, intuitive, and feature-rich terminal experience for observing and managing services, containers, nodes, networks, and volumes in a Swarm cluster.

---

## ⚡️ Why swarmcli?

While Kubernetes has many tools for cluster management, Docker Swarm users often rely on CLI commands or custom dashboards with limited interactivity. **swarmcli** aims to fill this gap by providing:

- A **real-time, curses-based UI**
- Fast navigation between nodes, services, tasks, and containers
- Live inspection of logs and metrics
- Actions like scaling, updating, restarting, and removing resources
- Keyboard-driven workflows for efficiency

We believe Swarm deserves a first-class tool like k9s — and we’re here to build it.

---

## 🚀 Project Vision

This is an early-stage project inspired by the great work behind `k9s`. Our goal is to build something truly useful for the Docker Swarm community — a tool that combines speed, usability, and clarity.

We are actively **looking for contributors, testers, and sponsors** to help bring this vision to life.  
If you believe in Docker Swarm and want to support its ecosystem, we’d love your help!


---

## 🧭 Goals

- Build a minimal, fast, terminal UI for Docker Swarm
- Mirror some of the UX patterns and capabilities of `k9s`
- Maintain low dependency and easy installation
- Focus on practical use cases for real-world Swarm clusters

---

## 🔧 Coming Soon

- Service/task viewer
- Node status dashboard
- Container logs and shell access
- Swarm secrets and configs UI
- Overlay network inspection

---

## 💡 Inspired by k9s

This project is not affiliated with the k9s team, but we deeply admire their work. **swarmcli** is our attempt to bring a similarly powerful CLI tool to the Docker Swarm world.

## Using Docker container to build and run locally

```
docker build -t swarmcli-dev .
docker run --rm -it -v "$PWD":/app -v /var/run/docker.sock:/var/run/docker.sock  -w /app swarmcli-dev
```

or with docker compose:

```
docker compose run --build --rm swarmcli
```

Then run:
```
go run .
```

## Logging

```bash
# Production (default)
$ go run .
# → writes JSON logs to  ~/.local/state/swarmcli/app.log

# Development
$ SWARMCLI_ENV=dev go run .
# → writes pretty logs to ~/.local/state/swarmcli/app-debug.log
```

Colorize log tails. Not perfect but simple:
```bash
sudo apt install ccze
tail -f ~/.local/state/swarmcli/app-debug.log | ccze -A 
```

### Integration tests
The logs for the integration tests can be enabled with:

```bash
TEST_LOG=1 ./test-setup/testenv.sh test
```