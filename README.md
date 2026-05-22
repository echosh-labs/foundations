# foundations

> **Autonomous Living through Authentic Freedom.**

Foundations is an industrial-grade, multi-pipeline AI orchestration engine and telemetry hub designed to emancipate human consciousness from routine screen-time. It processes voice-ingested intent captured by [Axis Mundi](https://github.com/echosh-labs/axis-mundi) and executes complex digital workflows—from automated code generation and compilation to watercolor storyboard creation and full Google Workspace automation.

---

## 🌟 The Philosophy

In a world saturated with fragmented interfaces and high friction screen interactions, **Foundations** is built to return focus to what matters. The core goal is **authentic freedom through pure automation**: allowing individuals to speak their intents naturally, then step away while a secure, local, multi-threaded Go chassis translates their concepts into production-ready execution.

---

## 🛠️ System Architecture & Core Pipelines

Foundations operates as the execution chassis, receiving triaged directives from **Axis Mundi's SQLite database** and routing them into three high-integrity pipelines:

```
[Voice / Thought] ──> Gemini Mobile Ingestion ──> Axis Mundi Keep Stream
                                                            │
                                                            ▼ (SQLite Triage)
┌────────────────────────────────────────────────────────────────────────┐
│                              FOUNDATIONS                               │
│                                                                        │
│        ┌───────────────────┬───────────────────┬───────────────────┐   │
│        │   CREATIVE_STORY  │      CODE_DEV     │   STRATEGIC_PLAN  │   │
│        │ (Storyteller Core)│ (Compiler Chassis)│ (Workspace Admin) │   │
│        └─────────┬─────────┴─────────┬─────────┴─────────┬─────────┘   │
│                  │                   │                   │             │
│                  ▼                   ▼                   ▼             │
│             Generates Art,       Runs Tests,        Appends Budgets,   │
│             Storyboards, and    Compiles Code,      Drafts Docs, and   │
│             Responsive Pages     Deploys Binaries   Schedules Events   │
└──────────────────────────────────────┬─────────────────────────────────┘
                                       │
                                       ▼ (Unix Domain Socket Stream)
                        Telemetry Hub (SSE on Port 8085)
                                       │
                ┌──────────────────────┴──────────────────────┐
                │                                             │
                ▼                                             ▼
     Responsive Web Console                         Split-Pane Terminal TUI
   (Glassmorphic SSE Dashboard)                    (Lipgloss CLI Interface)
```

### 1. 🎨 The Creative Storyboard Pipeline (`CREATIVE_STORY`)
- **Purpose**: Generates high-fidelity visual storyboards, outlines scene scripts, and compiles narratives from raw speech.
- **Resilience Core**: If external visual generation APIs are rate-limited or unavailable, the system automatically falls back to dynamically composed, HSL-tailored local fallback visuals.
- **Outputs**: Deploys fully responsive, ambient watercolor presentations inside the `web-app/` directory (e.g. showcasing *Intuition, Idealism, and Illumination* motifs).

### 2. 💻 The Engineering Pipeline (`CODE_DEV`)
- **Purpose**: Automates standard software engineering tasks based on developer directives.
- **Actions**: Parses scripts, performs syntax checks, runs Go unit tests, compiles new binaries, and restarts internal services natively.

### 3. 📊 The Strategic Workspace Pipeline (`STRATEGIC_PLAN`)
- **Purpose**: Executes deep Google Workspace automation using GCP Service Accounts with Domain-Wide Delegation.
- **Actions**: Appends rows to financial ledger sheets, drafts outline documents, structures calendar blocks, and queues email threads.

---

## 📡 Telemetry & Log Monitoring

Foundations uses a sub-millisecond logging broker built on top of high-speed UNIX domain sockets and HTTP Server-Sent Events (SSE):

- **Go Telemetry Hub** (`main.go`): Connects to `/tmp/foundations.sock` to collect subsystem log events and broadcasts them via SSE on port `8085`.
- **Dual-Pane TUI Command Center** (`tui-app/`): Built using **Bubble Tea** and **Lipgloss**, this terminal interface splits your viewport into a task registry list on the left and a scrolling telemetry log monitor on the right. Tab switches panel focus.
- **Glassmorphic Web Dashboard** (`web-app/telemetry.html`): A sleek, responsive console showing real-time logs, stabilization progress rings, flux density gauges, and overall subsystem integrity metrics.

---

## 🔌 Integration with Axis Mundi

Foundations is designed to work in tandem with **Axis Mundi**, which serves as the ingestion and triage layer:

1. **Ingestion**: Speak directly to Gemini on your mobile device. Axis Mundi captures the payload in Google Keep and queues it as `Pending` in the SQLite registry database (`axis.db`).
2. **Triage**: Operators review the task inside the Axis Mundi keyboard TUI. Swapping status to `Execute` triggers the Foundations classifier pipeline.
3. **Execution**: Foundations autonomously completes the task, streams telemetry logs, and updates the task status to `Complete`.

---

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- Python 3+
- Linux/WSL environment (for unix socket support)

### Setup & Run
1. Start **Axis Mundi** on port 8080:
   ```bash
   cd /path/to/axis-mundi
   ./axis
   ```
2. Start the **Foundations Telemetry Hub**:
   ```bash
   cd /path/to/foundations
   go run main.go
   ```
3. Open the TUI Dashboard:
   ```bash
   cd tui-app
   go run main.go
   ```
4. Access the Web Telemetry Dashboard by starting the local web server:
   ```bash
   python3 -m http.server 8000 --directory ./web-app
   ```
   Navigate your browser to `http://localhost:8000/telemetry.html`.

---

## 🌐 Links & Licensing

- **Official Website**: [echosh-labs.com/axis-mundi](https://echosh-labs.com/axis-mundi)
- **Primary Ingestion Bridge**: [GitHub - echoSH axis-mundi](https://github.com/echosh-labs/axis-mundi)

### Licensing Model
Use of Foundations is dual-licensed:
1. **Open Source Edition**: Governed by the **AGPL-3.0** license for individual, personal, and educational experimentation.
2. **Commercial Edition**: Requires a valid commercial license key from echoSH labs. The commercial license is required for corporate environments, closed-source integration, and advanced features including:
   - **Domain-Wide Delegation** for whole-domain enterprise impersonation.
   - **AUTO background mode** for non-interactive task routing.
   - **Write-Access Workspace Modules** for programmatic edits to spreadsheets and email threads.

### Licensing & Support Inquiries
For commercial pricing tiers, integration support, or license keys, contact:
- **Maintainer**: Justin Andrew Wood
- **Email**: [justin@echosh-labs.com](mailto:justin@echosh-labs.com)
- **Domain**: [echosh-labs.com](https://echosh-labs.com)
