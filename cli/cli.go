package cli

import (
	"fmt"
	"os"
	"strings"
)

type cmd struct {
	Name string
	Run  func(args []string) error
}

var cmds []cmd

func register(name string, run func(args []string) error) {
	cmds = append(cmds, cmd{Name: name, Run: run})
}

func Run() error {
	if len(os.Args) < 2 {
		usage()
		return nil
	}

	args := os.Args[2:] // args after the subcommand

	// Try full command match first.
	full := os.Args[1]
	for _, c := range cmds {
		if c.Name == full {
			return c.Run(args)
		}
	}

	// Try prefix abbreviation.
	var match *cmd
	for i := range cmds {
		if strings.HasPrefix(cmds[i].Name, full) {
			if match != nil {
				return fmt.Errorf("ambiguous command %q (matches %s, %s)", full, match.Name, cmds[i].Name)
			}
			match = &cmds[i]
		}
	}
	if match != nil {
		return match.Run(args)
	}

	return fmt.Errorf("unknown command: %s", full)
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
