package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CompilePDF runs xelatex on the given .tex content and returns the PDF bytes.
// Returns an error if xelatex is not available or compilation fails.
func CompilePDF(texContent string) ([]byte, error) {
	xelatex, err := exec.LookPath("xelatex")
	if err != nil {
		return nil, fmt.Errorf("xelatex not found: %w", err)
	}

	dir, err := os.MkdirTemp("", "lingo-export-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	texPath := filepath.Join(dir, "document.tex")
	if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
		return nil, fmt.Errorf("write tex file: %w", err)
	}

	// Run xelatex twice (for TOC, cross-refs).
	for i := 0; i < 2; i++ {
		cmd := exec.Command(xelatex,
			"-interaction=nonstopmode",
			"-output-directory="+dir,
			texPath,
		)
		cmd.Dir = dir
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("xelatex compile: %w", err)
		}
	}

	pdfPath := filepath.Join(dir, "document.pdf")
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}

	return pdf, nil
}
