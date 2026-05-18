package cli

import (
	"fmt"
	"os"
	"strings"

	"lingo/model"
)

// Stores holds references to all data stores and web starter.
type Stores struct {
	Words     WordOperations
	Phrases   PhraseOperations
	Sentences SentenceOperations
	Articles  ArticleOperations
	Tags      TagOperations
	StartWeb  func(port string) // called by sd web
}

// WordOperations is the interface for word store operations needed by CLI.
type WordOperations interface {
	Add(model.Word) error
	Get(idPrefix string) (*model.Word, error)
	Update(model.Word) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Word, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	AllIDs() (map[string]string, error)
	Load() ([]model.Word, error)
	Save([]model.Word) error
}

// PhraseOperations is the interface for phrase store operations needed by CLI.
type PhraseOperations interface {
	Add(model.Phrase) error
	Get(idPrefix string) (*model.Phrase, error)
	Update(model.Phrase) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Phrase, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Phrase, error)
}

// SentenceOperations is the interface for sentence store operations needed by CLI.
type SentenceOperations interface {
	Add(model.Sentence) error
	Get(idPrefix string) (*model.Sentence, error)
	Update(model.Sentence) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Sentence, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Sentence, error)
}

// ArticleOperations is the interface for article store operations needed by CLI.
type ArticleOperations interface {
	Add(model.Article) error
	Get(idPrefix string) (*model.Article, error)
	Update(model.Article) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Article, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Article, error)
}

// TagOperations is the interface for tag store operations needed by CLI.
type TagOperations interface {
	Add(model.Tag) error
	Get(name string) (*model.Tag, error)
	Delete(name string) error
	Rename(oldName, newName string) error
	Load() ([]model.Tag, error)
	Save([]model.Tag) error
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
