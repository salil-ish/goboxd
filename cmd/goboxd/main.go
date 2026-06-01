package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// --- API Contract Models ---

type RunRequest struct {
	Language         string     `json:"language"`
	Source           string     `json:"source"`
	SourceFilename   string     `json:"source_filename,omitempty"`
	ArtifactFilename string     `json:"artifact_filename,omitempty"`
	Tests            []TestCase `json:"tests"`
}

type TestCase struct {
	Stdin          string `json:"stdin"`
	ExpectedStdout string `json:"expected_stdout"`
}

type RunResponse struct {
	Status string       `json:"status"`
	Build  BuildResult  `json:"build"`
	Tests  []TestResult `json:"tests"`
}

type BuildResult struct {
	Status     string `json:"status"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int    `json:"duration_ms"`
}

type TestResult struct {
	Status       string `json:"status"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	DurationMs   int    `json:"duration_ms"`
	MemoryPeakKb int    `json:"memory_peak_kb"` // Stubbed for Stage 1
}

// --- Handlers ---

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]map[string]string{
			"error": {"code": "bad_request", "message": "Invalid JSON"},
		})
		return
	}

	// 1. Setup secure temp directory
	tmpDir, err := os.MkdirTemp("", "goboxd-*")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)
	
	// Ensure temp dir is accessible by the nsjail dummy user
	os.Chmod(tmpDir, 0777)

	resp := RunResponse{
		Status: "accepted",
		Build:  BuildResult{Status: "ok"},
		Tests:  make([]TestResult, len(req.Tests)),
	}

	// 2. Language Routing & Build Step
	var runCmd []string
	
	if req.Language == "py3" {
		sourcePath := filepath.Join(tmpDir, "solution.py")
		os.WriteFile(sourcePath, []byte(req.Source), 0777)
		runCmd = []string{"/usr/bin/python3", "/app/solution.py"}
		
	} else if req.Language == "cpp" {
		sourcePath := filepath.Join(tmpDir, "solution.cpp")
		artifactPath := filepath.Join(tmpDir, "solution")
		os.WriteFile(sourcePath, []byte(req.Source), 0777)
		
		// Build C++
		start := time.Now()
		buildExec := exec.Command("g++", "-O2", "-o", artifactPath, sourcePath)
		var stderr bytes.Buffer
		buildExec.Stderr = &stderr
		
		if err := buildExec.Run(); err != nil {
			resp.Status = "build_failed"
			resp.Build.Status = "failed"
			resp.Build.Stderr = stderr.String()
			resp.Build.DurationMs = int(time.Since(start).Milliseconds())
			
			// Mark all tests as not_executed
			for i := range resp.Tests {
				resp.Tests[i].Status = "not_executed"
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp.Build.DurationMs = int(time.Since(start).Milliseconds())
		runCmd = []string{"/app/solution"}
		
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]map[string]string{
			"error": {"code": "unknown_language", "message": "Language not supported"},
		})
		return
	}

	// 3. Execution Loop with nsjail
	topLevelStatus := "accepted"
	
	for i, test := range req.Tests {
		start := time.Now()
		
		// nsjail args: mount tmp as /app, run as dummy user, cap resources
		nsjailArgs := append([]string{
			"--quiet", "-Mo", "--chroot", "/", "--bindmount", tmpDir + ":/app",
			"--user", "99999", "--group", "99999", "--cwd", "/app", "--",
		}, runCmd...)

		cmd := exec.Command("nsjail", nsjailArgs...)
		cmd.Stdin = strings.NewReader(test.Stdin)
		
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		duration := int(time.Since(start).Milliseconds())

		actualOutput := strings.TrimSpace(stdout.String())
		expectedOutput := strings.TrimSpace(test.ExpectedStdout)

		testStatus := "accepted"
		if err != nil {
			testStatus = "runtime_error"
		} else if actualOutput != expectedOutput {
			testStatus = "wrong_output"
		}

		if testStatus != "accepted" && topLevelStatus == "accepted" {
			topLevelStatus = testStatus
		}

		resp.Tests[i] = TestResult{
			Status:     testStatus,
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
			DurationMs: duration,
		}
	}

	resp.Status = topLevelStatus
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/run", runHandler)

	log.Println("Starting goboxd on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}