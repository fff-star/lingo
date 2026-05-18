package cli

import (
	"fmt"
	"strings"

	"lingo/model"
	"lingo/store"
)

var tagStore *store.TagStore

func InitTag(s *store.TagStore) {
	tagStore = s
}

func init() {
	register("tag", runTag)
	register("tags", runTag)
}

func runTag(args []string) error {
	if len(args) == 0 {
		return tagList()
	}

	switch args[0] {
	case "list", "ls":
		return tagList()
	case "add":
		return tagAdd(args[1:])
	case "rm", "remove", "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: lingo tag rm <name>")
		}
		return tagDelete(args[1])
	case "rename":
		if len(args) < 3 {
			return fmt.Errorf("usage: lingo tag rename <old> <new>")
		}
		return tagStore.Rename(args[1], args[2])
	default:
		return tagList()
	}
}

func tagList() error {
	tags, err := tagStore.Load()
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		fmt.Println("No tags defined.")
		return nil
	}
	for _, t := range tags {
		fmt.Printf("  %s  %-20s  %s  %s\n", t.ID, t.Name, t.Category, t.Description)
	}
	return nil
}

func tagAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo tag add <name> [--color #hex]")
	}

	name := args[0]
	color := "#888888"
	for i := 1; i < len(args); i++ {
		if args[i] == "--color" && i+1 < len(args) {
			color = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--color=") {
			color = strings.TrimPrefix(args[i], "--color=")
		}
	}

	tag := model.Tag{
		ID:    store.NewID("tag"),
		Name:  name,
		Color: color,
	}
	return tagStore.Add(tag)
}

func tagDelete(name string) error {
	return tagStore.Delete(name)
}
