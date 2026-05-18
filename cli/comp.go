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

var compStore *store.CompositionStore

func InitComp(cs *store.CompositionStore) {
	compStore = cs
}

func init() {
	register("comp", runComp)
	register("composition", runComp)
	register("comps", runComp)
	register("compositions", runComp)
}

func runComp(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo comp <id|subcommand>")
	}

	first := args[0]
	switch first {
	case "add":
		return compAdd(args[1:])
	case "list", "ls":
		return compList(args[1:])
	case "analyze", "anal", "process":
		return compAnalyze(args[1:])
	}

	var deleteFlag bool
	for _, a := range args[1:] {
		if a == "--delete" || a == "-d" {
			deleteFlag = true
		}
	}

	if deleteFlag {
		return compDelete(first)
	}
	return compShow(first)
}

func compAdd(args []string) error {
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
		case "--process", "-p", "--analyze":
			doProcess = true
		}
	}

	var id string
	if fromFile {
		c, err := compAddFromFile(filePath)
		if err != nil {
			return err
		}
		id = c.ID
	} else {
		c, err := compAddInteractive()
		if err != nil {
			return err
		}
		if c == nil {
			return nil // cancelled
		}
		id = c.ID
	}

	if doProcess {
		return compAnalyze([]string{id})
	}
	return nil
}

func compAddInteractive() (*model.Composition, error) {
	now := time.Now().UTC()
	c := model.Composition{
		ID:        store.NewID("cp"),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Title: ")
	if scanner.Scan() {
		c.Title = strings.TrimSpace(scanner.Text())
	}
	if c.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	fmt.Print("Author (optional): ")
	if scanner.Scan() {
		c.Author = strings.TrimSpace(scanner.Text())
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
	c.Content = strings.Join(lines, "\n")

	fmt.Print("Tags (comma-separated): ")
	if scanner.Scan() {
		tags := strings.TrimSpace(scanner.Text())
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				c.Tags = append(c.Tags, strings.TrimSpace(t))
			}
		}
	}

	fmt.Print("Notes: ")
	if scanner.Scan() {
		c.Notes = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("\nSave? [Y/n]: ")
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil, nil
		}
	}

	if err := compStore.Add(c); err != nil {
		return nil, err
	}
	return &c, nil
}

func compAddFromFile(path string) (*model.Composition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var c model.Composition
	c.ID = store.NewID("cp")
	c.Tags = []string{}
	c.CreatedAt = now
	c.UpdatedAt = now

	// First line as title.
	if scanner.Scan() {
		c.Title = strings.TrimSpace(scanner.Text())
	}

	// Rest as content.
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	c.Content = strings.Join(lines, "\n")

	if err := compStore.Add(c); err != nil {
		return nil, err
	}
	return &c, nil
}

func compShow(idPrefix string) error {
	c, err := compStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Print(formatComp(c))
	return nil
}

func compDelete(idPrefix string) error {
	c, err := compStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Printf("Delete '%s' (%s)? [y/N]: ", c.Title, c.ID)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return compStore.Delete(idPrefix)
}

func compList(args []string) error {
	comps, err := compStore.Search(nil, nil)
	if err != nil {
		return err
	}

	for _, c := range comps {
		analyzed := ""
		if c.AIAnalysis != nil {
			analyzed = " [analyzed]"
		}
		fmt.Printf("%s  %-40s  %s  %s%s\n", c.ID, c.Title, c.Author, strings.Join(c.Tags, ","), analyzed)
	}
	fmt.Printf("\n%d compositions\n", len(comps))
	return nil
}

func compAnalyze(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo comp analyze <id>")
	}

	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		return err
	}

	c, err := compStore.Get(args[0])
	if err != nil {
		return err
	}

	if c.Content == "" {
		return fmt.Errorf("composition has no content")
	}

	fmt.Printf("Analyzing: %s\n", c.Title)
	fmt.Printf("Content length: %d chars\n", len([]rune(c.Content)))
	fmt.Println("Sending to AI...")

	items, err := llm.AnalyzeComposition(cfg, c.Content, c.Title)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Show summary.
	fmt.Printf("\n=== Analysis ===\n%s\n", items.Summary)
	if len(items.SuggestedTags) > 0 {
		fmt.Printf("\nSuggested tags: %s\n", strings.Join(items.SuggestedTags, ", "))
	}

	// Show words.
	if len(items.Words) > 0 {
		fmt.Printf("\n=== Words (%d) ===\n", len(items.Words))
		for i, w := range items.Words {
			fmt.Printf("  [%d] %s", i+1, w.Word)
			if len(w.Definitions) > 0 {
				fmt.Printf("  %s %s", w.Definitions[0].Pos, w.Definitions[0].Meaning)
			}
			fmt.Println()
			if w.Example != "" {
				fmt.Printf("      ex: %s\n", truncate(w.Example, 100))
			}
			if w.Notes != "" {
				fmt.Printf("      note: %s\n", w.Notes)
			}
		}
	}

	// Show phrases.
	if len(items.Phrases) > 0 {
		fmt.Printf("\n=== Phrases (%d) ===\n", len(items.Phrases))
		for i, p := range items.Phrases {
			fmt.Printf("  [%d] %s  [%s]\n", i+1, p.Phrase, p.Type)
			fmt.Printf("      %s\n", p.Definition)
			if p.Example != "" {
				fmt.Printf("      ex: %s\n", truncate(p.Example, 100))
			}
		}
	}

	// Show sentences.
	if len(items.Sentences) > 0 {
		fmt.Printf("\n=== Sentences (%d) ===\n", len(items.Sentences))
		for i, s := range items.Sentences {
			fmt.Printf("  [%d] %s\n", i+1, truncate(s.Text, 120))
			if s.Translation != "" {
				fmt.Printf("      → %s\n", s.Translation)
			}
			if s.Why != "" {
				fmt.Printf("      why: %s\n", s.Why)
			}
		}
	}

	// Show grammar errors.
	if len(items.GrammarErrors) > 0 {
		fmt.Printf("\n=== Grammar Errors (%d) ===\n", len(items.GrammarErrors))
		for i, g := range items.GrammarErrors {
			fmt.Printf("  [%d] %s\n", i+1, g.ErrorType)
			fmt.Printf("      ✗ %s\n", g.Sentence)
			fmt.Printf("      ✓ %s\n", g.Correction)
			if g.Explanation != "" {
				fmt.Printf("      → %s\n", g.Explanation)
			}
		}
	}

	// Confirm save.
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\nSave analysis to composition? [Y/n]: ")
	if !scanner.Scan() {
		return nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer == "n" || answer == "no" {
		fmt.Println("Cancelled.")
		return nil
	}

	c.AIAnalysis = items.ToAIAnalysis()
	if len(items.SuggestedTags) > 0 {
		existing := make(map[string]bool)
		for _, t := range c.Tags {
			existing[t] = true
		}
		for _, t := range items.SuggestedTags {
			if !existing[t] {
				c.Tags = append(c.Tags, t)
				existing[t] = true
			}
		}
	}
	c.UpdatedAt = time.Now().UTC()
	if err := compStore.Update(*c); err != nil {
		return fmt.Errorf("update composition: %w", err)
	}

	fmt.Println("Analysis saved.")
	return nil
}
func formatComp(c *model.Composition) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", c.ID))
	b.WriteString(fmt.Sprintf("Title: %s\n", c.Title))
	if c.Author != "" {
		b.WriteString(fmt.Sprintf("Author: %s\n", c.Author))
	}
	if len(c.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(c.Tags, ", ")))
	}
	if c.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", c.Notes))
	}
	b.WriteString(fmt.Sprintf("\n%s\n", c.Content))

	if c.AIAnalysis != nil {
		b.WriteString(fmt.Sprintf("\n--- AI Analysis ---\n%s\n", c.AIAnalysis.Summary))
		if len(c.AIAnalysis.Words) > 0 {
			b.WriteString(fmt.Sprintf("\nWords (%d):\n", len(c.AIAnalysis.Words)))
			for _, w := range c.AIAnalysis.Words {
				b.WriteString(fmt.Sprintf("  %s", w.Word))
				if len(w.Definitions) > 0 {
					b.WriteString(fmt.Sprintf("  %s %s", w.Definitions[0].Pos, w.Definitions[0].Meaning))
				}
				b.WriteString("\n")
			}
		}
		if len(c.AIAnalysis.Phrases) > 0 {
			b.WriteString(fmt.Sprintf("\nPhrases (%d):\n", len(c.AIAnalysis.Phrases)))
			for _, p := range c.AIAnalysis.Phrases {
				b.WriteString(fmt.Sprintf("  %s [%s]  %s\n", p.Phrase, p.Type, p.Definition))
			}
		}
		if len(c.AIAnalysis.Sentences) > 0 {
			b.WriteString(fmt.Sprintf("\nSentences (%d):\n", len(c.AIAnalysis.Sentences)))
			for _, s := range c.AIAnalysis.Sentences {
				b.WriteString(fmt.Sprintf("  %s\n", truncate(s.Text, 100)))
			}
		}
		if len(c.AIAnalysis.GrammarErrors) > 0 {
			b.WriteString(fmt.Sprintf("\nGrammar Errors (%d):\n", len(c.AIAnalysis.GrammarErrors)))
			for _, g := range c.AIAnalysis.GrammarErrors {
				b.WriteString(fmt.Sprintf("  [%s] ✗ %s\n      ✓ %s\n", g.ErrorType, g.Sentence, g.Correction))
				if g.Explanation != "" {
					b.WriteString(fmt.Sprintf("      → %s\n", g.Explanation))
				}
			}
		}
	}

	b.WriteString(fmt.Sprintf("\nCreated: %s\n", c.CreatedAt.Format("2006-01-02 15:04")))
	return b.String()
}
