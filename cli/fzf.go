package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runFzf(lines []string) error {
	_, err := exec.LookPath("fzf")
	if err != nil {
		// Fallback: just print the lines.
		for _, l := range lines {
			fmt.Println(strings.ReplaceAll(l, "\t", " | "))
		}
		return nil
	}

	cmd := exec.Command("fzf",
		"--delimiter", "\t",
		"--with-nth", "2,3,4",
		"--preview", "echo {} | cut -f2",
		"--preview-window", "down:wrap",
	)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
