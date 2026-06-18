package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

// latexRequest is the JSON body for POST /api/latex
type latexRequest struct {
	Source string `json:"source"`
}

// latexResponse is the JSON response for the LaTeX compilation endpoint.
type latexResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Output  string `json:"output,omitempty"` // pdflatex output (for debugging)
}

// handleLatexCompile compiles LaTeX source to PDF and serves the result.
// POST /api/latex
func (s *Server) handleLatexCompile(w http.ResponseWriter, r *http.Request) {
	var req latexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		writeJSON(w, http.StatusBadRequest, latexResponse{Success: false, Error: "source is required"})
		return
	}

	// Check if pdflatex is available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, latexResponse{
			Success: false,
			Error:   "pdflatex is not installed. Install a TeX distribution (e.g. brew install --cask mactex on macOS, or sudo apt install texlive on Linux).",
		})
		return
	}

	// Create a temp directory for compilation
	tmpDir, err := os.MkdirTemp("", "ogcode-latex-api-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, latexResponse{Success: false, Error: fmt.Sprintf("create temp dir: %v", err)})
		return
	}
	defer os.RemoveAll(tmpDir)

	// Write the LaTeX source
	texPath := filepath.Join(tmpDir, "input.tex")
	if err := os.WriteFile(texPath, []byte(req.Source), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, latexResponse{Success: false, Error: fmt.Sprintf("write tex file: %v", err)})
		return
	}

	// Run pdflatex (two passes for references, TOC, etc.)
	var compileOutput string
	for pass := 1; pass <= 2; pass++ {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		cmd := exec.CommandContext(ctx, "pdflatex",
			"-interaction=nonstopmode",
			"-halt-on-error",
			"-output-directory="+tmpDir,
			texPath,
		)
		cmd.Dir = tmpDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		cancel()

		if err != nil {
			fullOutput := stdout.String() + stderr.String()
			if ctx.Err() == context.DeadlineExceeded {
				writeJSON(w, http.StatusGatewayTimeout, latexResponse{
					Success: false,
					Error:   "pdflatex compilation timed out after 60 seconds.",
				})
				return
			}
			writeJSON(w, http.StatusUnprocessableEntity, latexResponse{
				Success: false,
				Error:   fmt.Sprintf("pdflatex compilation failed (pass %d)", pass),
				Output:  fullOutput,
			})
			return
		}
		compileOutput = stdout.String()
	}

	// Check the PDF was generated
	pdfPath := filepath.Join(tmpDir, "input.pdf")
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, latexResponse{
			Success: false,
			Error:   "pdflatex completed but no PDF was generated",
			Output:  compileOutput,
		})
		return
	}

	// Serve the PDF directly
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="output.pdf"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
	w.Write(pdfData)
}

// handleLatexStatus returns whether pdflatex is available on the system.
// GET /api/latex/status
func (s *Server) handleLatexStatus(w http.ResponseWriter, r *http.Request) {
	_, err := exec.LookPath("pdflatex")
	available := err == nil
	writeJSON(w, http.StatusOK, map[string]bool{
		"available": available,
	})
}

func (s *Server) registerLatexRoutes(r chi.Router) {
	r.Post("/latex", s.handleLatexCompile)
	r.Get("/latex/status", s.handleLatexStatus)
}