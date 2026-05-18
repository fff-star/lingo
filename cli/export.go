package cli

import (
	"fmt"
	"strings"
)

func init() {
	register("export", runExport)
	register("list", runList)
	register("ls", runList)
}

func runExport(args []string) error {
	typeFilter := ""
	format := "text"
	tagFilter := ""

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--type" && i+1 < len(args):
			typeFilter = args[i+1]; i++
		case strings.HasPrefix(args[i], "--type="):
			typeFilter = strings.TrimPrefix(args[i], "--type=")
		case args[i] == "--format" && i+1 < len(args):
			format = args[i+1]; i++
		case strings.HasPrefix(args[i], "--format="):
			format = strings.TrimPrefix(args[i], "--format=")
		case args[i] == "--tag" && i+1 < len(args):
			tagFilter = args[i+1]; i++
		case strings.HasPrefix(args[i], "--tag="):
			tagFilter = strings.TrimPrefix(args[i], "--tag=")
		}
	}

	var tags []string
	if tagFilter != "" {
		tags = strings.Split(tagFilter, ",")
	}

	types := strings.Split(typeFilter, ",")
	if typeFilter == "" {
		types = []string{"word", "phrase", "sent", "article"}
	}

	if format == "json" {
		return exportJSON(types, tags)
	}
	if format == "csv" {
		return exportCSV(types, tags)
	}
	return exportText(types, tags)
}

func exportText(types []string, tags []string) error {
	for _, t := range types {
		switch strings.TrimSpace(t) {
		case "word":
			words, err := wordStore.Search(nil, tags)
			if err != nil {
				return err
			}
			for _, w := range words {
				def := ""
				if len(w.Definitions) > 0 {
					def = w.Definitions[0].Meaning
				}
				fmt.Printf("%s\t%s\t%s\n", w.Word, strings.Join(w.Tags, ","), def)
			}
		case "phrase":
			phrases, err := phraseStore.Search(nil, tags)
			if err != nil {
				return err
			}
			for _, p := range phrases {
				fmt.Printf("%s\t%s\t%s\n", p.Phrase, strings.Join(p.Tags, ","), p.Definition)
			}
		case "sent", "sentence":
			sentences, err := sentenceStore.Search(nil, tags)
			if err != nil {
				return err
			}
			for _, s := range sentences {
				fmt.Printf("%s\t%s\t%s\n", s.Text, strings.Join(s.Tags, ","), s.Translation)
			}
		case "article":
			articles, err := articleStore.Search(nil, tags)
			if err != nil {
				return err
			}
			for _, a := range articles {
				fmt.Printf("%s\t%s\t%s\n", a.Title, a.Source, strings.Join(a.Tags, ","))
			}
		}
	}
	return nil
}

func exportJSON(types []string, tags []string) error {
	fmt.Println("{")
	firstType := true
	for _, t := range types {
		switch strings.TrimSpace(t) {
		case "word":
			words, _ := wordStore.Search(nil, tags)
			if !firstType {
				fmt.Println(",")
			}
			firstType = false
			fmt.Printf("  \"words\": ")
			data, _ := prettyJSON(words)
			fmt.Print(string(data))
		case "phrase":
			phrases, _ := phraseStore.Search(nil, tags)
			if !firstType {
				fmt.Println(",")
			}
			firstType = false
			fmt.Printf("  \"phrases\": ")
			data, _ := prettyJSON(phrases)
			fmt.Print(string(data))
		case "sent", "sentence":
			sentences, _ := sentenceStore.Search(nil, tags)
			if !firstType {
				fmt.Println(",")
			}
			firstType = false
			fmt.Printf("  \"sentences\": ")
			data, _ := prettyJSON(sentences)
			fmt.Print(string(data))
		case "article":
			articles, _ := articleStore.Search(nil, tags)
			if !firstType {
				fmt.Println(",")
			}
			firstType = false
			fmt.Printf("  \"articles\": ")
			data, _ := prettyJSON(articles)
			fmt.Print(string(data))
		}
	}
	fmt.Println("\n}")
	return nil
}

func exportCSV(types []string, tags []string) error {
	fmt.Println("type,text,definition,tags")
	for _, t := range types {
		switch strings.TrimSpace(t) {
		case "word":
			words, _ := wordStore.Search(nil, tags)
			for _, w := range words {
				def := ""
				if len(w.Definitions) > 0 {
					def = w.Definitions[0].Meaning
				}
				fmt.Printf("word,%s,%s,%s\n", escapeCSV(w.Word), escapeCSV(def), escapeCSV(strings.Join(w.Tags, " ")))
			}
		case "phrase":
			phrases, _ := phraseStore.Search(nil, tags)
			for _, p := range phrases {
				fmt.Printf("phrase,%s,%s,%s\n", escapeCSV(p.Phrase), escapeCSV(p.Definition), escapeCSV(strings.Join(p.Tags, " ")))
			}
		case "sent", "sentence":
			sentences, _ := sentenceStore.Search(nil, tags)
			for _, s := range sentences {
				fmt.Printf("sentence,%s,%s,%s\n", escapeCSV(s.Text), escapeCSV(s.Translation), escapeCSV(strings.Join(s.Tags, " ")))
			}
		case "article":
			articles, _ := articleStore.Search(nil, tags)
			for _, a := range articles {
				fmt.Printf("article,%s,%s,%s\n", escapeCSV(a.Title), escapeCSV(a.Source), escapeCSV(strings.Join(a.Tags, " ")))
			}
		}
	}
	return nil
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func runList(args []string) error {
	typeFilter := ""
	tagFilter := ""

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--type" && i+1 < len(args):
			typeFilter = args[i+1]; i++
		case strings.HasPrefix(args[i], "--type="):
			typeFilter = strings.TrimPrefix(args[i], "--type=")
		case args[i] == "--tag" && i+1 < len(args):
			tagFilter = args[i+1]; i++
		case strings.HasPrefix(args[i], "--tag="):
			tagFilter = strings.TrimPrefix(args[i], "--tag=")
		}
	}

	var tags []string
	if tagFilter != "" {
		tags = strings.Split(tagFilter, ",")
	}

	if typeFilter == "" || typeFilter == "word" || strings.HasPrefix("word", typeFilter) {
		words, _ := wordStore.Search(nil, tags)
		for _, w := range words {
			def := ""
			if len(w.Definitions) > 0 {
				def = w.Definitions[0].Meaning
				if len(def) > 60 {
					def = def[:57] + "..."
				}
			}
			fmt.Printf("[W] %-20s  %s  %s\n", w.Word, strings.Join(w.Tags, ","), def)
		}
	}

	if typeFilter == "" || typeFilter == "phrase" || strings.HasPrefix("phrase", typeFilter) {
		phrases, _ := phraseStore.Search(nil, tags)
		for _, p := range phrases {
			def := p.Definition
			if len(def) > 60 {
				def = def[:57] + "..."
			}
			fmt.Printf("[P] %-25s  %s  %s\n", p.Phrase, strings.Join(p.Tags, ","), def)
		}
	}

	if typeFilter == "" || typeFilter == "sent" || typeFilter == "sentence" || strings.HasPrefix("sentence", typeFilter) {
		sentences, _ := sentenceStore.Search(nil, tags)
		for _, s := range sentences {
			text := s.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			fmt.Printf("[S] %-60s  %s\n", text, strings.Join(s.Tags, ","))
		}
	}

	if typeFilter == "" || typeFilter == "article" || strings.HasPrefix("article", typeFilter) {
		articles, _ := articleStore.Search(nil, tags)
		for _, a := range articles {
			fmt.Printf("[A] %-40s  %s  %s\n", a.Title, a.Source, strings.Join(a.Tags, ","))
		}
	}

	return nil
}
