package cli

import (
	"fmt"
	"os"
	"strings"

	"lingo/store"
)

// Stores holds references to all data stores and web starter.
type Stores struct {
	Words     store.WordStore
	Phrases   store.PhraseStore
	Sentences store.SentenceStore
	Articles  store.ArticleStore
	Tags      store.TagStore
	StartWeb  func(port string) // called by lingo web
}

type cmd struct {
	Name string
	Run  func(args []string) error
}

var cmds []cmd

func register(name string, run func(args []string) error) {
	cmds = append(cmds, cmd{Name: name, Run: run})
}

func Run(stores Stores) {
	if len(os.Args) < 2 {
		usage()
		return
	}

	args := os.Args[2:] // args after the subcommand

	// Try full command match first.
	full := os.Args[1]
	for _, c := range cmds {
		if c.Name == full {
			if err := c.Run(args); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		}
	}

	// Try prefix abbreviation.
	var match *cmd
	for i := range cmds {
		if strings.HasPrefix(cmds[i].Name, full) {
			if match != nil {
				fmt.Fprintf(os.Stderr, "ambiguous command %q (matches %s, %s)\n", full, match.Name, cmds[i].Name)
				os.Exit(1)
			}
			match = &cmds[i]
		}
	}
	if match != nil {
		if err := match.Run(args); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command: %s\n", full)
	usage()
	os.Exit(1)
}

func usage() {
	fmt.Println("Usage: lingo <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, c := range cmds {
		fmt.Printf("  %s\n", c.Name)
	}
	fmt.Println()
	fmt.Println("All commands support prefix abbreviation (e.g., 'lingo w' = 'lingo word')")
}
