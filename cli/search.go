package cli

import (
	"fmt"
	"strings"
)

func init() {
	register("search", runSearch)
}

func runSearch(args []string) error {
	typeFilter := ""
	var keywords []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				typeFilter = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(args[i], "--type=") {
				typeFilter = strings.TrimPrefix(args[i], "--type=")
			} else {
				keywords = append(keywords, args[i])
			}
		}
	}

	if len(keywords) == 0 {
		return fuzzySearch(typeFilter)
	}

	var count int

	if typeFilter == "" || typeFilter == "word" || strings.HasPrefix("word", typeFilter) {
		words, err := wordStore.Search(keywords, nil)
		if err != nil {
			return err
		}
		for _, w := range words {
			def := ""
			if len(w.Definitions) > 0 {
				def = w.Definitions[0].Meaning
				if len(def) > 50 {
					def = def[:47] + "..."
				}
			}
			fmt.Printf("[W] %-20s  %s  %s\n", w.Word, strings.Join(w.Tags, ","), def)
		}
		count += len(words)
	}

	if typeFilter == "" || typeFilter == "phrase" || strings.HasPrefix("phrase", typeFilter) {
		phrases, err := phraseStore.Search(keywords, nil)
		if err != nil {
			return err
		}
		for _, p := range phrases {
			def := p.Definition
			if len(def) > 50 {
				def = def[:47] + "..."
			}
			fmt.Printf("[P] %-25s  %s  %s\n", p.Phrase, strings.Join(p.Tags, ","), def)
		}
		count += len(phrases)
	}

	if typeFilter == "" || typeFilter == "sent" || typeFilter == "sentence" || strings.HasPrefix("sentence", typeFilter) {
		sentences, err := sentenceStore.Search(keywords, nil)
		if err != nil {
			return err
		}
		for _, s := range sentences {
			text := s.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			fmt.Printf("[S] %-60s  %s\n", text, strings.Join(s.Tags, ","))
		}
		count += len(sentences)
	}

	if typeFilter == "" || typeFilter == "article" || strings.HasPrefix("article", typeFilter) {
		articles, err := articleStore.Search(keywords, nil)
		if err != nil {
			return err
		}
		for _, a := range articles {
			fmt.Printf("[A] %-40s  %s  %s\n", a.Title, a.Source, strings.Join(a.Tags, ","))
		}
		count += len(articles)
	}

	fmt.Printf("\n%d results\n", count)
	return nil
}

func fuzzySearch(typeFilter string) error {
	var items []fuzzyItem

	if typeFilter == "" || typeFilter == "word" {
		words, _ := wordStore.Search(nil, nil)
		for _, w := range words {
			def := ""
			if len(w.Definitions) > 0 {
				def = w.Definitions[0].Meaning
			}
			items = append(items, fuzzyItem{
				Type:   "[W]",
				ID:     w.ID,
				Text:   w.Word,
				Detail: def,
				Tags:   strings.Join(w.Tags, ","),
			})
		}
	}
	if typeFilter == "" || typeFilter == "phrase" {
		phrases, _ := phraseStore.Search(nil, nil)
		for _, p := range phrases {
			items = append(items, fuzzyItem{
				Type:   "[P]",
				ID:     p.ID,
				Text:   p.Phrase,
				Detail: p.Definition,
				Tags:   strings.Join(p.Tags, ","),
			})
		}
	}
	if typeFilter == "" || typeFilter == "sent" || typeFilter == "sentence" {
		sentences, _ := sentenceStore.Search(nil, nil)
		for _, s := range sentences {
			detail := s.Translation
			if detail == "" {
				detail = s.Source
			}
			items = append(items, fuzzyItem{
				Type:   "[S]",
				ID:     s.ID,
				Text:   s.Text,
				Detail: detail,
				Tags:   strings.Join(s.Tags, ","),
			})
		}
	}
	if typeFilter == "" || typeFilter == "article" {
		articles, _ := articleStore.Search(nil, nil)
		for _, a := range articles {
			items = append(items, fuzzyItem{
				Type:   "[A]",
				ID:     a.ID,
				Text:   a.Title,
				Detail: a.Source,
				Tags:   strings.Join(a.Tags, ","),
			})
		}
	}

	if len(items) == 0 {
		fmt.Println("No items.")
		return nil
	}

	selected, err := fuzzyPicker(items)
	if err != nil {
		return err
	}
	if selected == nil {
		return nil
	}

	printDetail(selected.Type, selected.ID)
	return nil
}

func printDetail(typ, id string) {
	fmt.Println()
	switch typ {
	case "[W]":
		wordShow(id)
	case "[P]":
		phraseShow(id)
	case "[S]":
		sentenceShow(id)
	case "[A]":
		articleShow(id)
	}
}
