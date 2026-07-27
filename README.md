# Deliver

A simple assignment posting app.

**Deliver** is a 3-tier web application for managing student assignment submissions. Students fork a template repository, complete their work, and submit their GitHub repository URL. Instructors can then review each student's commits and code.

> _A 3-tier web app, HTMX, Go and SQLite3; the point of the app is to encapsulate a workflow of adult students who need to submit GitHub repos for various weekly and periodic projects. The student forks and clones a template of a project, they then submit the URL of their solution to the activity (usually a programming project, sometimes a research project). An instructor can then look at the submissions of each student, to review their commits and code._

## Features

- **Cohorts** – Named groups (e.g. Summer26, Fall26) with a focus area (Java, Data, etc.)
- **Weeks** – Numbered weeks within each cohort
- **Activities** – Programming or research assignments with:
  - Title, description, source/template repo URL
  - Assigned date and due date
  - Individual or group submission mode
- **Submissions** – Students submit their forked repository URL; instructors see all submissions
- **Groups** – For group activities, instructors create groups and students submit as a group
- **Authentication**:
  - Local admin (username + password) for initial setup
  - GitHub OAuth for students and instructors (must be whitelisted by an admin)
- **Roles**: `admin`, `instructor`, `student`

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | [HTMX](https://htmx.org) + Go `html/template` |
| Backend | Go 1.21+, [chi](https://github.com/go-chi/chi) router |
| Auth | [markbates/goth](https://github.com/markbates/goth) (GitHub OAuth) + [gorilla/sessions](https://github.com/gorilla/sessions) |
| Database | SQLite3 via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) |

## Getting Started

### Prerequisites

- Go 1.21+
- A GitHub OAuth App (for student/instructor login)
  - Create one at <https://github.com/settings/developers>
  - Set callback URL to your deployed app URL, for example `https://deliver.zipcode.rocks/auth/github/callback`

### Run

```bash
# Clone and enter the repo
git clone https://github.com/kristofer/delivery.git
cd delivery

# Set required environment variables
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
export SESSION_SECRET=a-long-random-secret
export LOCAL_ADMIN_USERNAME=admin
export LOCAL_ADMIN_PASSWORD=change-this-password

# Optional overrides
export DB_PATH=deliver.db          # default: deliver.db
export ADDR=:8080                  # default: :8080
export APP_URL=http://localhost:8080
export GITHUB_CALLBACK_URL=http://localhost:8080/auth/github/callback

# Build and run
go build -o deliver .
./deliver
```

Then open <http://localhost:8080/login>. If no local admins exist and `LOCAL_ADMIN_USERNAME` + `LOCAL_ADMIN_PASSWORD` are set, the app creates that first administrator automatically. Without those env vars, use <http://localhost:8080/setup> to create the first administrator account manually.

### Run with Docker

```bash
docker build -t deliver .

docker run --rm \
  -p 9209:8080 \
  -e SESSION_SECRET=a-long-random-secret \
  -e LOCAL_ADMIN_USERNAME=admin \
  -e LOCAL_ADMIN_PASSWORD=change-this-password \
  -e GITHUB_CLIENT_ID=your_client_id \
  -e GITHUB_CLIENT_SECRET=your_client_secret \
  -e APP_URL=https://deliver.zipcode.rocks \
  -v deliver-data:/data \
  deliver
```

Or use Docker Compose:

```bash
SESSION_SECRET=a-long-random-secret \
LOCAL_ADMIN_USERNAME=admin \
LOCAL_ADMIN_PASSWORD=change-this-password \
GITHUB_CLIENT_ID=your_client_id \
GITHUB_CLIENT_SECRET=your_client_secret \
APP_URL=https://deliver.zipcode.rocks \
docker compose up -d --build
```

The container stores the SQLite database at `/data/deliver.db` by default, so mount `/data` if you want the data to persist across restarts. The example above publishes the app on host port `9209` while the container continues listening on port `8080`.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `deliver.db` | Path to SQLite database file |
| `ADDR` | `:8080` | Listen address |
| `PORT` | — | Listen port used when `ADDR` is not set (useful for container platforms) |
| `APP_URL` | derived from `ADDR`/`PORT` | Public base URL used for Docker/proxy deployments and the default OAuth callback |
| `SESSION_SECRET` | *(insecure default)* | Cookie signing secret — **change in production** |
| `LOCAL_ADMIN_USERNAME` | — | When set with `LOCAL_ADMIN_PASSWORD`, bootstraps the first local admin account automatically |
| `LOCAL_ADMIN_PASSWORD` | — | Password used for first local admin bootstrap (set a strong secret in production) |
| `GITHUB_CLIENT_ID` | — | GitHub OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth App client secret |
| `GITHUB_CALLBACK_URL` | `APP_URL + /auth/github/callback` | OAuth redirect URL override |
| `SESSION_SECURE` | — | Set to `true` to force the Secure cookie flag (it is also enabled automatically when `APP_URL` uses HTTPS) |

## Workflow

1. **Admin** logs in with bootstrap credentials from `LOCAL_ADMIN_USERNAME`/`LOCAL_ADMIN_PASSWORD`, or visits `/setup` on first run if those env vars are not set
2. **Admin** logs in and goes to `/admin` to whitelist GitHub usernames and assign roles
3. **Instructor** creates cohorts, adds weeks, and adds activities to each week
4. **Instructor** enrolls students in cohorts
5. **Students** log in via GitHub and navigate to their cohort's activities
6. **Students** fork the source template, complete the assignment, and submit their repo URL
7. **Instructor** reviews all submissions from the activity detail page

## Running Tests

```bash
go test ./...
```
