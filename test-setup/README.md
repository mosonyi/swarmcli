# 🐳 Local Docker Swarm Test Environment

Spin up a 1× manager + 2× worker Docker‑in‑Docker (DinD) Swarm locally, expose the manager’s Docker API on **localhost:22375**, and interact with it via a Docker context.

> ⚠️ Security note: this setup binds the Docker API without TLS for local testing only. Do **not** expose it on an untrusted network.

---

## Prerequisites

- Docker Engine (with the **docker compose** plugin)
- Linux/macOS/WSL recommended
- Ports **22375/tcp** on localhost must be free

---

## Quick start

```bash
# From the repo folder that contains docker-compose.yml
docker compose up -d --build
```

Wait ~10–20 seconds for the Swarm to bootstrap (manager + workers join).

Check containers:
```bash
docker compose ps manager worker1 worker2
```

Create a Docker context pointing to the manager API on 22375:
* If you're using devcontainer:
  ```bash
  docker context create swarmcli --description "Test SwarmCLI" --docker "host=tcp://host.docker.internal:22375"
  ```

* Otherwise:
  ```bash
  docker context create swarmcli --description "Test SwarmCLI" --docker "host=tcp://localhost:22375"
  ```


Use the new context:
```bash
docker --context swarmcli info | sed -n '/Swarm:/,/ClusterID/p'
docker --context swarmcli node ls
```

You should see **Swarm: active** and three nodes (1 manager, 2 workers).

---

## Verify from inside the manager (no hardcoded container names)

Container names get a project prefix (e.g. `test-manager-1`). Always resolve the container dynamically:

```bash
# Exec into the manager container in a robust way
docker exec -it $(docker compose ps -q manager) sh -lc 'docker info | sed -n "/Swarm:/,/ClusterID/p"'
docker exec -it $(docker compose ps -q manager) docker node ls
```

Alternatively, use the compose-native form (avoids dealing with names/IDs):
```bash
docker compose exec manager docker info | sed -n '/Swarm:/,/ClusterID/p'
docker compose exec manager docker node ls
```

> Note: The earlier example that used `docker exec -it test-manager-1 ...` can fail if your Compose project name is not `test`. Use the dynamic form above instead.

---

## Deploy a quick test service (optional)

```bash
docker --context swarmcli service create --name whoami --publish 8080:80 traefik/whoami:v1.10
curl -fsS http://localhost:8080
docker --context swarmcli service rm whoami
```

## Deploy a quick test stack (optional)

```bash
docker --context swarmcli stack deploy -c test-setup/test-stack.yml demo
```

---

## Tear down

```bash
# Stop and remove containers, networks, and volumes created by the stack
docker compose down -v
```

> 💡 **Tip**: If you encounter "process with PID 1 is still running" errors when starting, clean up completely and rebuild:
> ```bash
> docker compose down -v  # Remove containers and volumes
> docker compose up --build
> ```

### If port 22375 still appears to be listening

Very rarely the DinD dockerd inside the manager can linger. If `ss` shows the port still open:

```bash
sudo ss -ltnp | grep :22375 || sudo lsof -iTCP:22375 -sTCP:LISTEN -P -n
```

Fix by restarting your host Docker service (this won’t affect your system beyond restarting Docker):

```bash
# Linux (systemd)
sudo systemctl restart docker

# macOS / Docker Desktop: quit and relaunch Docker Desktop
# Windows / Docker Desktop: quit and relaunch Docker Desktop
```

After restart, re‑run:
```bash
docker compose up -d
```

---

## Tips

- To force a clean project prefix for container names, run:
  ```bash
  docker compose --project-name test-setup up -d
  ```
  Then your containers will be `test-manager-1`, `test-worker1-1`, etc.

- To point the CLI directly without creating a context (temporary):
  ```bash
  export DOCKER_HOST=tcp://localhost:22375
  docker info
  ```

---

## What’s included

- **manager**: DinD with Docker API published on **22375/tcp** (localhost)
- **worker1**, **worker2**: DinD workers joining the Swarm
- **swarm-init**: one-shot helper to initialize the manager and generate a join token
- **worker*-join**: one-shot helpers to join the workers

---

## Known warnings

You’ll see deprecation warnings about binding the Docker API without TLS. This is expected for this local testbed. For any real environment, enable TLS with client verification.
