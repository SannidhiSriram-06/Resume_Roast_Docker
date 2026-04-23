# Resume Roaster 🔥

A Go HTTP service that roasts your resume against a job description using Groq's LLaMA model. Upload your PDF, paste the JD, get brutally honest feedback.

## Stack

- **Go** — HTTP server (stdlib only, no frameworks)
- **Groq API** — LLaMA 3.1 8B for the roasting
- **poppler-utils** — PDF text extraction via `pdftotext`
- **Docker** — Multi-stage build (Go builder → Debian slim runtime)

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Web UI |
| GET | `/health` | Health check |
| POST | `/roast` | Upload resume PDF + JD, get roasted |

## Running Locally

### With Docker Compose (recommended)

```bash
# 1. Clone the repo
git clone https://github.com/yourusername/resume-roaster
cd resume-roaster

# 2. Set your Groq API key
cp .env.example .env
# Edit .env and add your GROQ_API_KEY

# 3. Build and run
docker compose up --build

# 4. Open http://localhost:8080
```

### Without Docker

```bash
# Requires Go 1.22+ and poppler-utils installed
export GROQ_API_KEY=your_key_here
go run .
```

## API Usage

```bash
curl -X POST http://localhost:8080/roast \
  -F "resume=@/path/to/resume.pdf" \
  -F "jd=We are looking for a DevOps engineer with 3+ years of Kubernetes experience..."
```

## Project Structure

```
resume-roaster/
├── main.go              # Go HTTP server
├── static/
│   └── index.html       # Web UI
├── Dockerfile           # Multi-stage build
├── docker-compose.yml
├── .env.example
└── README.md
```
