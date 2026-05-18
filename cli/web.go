package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

var webStart func(port string)

func InitWeb(startFn func(port string)) {
	webStart = startFn
}

func init() {
	register("web", runWeb)
	register("stats", runStats)
}

func runWeb(args []string) error {
	port := "8080"
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			port = args[i+1]
			i++
		} else if args[i] == "-p" && i+1 < len(args) {
			port = args[i+1]
			i++
		}
	}

	url := "http://localhost:" + port
	fmt.Printf("Starting web server at %s\n", url)
	fmt.Println("Press Ctrl+C to stop.")

	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "darwin":
			cmd = exec.Command("open", url)
		}
		if cmd != nil {
			cmd.Start()
		}
	}()

	if webStart != nil {
		webStart(port)
	}
	return nil
}

func runStats(args []string) error {
	wordCount, _ := wordStore.Count()
	phraseCount, _ := phraseStore.Count()
	sentenceCount, _ := sentenceStore.Count()
	articleCount, _ := articleStore.Count()
	tagCount := 0
	if tags, err := tagStore.Load(); err == nil {
		tagCount = len(tags)
	}

	fmt.Println("═══════ Library Stats ═══════")
	fmt.Printf("  Words:     %d\n", wordCount)
	fmt.Printf("  Phrases:   %d\n", phraseCount)
	fmt.Printf("  Sentences: %d\n", sentenceCount)
	fmt.Printf("  Articles:  %d\n", articleCount)
	fmt.Printf("  Tags:      %d\n", tagCount)
	fmt.Println("──────────────────────────────")
	fmt.Printf("  Total:     %d\n", wordCount+phraseCount+sentenceCount+articleCount)
	return nil
}
