package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"lingo/llm"
	"lingo/model"
	"lingo/store"
)

var articleStore store.ArticleStore

func InitArticle(as store.ArticleStore) {
	articleStore = as
}

func init() {
	register("article", runArticle)
	register("articles", runArticle)
}

func runArticle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo article <id|subcommand>")
	}

	first := args[0]
	switch first {
	case "add":
		return articleAdd(args[1:])
	case "list", "ls":
		return articleList(args[1:])
	case "process":
		return articleProcess(args[1:])
	}

	var deleteFlag bool
	for _, a := range args[1:] {
		if a == "--delete" || a == "-d" {
			deleteFlag = true
		}
	}

	if deleteFlag {
		return articleDelete(first)
	}
	return articleShow(first)
}

func articleAdd(args []string) error {
	fromFile := false
	doProcess := false
	var filePath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 < len(args) {
				fromFile = true
				filePath = args[i+1]
				i++
			}
		case "--process", "-p":
			doProcess = true
		}
	}

	var id string
	if fromFile {
		a, err := articleAddFromFile(filePath)
		if err != nil {
			return err
		}
		id = a.ID
	} else {
		a, err := articleAddInteractive()
		if err != nil {
			return err
		}
		if a == nil {
			return nil // cancelled
		}
		id = a.ID
	}

	if doProcess {
		return articleProcess([]string{id})
	}
	return nil
}

func articleAddInteractive() (*model.Article, error) {
	now := time.Now().UTC()
	a := model.Article{
		ID:        store.NewID("ar"),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Title: ")
	if scanner.Scan() {
		a.Title = strings.TrimSpace(scanner.Text())
	}
	if a.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	fmt.Print("Author (optional): ")
	if scanner.Scan() {
		a.Author = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Source (optional): ")
	if scanner.Scan() {
		a.Source = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Source URL (optional): ")
	if scanner.Scan() {
		a.SourceURL = strings.TrimSpace(scanner.Text())
	}

	fmt.Println("Content (enter a single '.' on a line to finish):")
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "." {
			break
		}
		lines = append(lines, line)
	}
	a.Content = strings.Join(lines, "\n")

	fmt.Print("Tags (comma-separated): ")
	if scanner.Scan() {
		tags := strings.TrimSpace(scanner.Text())
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				a.Tags = append(a.Tags, strings.TrimSpace(t))
			}
		}
	}

	fmt.Print("Notes: ")
	if scanner.Scan() {
		a.Notes = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("\nSave? [Y/n]: ")
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil, nil
		}
	}

	if err := articleStore.Add(a); err != nil {
		return nil, err
	}
	return &a, nil
}

func articleAddFromFile(path string) (*model.Article, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var a model.Article
	a.ID = store.NewID("ar")
	a.Tags = []string{}
	a.CreatedAt = now
	a.UpdatedAt = now

	// First line as title.
	if scanner.Scan() {
		a.Title = strings.TrimSpace(scanner.Text())
	}

	// Rest as content.
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	a.Content = strings.Join(lines, "\n")

	if err := articleStore.Add(a); err != nil {
		return nil, err
	}
	return &a, nil
}

func articleShow(idPrefix string) error {
	a, err := articleStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Print(formatArticle(a))
	return nil
}

func articleDelete(idPrefix string) error {
	a, err := articleStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Printf("Delete '%s' (%s)? [y/N]: ", a.Title, a.ID)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return articleStore.Delete(idPrefix)
}

func articleList(args []string) error {
	articles, err := articleStore.Search(nil, nil)
	if err != nil {
		return err
	}

	for _, a := range articles {
		fmt.Printf("%s  %-40s  %s  %s\n", a.ID, a.Title, a.Source, strings.Join(a.Tags, ","))
	}
	fmt.Printf("\n%d articles\n", len(articles))
	return nil
}

func formatArticle(a *model.Article) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", a.ID))
	b.WriteString(fmt.Sprintf("Title: %s\n", a.Title))
	if a.Author != "" {
		b.WriteString(fmt.Sprintf("Author: %s\n", a.Author))
	}
	if a.Source != "" {
		b.WriteString(fmt.Sprintf("Source: %s", a.Source))
		if a.SourceURL != "" {
			b.WriteString(fmt.Sprintf(" (%s)", a.SourceURL))
		}
		b.WriteString("\n")
	}
	if len(a.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(a.Tags, ", ")))
	}
	if a.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", a.Notes))
	}
	b.WriteString(fmt.Sprintf("\n%s\n", a.Content))
	b.WriteString(fmt.Sprintf("\nCreated: %s\n", a.CreatedAt.Format("2006-01-02 15:04")))
	return b.String()
}

func articleProcess(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo article process <article-id>")
	}

	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		return err
	}

	a, err := articleStore.Get(args[0])
	if err != nil {
		return err
	}

	if a.Content == "" {
		return fmt.Errorf("article has no content")
	}

	fmt.Printf("Processing: %s\n", a.Title)
	fmt.Printf("Content length: %d chars\n", len([]rune(a.Content)))
	fmt.Println("Sending to DeepSeek...")

	items, err := llm.ProcessArticle(cfg, a.Content, a.Title)
	if err != nil {
		return fmt.Errorf("LLM processing failed: %w", err)
	}

	// Show summary.
	fmt.Printf("\n=== Summary ===\n%s\n", items.Summary)
	if len(items.SuggestedTags) > 0 {
		fmt.Printf("\nSuggested tags: %s\n", strings.Join(items.SuggestedTags, ", "))
	}

	// Show extracted words.
	if len(items.Words) > 0 {
		fmt.Printf("\n=== Words (%d) ===\n", len(items.Words))
		for i, w := range items.Words {
			fmt.Printf("  [W%d] %s", i+1, w.Word)
			if len(w.Definitions) > 0 {
				fmt.Printf("  %s %s", w.Definitions[0].Pos, w.Definitions[0].Meaning)
			}
			fmt.Println()
			if w.Example != "" {
				fmt.Printf("       ex: %s\n", truncate(w.Example, 100))
			}
		}
	} else {
		fmt.Println("\n(no words extracted)")
	}

	// Show extracted phrases.
	if len(items.Phrases) > 0 {
		fmt.Printf("\n=== Phrases (%d) ===\n", len(items.Phrases))
		for i, p := range items.Phrases {
			fmt.Printf("  [P%d] %s  [%s]\n", i+1, p.Phrase, p.Type)
			fmt.Printf("       %s\n", p.Definition)
			if p.Example != "" {
				fmt.Printf("       ex: %s\n", truncate(p.Example, 100))
			}
		}
	} else {
		fmt.Println("\n(no phrases extracted)")
	}

	// Show extracted sentences.
	if len(items.Sentences) > 0 {
		fmt.Printf("\n=== Sentences (%d) ===\n", len(items.Sentences))
		for i, s := range items.Sentences {
			fmt.Printf("  [S%d] %s\n", i+1, truncate(s.Text, 120))
			if s.Translation != "" {
				fmt.Printf("       → %s\n", s.Translation)
			}
			if s.Why != "" {
				fmt.Printf("       why: %s\n", s.Why)
			}
		}
	} else {
		fmt.Println("\n(no sentences extracted)")
	}

	// Confirm and save analysis inline.
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\nSave analysis to article? [Y/n]: ")
	if !scanner.Scan() {
		return nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer == "n" || answer == "no" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Store analysis inline — no additions to word/phrase/sentence stores.
	a.AIAnalysis = items.ToAIAnalysis()

	if len(items.SuggestedTags) > 0 {
		existing := make(map[string]bool)
		for _, t := range a.Tags {
			existing[t] = true
		}
		for _, t := range items.SuggestedTags {
			if !existing[t] {
				a.Tags = append(a.Tags, t)
				existing[t] = true
			}
		}
	}
	a.UpdatedAt = time.Now().UTC()
	if err := articleStore.Update(*a); err != nil {
		return fmt.Errorf("update article: %w", err)
	}

	fmt.Println("Analysis saved.")
	return nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
