# Task Manager — Go Microservices with Jenkins CI/CD on AWS EC2

A learning project that demonstrates a full DevOps workflow: two Go microservices backed by PostgreSQL, containerised with Docker Compose, and automatically built and deployed by a Jenkins pipeline whenever code is pushed to GitHub.

---

## Table of Contents

1. [How the System Works](#how-the-system-works)
2. [EC2 Setup](#ec2-setup)
   - [Step 1: Launch an EC2 Instance](#step-1-launch-an-ec2-instance)
   - [Step 2: Configure the Security Group](#step-2-configure-the-security-group)
   - [Step 3: Connect to the Instance](#step-3-connect-to-the-instance)
   - [Step 4: Run the Bootstrap Script](#step-4-run-the-bootstrap-script)
   - [Step 5: Complete the Jenkins Setup Wizard](#step-5-complete-the-jenkins-setup-wizard)
   - [Step 6: Create the Secrets File](#step-6-create-the-secrets-file)
   - [Step 7: Create the Jenkins Pipeline Job](#step-7-create-the-jenkins-pipeline-job)
   - [Step 8: Configure the GitHub Webhook](#step-8-configure-the-github-webhook)
   - [Step 9: Trigger and Verify the First Pipeline Run](#step-9-trigger-and-verify-the-first-pipeline-run)
3. [The Jenkins Pipeline — Explained](#the-jenkins-pipeline--explained)
   - [Setup](#setup-before-any-stage-runs)
   - [Stage 1 — Checkout](#stage-1--checkout)
   - [Stage 2 — Test](#stage-2--test-parallel)
   - [Stage 3 — Build Images](#stage-3--build-images-parallel)
   - [Stage 4 — Deploy](#stage-4--deploy)
   - [Full trigger-to-live sequence](#full-trigger-to-live-sequence)
4. [Application Architecture](#application-architecture)
5. [API Reference](#api-reference)
6. [Go Code Conventions](#go-code-conventions)

---

## How the System Works

```
Developer → git push → GitHub → webhook → Jenkins (EC2)
                                               │
                               ┌───────────────┼───────────────┐
                               ▼               ▼               ▼
                            Run tests    Build images      Deploy via
                           (Go tests)    (Docker build)  docker compose
                                               │
                               ┌───────────────┴───────────────┐
                               ▼                               ▼
                         auth-service                    api-service
                           :8081                            :8090
                               │                               │
                               └───────────┬───────────────────┘
                                           ▼
                                       PostgreSQL
                                    (two databases)
```

Everything runs on **one AWS EC2 instance**. Jenkins is installed directly on the machine. The two Go services and PostgreSQL run as Docker containers managed by Docker Compose.

---

## EC2 Setup

### Step 1: Launch an EC2 Instance

1. Navigate to the **AWS EC2 console** and click **Launch Instance**.
2. Choose **Amazon Linux 2023 AMI**.
3. Select instance type — **t3.small** (2 GB RAM minimum) or **t3.medium** (4 GB, recommended).
4. Create and download a new key pair (`.pem` file) for SSH access.
5. Under **Network settings**, create a new security group with the inbound rules in Step 2 below.

---

### Step 2: Configure the Security Group

| Type       | Protocol | Port | Source   | Purpose      |
|------------|----------|------|----------|--------------|
| SSH        | TCP      | 22   | Your IP  | SSH access   |
| Custom TCP | TCP      | 8080 | Anywhere | Jenkins UI   |
| Custom TCP | TCP      | 8081 | Anywhere | auth-service |
| Custom TCP | TCP      | 8090 | Anywhere | api-service  |

> **Tip:** Allocate an **Elastic IP** in the EC2 console and associate it with your instance. This keeps the Jenkins webhook URL stable if the instance ever restarts.

---

### Step 3: Connect to the Instance

```bash
ssh -i /path/to/your-key.pem ec2-user@<EC2_PUBLIC_IP>
```

---

### Step 4: Run the Bootstrap Script

Clone your repository and run the one-time setup script. It installs all required tools and wires up permissions automatically.

```bash
# Clone the repo
git clone https://github.com/<your-username>/devops-ci-cd-n-tier-app.git
cd devops-ci-cd-n-tier-app

# Run the bootstrap (takes ~3–5 minutes)
bash scripts/ec2-setup.sh
```

What the script installs, in order:

1. **Docker** — container runtime; `ec2-user` added to `docker` group
2. **Docker Compose v2** — as a Docker CLI plugin
3. **Java 21** (Amazon Corretto) — required by Jenkins
4. **Jenkins LTS** — enabled as a systemd service on port 8080
5. **Go 1.22** — installed to `/usr/local/go`; path written to `/etc/profile.d/golang.sh`
6. `jenkins` user added to `docker` group so the pipeline can run `docker compose`

At the end the script prints the **Jenkins initial admin password** and your instance's public IP.

---

### Step 5: Complete the Jenkins Setup Wizard

1. Open `http://<EC2_PUBLIC_IP>:8080` in your browser.
2. Paste the initial admin password (printed by the script, or retrieve it with):
   ```bash
   sudo cat /var/lib/jenkins/secrets/initialAdminPassword
   ```
3. Click **Install suggested plugins** and wait for installation to finish.
4. Create your admin user and complete the wizard.

---

### Step 6: Create the Secrets File

This file holds the database password and JWT signing key. It lives only on the EC2 instance and is **never committed to Git**.

```bash
sudo tee /home/ec2-user/app.env > /dev/null <<'EOF'
POSTGRES_PASSWORD=<choose a strong password>
JWT_SECRET=<random string, at least 32 characters>
EOF

sudo chmod 600 /home/ec2-user/app.env
sudo chown jenkins:jenkins /home/ec2-user/app.env
```

---

### Step 7: Create the Jenkins Pipeline Job

1. From the Jenkins dashboard click **New Item**.
2. Enter name `task-manager`, select **Pipeline**, click **OK**.
3. Under **Build Triggers** → check **GitHub hook trigger for GITScm polling**.
4. Under **Pipeline**, set:
   - **Definition:** Pipeline script from SCM
   - **SCM:** Git
   - **Repository URL:** `https://github.com/<your-username>/devops-ci-cd-n-tier-app.git`
   - **Branch:** `*/main`
   - **Script Path:** `Jenkinsfile`
5. Click **Save**.

---

### Step 8: Configure the GitHub Webhook

1. In your GitHub repository go to **Settings → Webhooks → Add webhook**.
2. Fill in the fields:
   - **Payload URL:** `http://<EC2_PUBLIC_IP>:8080/github-webhook/`
   - **Content type:** `application/json`
   - **Which events:** Just the push event
3. Click **Add webhook**.

GitHub sends a ping immediately. A green tick next to the webhook means Jenkins received it successfully.

---

### Step 9: Trigger and Verify the First Pipeline Run

Push any commit to `main`:

```bash
git commit --allow-empty -m "trigger first pipeline run"
git push origin main
```

Jenkins should start the pipeline within seconds. After it turns green, verify the services are live:

```bash
curl http://<EC2_PUBLIC_IP>:8081/health
# → {"status":"ok"}

curl http://<EC2_PUBLIC_IP>:8090/health
# → {"status":"ok"}
```

Confirm all three containers are running on the instance:

```bash
docker ps
# postgres, auth-service, and api-service should all show "Up"
```

---

## The Jenkins Pipeline — Explained

The `Jenkinsfile` at the root of this repo defines a **declarative pipeline** with 4 sequential stages. Jenkins reads this file from Git and executes it every time a push is made to the `main` branch.

### Setup (before any stage runs)

```groovy
IMAGE_TAG = "${env.GIT_COMMIT[0..6]}"
PATH = "/usr/local/go/bin:${env.PATH}"
```

- `IMAGE_TAG` captures the first 7 characters of the Git commit SHA (e.g. `a3f91c2`). Every Docker image built in this run gets tagged with it — so you can always trace which commit a running container came from.
- `PATH` injects the Go binary location because Jenkins' shell doesn't load the system profile where Go was installed.

---

### Stage 1 — Checkout

```groovy
checkout scm
```

Jenkins clones (or pulls) the GitHub repository into its workspace on the EC2 machine at `/var/lib/jenkins/workspace/task-manager/`. All subsequent stages work from this directory.

---

### Stage 2 — Test *(parallel)*

```groovy
dir('services/api-service')  { sh 'go test ./... -v -count=1' }
dir('services/auth-service') { sh 'go test ./... -v -count=1' }
```

Both test suites run **at the same time** to save time. `-count=1` disables Go's test cache so tests always actually execute.

**If either test fails, the pipeline stops here.** Broken code never gets built into an image or deployed.

---

### Stage 3 — Build Images *(parallel)*

```groovy
docker build -t api-service:a3f91c2 -t api-service:latest  services/api-service/
docker build -t auth-service:a3f91c2 -t auth-service:latest services/auth-service/
```

Both Docker images are built **at the same time**. Each image gets two tags:
- `api-service:a3f91c2` — immutable, tied to this exact commit (for traceability)
- `api-service:latest` — the "current" pointer that Docker Compose resolves by default

Images are stored locally on the EC2 instance. There is no external registry.

---

### Stage 4 — Deploy

```groovy
set -a
source /home/ec2-user/app.env   # loads POSTGRES_PASSWORD and JWT_SECRET
set +a
IMAGE_TAG=a3f91c2 docker compose -f docker-compose.yml up -d --remove-orphans
```

1. **Sources `app.env`** — the secrets file that lives only on the EC2 instance, created once manually during setup. It is never committed to Git.
2. **`-f docker-compose.yml`** — uses only the production Compose file, skipping `docker-compose.override.yml` (the local dev file with `build:` directives).
3. **`up -d --remove-orphans`** — Docker Compose compares what you want against what is already running and only restarts containers whose image changed. The PostgreSQL container, whose image tag is the fixed `postgres:16-alpine`, is left untouched. **Your data is safe across deployments.**

---

### Post block (always runs)

```groovy
always { sh 'docker image prune -f --filter "until=24h"' }
```

Deletes Docker images older than 24 hours from the EC2 disk. Without this, every push would accumulate a new set of images and eventually fill the disk.

---

### Full trigger-to-live sequence

```
git push origin main
       │
       ▼
GitHub sends webhook POST
→ http://<EC2_IP>:8080/github-webhook/
       │
       ▼
Jenkins pipeline starts
       │
       ├─ [Checkout]      pulls latest code from GitHub
       │
       ├─ [Test]          ┌─ api-service tests  ┐ parallel
       │                  └─ auth-service tests ┘
       │                  ↳ failure here = pipeline aborts, nothing deployed
       │
       ├─ [Build Images]  ┌─ build api-service  ┐ parallel
       │                  └─ build auth-service ┘
       │
       ├─ [Deploy]        docker compose up -d
       │                  ↳ only changed containers restart
       │                  ↳ postgres is untouched, data intact
       │
       └─ [Post]          prune images older than 24h
```

---

## Application Architecture

### Two Services

**auth-service (port 8081)**
Handles user identity. It stores users in PostgreSQL, hashes passwords with bcrypt, and issues signed JWT tokens on login.

**api-service (port 8090)**
Handles task CRUD operations. It does **not** validate JWTs itself — it delegates that to auth-service by making an internal HTTP call on every protected request.

### How a Request Flows Through the System

Here is what happens when a client calls `GET /tasks`:

```
Client
  │
  │  GET /tasks
  │  Authorization: Bearer <jwt>
  ▼
api-service :8090
  │
  │  auth middleware intercepts the request
  │  GET http://auth-service:8081/validate
  │  Authorization: Bearer <jwt>   ← forwards the header
  ▼
auth-service :8081
  │
  │  parses and verifies the JWT signature
  │  returns {"user_id": "abc-123"}  or  401
  ▼
api-service (middleware receives result)
  │
  │  if 401 → immediately return 401 to client
  │  if 200 → inject user_id into request context, continue
  ▼
tasks handler
  │
  │  SELECT * FROM tasks WHERE user_id = 'abc-123'
  ▼
PostgreSQL :5432 (tasks_db)
  │
  ▼
api-service returns JSON task list to client
```

The two services communicate over a private Docker network (`app-net`). PostgreSQL is only reachable inside that network — it is never exposed to the internet.

### Database Layout

One PostgreSQL container holds two separate databases:

| Database   | Used by      | Tables  |
|------------|--------------|---------|
| `auth_db`  | auth-service | `users` |
| `tasks_db` | api-service  | `tasks` |

The databases are created automatically the first time Postgres starts, by SQL scripts mounted into the container from `scripts/db-init/`.

---

## API Reference

### auth-service

| Method | Path      | Auth required | Description                               |
|--------|-----------|---------------|-------------------------------------------|
| POST   | /register | No            | Create account. Body: `{email, password}` |
| POST   | /login    | No            | Returns `{token}`. Body: `{email, password}` |
| GET    | /validate | Bearer JWT    | Returns `{user_id}` if token is valid     |
| GET    | /health   | No            | Returns `{"status":"ok"}`                 |

### api-service

| Method | Path        | Auth required | Description             |
|--------|-------------|---------------|-------------------------|
| GET    | /tasks      | Bearer JWT    | List all tasks for user |
| POST   | /tasks      | Bearer JWT    | Create a task           |
| GET    | /tasks/{id} | Bearer JWT    | Get one task            |
| PUT    | /tasks/{id} | Bearer JWT    | Update a task           |
| DELETE | /tasks/{id} | Bearer JWT    | Delete a task           |
| GET    | /health     | No            | Returns `{"status":"ok"}` |

Task statuses: `pending` → `in_progress` → `done`

---

## Go Code Conventions

- **Go 1.22 stdlib `net/http`** with method-scoped routes (`GET /tasks/{id}`) — no external router needed.
- **Raw `database/sql`** with `lib/pq` driver — no ORM. SQL is written by hand to keep the data layer transparent.
- **Repository interfaces** in each service allow the service layer to be unit tested with testify mocks, without needing a real database.
- **`ErrNotFound` sentinel** in the service layer — handlers translate it to HTTP 404 without leaking internal error details.
- **`PasswordHash` is `json:"-"`** on the User model — it can never accidentally appear in an API response.
