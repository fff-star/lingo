package cli

import (
	"fmt"
	"strings"

	"lingo/model"
	"lingo/store"
)

var tagStore store.TagStore

func InitTag(s store.TagStore) {
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
	case "assign":
		return tagBatchAssign(args[1:])
	case "unassign":
		return tagBatchUnassign(args[1:])
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

// tagBatchAssign assigns a tag to multiple items at once.
// Usage: lingo tag assign <tagname> --type word|phrase|sent|article|comp [--ids id1,id2] [--all] [--filter-tag <tag>]
func tagBatchAssign(args []string) error {
	return tagBatchOp(args, true)
}

// tagBatchUnassign removes a tag from multiple items at once.
func tagBatchUnassign(args []string) error {
	return tagBatchOp(args, false)
}

func tagBatchOp(args []string, assign bool) error {
	if len(args) == 0 {
		action := "unassign"
		if assign {
			action = "assign"
		}
		return fmt.Errorf("usage: lingo tag %s <tagname> --type <type> [--ids id1,id2] [--all] [--filter-tag <tag>]", action)
	}

	tagName := args[0]
	args = args[1:]

	var typ string
	var ids []string
	var all bool
	var filterTag string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				typ = args[i+1]
				i++
			}
		case "--ids":
			if i+1 < len(args) {
				for _, id := range strings.Split(args[i+1], ",") {
					id = strings.TrimSpace(id)
					if id != "" {
						ids = append(ids, id)
					}
				}
				i++
			}
		case "--all":
			all = true
		case "--filter-tag":
			if i+1 < len(args) {
				filterTag = args[i+1]
				i++
			}
		}
	}

	if typ == "" {
		return fmt.Errorf("--type is required (word, phrase, sent, article, comp)")
	}
	if !all && len(ids) == 0 {
		return fmt.Errorf("use --all or specify --ids")
	}

	switch strings.ToLower(typ) {
	case "word", "words":
		return batchTagWords(tagName, ids, all, filterTag, assign)
	case "phrase", "phrases":
		return batchTagPhrases(tagName, ids, all, filterTag, assign)
	case "sent", "sentence", "sentences":
		return batchTagSentences(tagName, ids, all, filterTag, assign)
	case "article", "articles":
		return batchTagArticles(tagName, ids, all, filterTag, assign)
	case "comp", "composition", "comps", "compositions":
		return batchTagCompositions(tagName, ids, all, filterTag, assign)
	default:
		return fmt.Errorf("unknown type: %s", typ)
	}
}


func batchTagWords(tagName string, ids []string, all bool, filterTag string, assign bool) error {
	words, err := wordStore.Load()
	if err != nil {
		return err
	}
	count := 0
	for i := range words {
		w := &words[i]
		if !all {
			if !store.HasString(ids, w.ID) {
				continue
			}
		}
		if filterTag != "" && !store.HasString(w.Tags, filterTag) {
			continue
		}
		if assign {
			if store.HasString(w.Tags, tagName) {
				continue
			}
			w.Tags = append(w.Tags, tagName)
		} else {
			if !store.HasString(w.Tags, tagName) {
				continue
			}
			w.Tags = store.RemoveString(w.Tags, tagName)
		}
		if err := wordStore.Update(*w); err != nil {
			return err
		}
		count++
	}
	action := "removed from"
		if assign {
			action = "assigned to"
		}
	fmt.Printf("Tag \"%s\" %s %d word(s).\n", tagName, action, count)
	return nil
}

func batchTagPhrases(tagName string, ids []string, all bool, filterTag string, assign bool) error {
	phrases, err := phraseStore.Load()
	if err != nil {
		return err
	}
	count := 0
	for i := range phrases {
		p := &phrases[i]
		if !all {
			if !store.HasString(ids, p.ID) {
				continue
			}
		}
		if filterTag != "" && !store.HasString(p.Tags, filterTag) {
			continue
		}
		if assign {
			if store.HasString(p.Tags, tagName) {
				continue
			}
			p.Tags = append(p.Tags, tagName)
		} else {
			if !store.HasString(p.Tags, tagName) {
				continue
			}
			p.Tags = store.RemoveString(p.Tags, tagName)
		}
		if err := phraseStore.Update(*p); err != nil {
			return err
		}
		count++
	}
	action := "removed from"
		if assign {
			action = "assigned to"
		}
	fmt.Printf("Tag \"%s\" %s %d phrase(s).\n", tagName, action, count)
	return nil
}

func batchTagSentences(tagName string, ids []string, all bool, filterTag string, assign bool) error {
	sentences, err := sentenceStore.Load()
	if err != nil {
		return err
	}
	count := 0
	for i := range sentences {
		st := &sentences[i]
		if !all {
			if !store.HasString(ids, st.ID) {
				continue
			}
		}
		if filterTag != "" && !store.HasString(st.Tags, filterTag) {
			continue
		}
		if assign {
			if store.HasString(st.Tags, tagName) {
				continue
			}
			st.Tags = append(st.Tags, tagName)
		} else {
			if !store.HasString(st.Tags, tagName) {
				continue
			}
			st.Tags = store.RemoveString(st.Tags, tagName)
		}
		if err := sentenceStore.Update(*st); err != nil {
			return err
		}
		count++
	}
	action := "removed from"
		if assign {
			action = "assigned to"
		}
	fmt.Printf("Tag \"%s\" %s %d sentence(s).\n", tagName, action, count)
	return nil
}

func batchTagArticles(tagName string, ids []string, all bool, filterTag string, assign bool) error {
	articles, err := articleStore.Load()
	if err != nil {
		return err
	}
	count := 0
	for i := range articles {
		a := &articles[i]
		if !all {
			if !store.HasString(ids, a.ID) {
				continue
			}
		}
		if filterTag != "" && !store.HasString(a.Tags, filterTag) {
			continue
		}
		if assign {
			if store.HasString(a.Tags, tagName) {
				continue
			}
			a.Tags = append(a.Tags, tagName)
		} else {
			if !store.HasString(a.Tags, tagName) {
				continue
			}
			a.Tags = store.RemoveString(a.Tags, tagName)
		}
		if err := articleStore.Update(*a); err != nil {
			return err
		}
		count++
	}
	action := "removed from"
		if assign {
			action = "assigned to"
		}
	fmt.Printf("Tag \"%s\" %s %d article(s).\n", tagName, action, count)
	return nil
}

func batchTagCompositions(tagName string, ids []string, all bool, filterTag string, assign bool) error {
	comps, err := compStore.Load()
	if err != nil {
		return err
	}
	count := 0
	for i := range comps {
		c := &comps[i]
		if !all {
			if !store.HasString(ids, c.ID) {
				continue
			}
		}
		if filterTag != "" && !store.HasString(c.Tags, filterTag) {
			continue
		}
		if assign {
			if store.HasString(c.Tags, tagName) {
				continue
			}
			c.Tags = append(c.Tags, tagName)
		} else {
			if !store.HasString(c.Tags, tagName) {
				continue
			}
			c.Tags = store.RemoveString(c.Tags, tagName)
		}
		if err := compStore.Update(*c); err != nil {
			return err
		}
		count++
	}
	action := "removed from"
		if assign {
			action = "assigned to"
		}
	fmt.Printf("Tag \"%s\" %s %d composition(s).\n", tagName, action, count)
	return nil
}
