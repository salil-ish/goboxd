<div align="center">

# goboxd

**A Go HTTP service for executing untrusted code in isolated sandboxes.**

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8.svg?logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-Required-2496ED.svg?logo=docker&logoColor=white)](https://www.docker.com)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/thesouldev/goboxd/pulls)

</div>

---

## Overview

goboxd is an HTTP service written in Go that compiles and runs untrusted code inside isolated sandboxes and returns the result. Optional test cases can be supplied to assert behaviour against expected output. It is built for safe execution of code across many languages, with strict isolation, bounded concurrency, and a plug and play language registry.

## Architecture & Framework Justification

goboxd uses Go's standard `net/http` library for routing. The standard library is sufficient for this project's simple API footprint and prevents the overhead of introducing third-party framework dependencies.

## Features

- Plug and play language registry driven by YAML
- Process isolation using Linux namespaces and cgroups
- Bounded concurrency with request queuing
- Fully containerised for local development and deployment
- Per request resource limits for time, memory, and processes
- Liveness and readiness probes for orchestration

## Getting started

### Prerequisites

- Docker with Compose v2

No Go toolchain or system dependencies are required on the host. Everything runs in containers.

### Installation

```sh
git clone [https://github.com/thesouldev/goboxd.git](https://github.com/thesouldev/goboxd.git)
cd goboxd
make build
