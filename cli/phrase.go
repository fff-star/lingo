package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"lingo/model"
	"lingo/review"
	"lingo/store"
)

var phraseStore *store.PhraseStore

func InitPhrase(s *store.PhraseStore) {
	phraseStore = s
}

func init() {
	register("phrase", runPhrase)
	register("phrases", runPhrase)
}

func runPhrase(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo phrase <phrase|id> [flags]")
	}

	first := args[0]
	switch first {
	case "add":
		return phraseAdd(args[1:])
	case "list", "ls":
		return phraseList(args[1:])
	case "search":
		return phraseSearch(args[1:])
	}

	var editFlag, deleteFlag bool
	var tagFlag, synonymFlag, advancedFlag string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--edit", "-e":
			editFlag = true
		case "--delete", "-d":
			deleteFlag = true
		default:
			if strings.HasPrefix(args[i], "--tag=") {
				tagFlag = strings.TrimPrefix(args[i], "--tag=")
			} else if strings.HasPrefix(args[i], "--synonym=") {
				synonymFlag = strings.TrimPrefix(args[i], "--synonym=")
			} else if strings.HasPrefix(args[i], "--advanced=") {
				advancedFlag = strings.TrimPrefix(args[i], "--advanced=")
			} else if args[i] == "--tag" && i+1 < len(args) {
				tagFlag = args[i+1]; i++
			} else if args[i] == "--synonym" && i+1 < len(args) {
				synonymFlag = args[i+1]; i++
			} else if args[i] == "--advanced" && i+1 < len(args) {
				advancedFlag = args[i+1]; i++
			}
		}
	}

	if deleteFlag {
		return phraseDelete(first)
	}
	if tagFlag != "" {
		return phraseSetTag(first, tagFlag)
	}
	if synonymFlag != "" {
		return phraseAddSynonym(first, synonymFlag)
	}
	if advancedFlag != "" {
		return phraseAddAdvanced(first, advancedFlag)
	}
	if editFlag {
		return phraseEdit(first)
	}
	return phraseShow(first)
}

func phraseAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo phrase add <phrase> [-t type]")
	}

	phrase := args[0]
	phraseType := "other"
	for i := 1; i < len(args); i++ {
		if args[i] == "-t" && i+1 < len(args) {
			phraseType = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-t=") {
			phraseType = strings.TrimPrefix(args[i], "-t=")
		}
	}

	now := time.Now().UTC()
	stability, difficulty, state := review.NewCardState(review.DefaultWeights)
	words := strings.Fields(phrase)

	p := model.Phrase{
		ID:          store.NewID("ph"),
		Phrase:      phrase,
		Type:        phraseType,
		Words:       words,
		Examples:    []string{},
		Synonyms:    []string{},
		Advanced:    []string{},
		Tags:        []string{},
		Stability:   stability,
		Difficulty:  difficulty,
		State:       state,
		NextReviewAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return manualPhraseAdd(&p)
}

func manualPhraseAdd(p *model.Phrase) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Type [%s]: ", p.Type)
	if scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t != "" {
			p.Type = t
		}
	}

	fmt.Print("Definition: ")
	if scanner.Scan() {
		p.Definition = strings.TrimSpace(scanner.Text())
	}

	fmt.Println("Examples (empty line to finish):")
	for {
		fmt.Print("  > ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		p.Examples = append(p.Examples, line)
	}

	fmt.Print("Tags (comma-separated): ")
	if scanner.Scan() {
		tags := strings.TrimSpace(scanner.Text())
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				p.Tags = append(p.Tags, strings.TrimSpace(t))
			}
		}
	}

	fmt.Print("Notes: ")
	if scanner.Scan() {
		p.Notes = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("\nSave? [Y/n]: ")
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	return phraseStore.Add(*p)
}

func phraseShow(idPrefix string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Print(formatPhrase(p))
	return nil
}

func phraseDelete(idPrefix string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Printf("Delete '%s' (%s)? [y/N]: ", p.Phrase, p.ID)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return phraseStore.Delete(idPrefix)
}

func phraseEdit(idPrefix string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}

	tmpFile := os.TempDir() + "/lingo_edit_phrase.json"
	pretty, _ := prettyJSON(p)
	if err := os.WriteFile(tmpFile, pretty, 0644); err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := execCommand(editor, tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return err
	}
	os.Remove(tmpFile)

	var updated model.Phrase
	if err := jsonUnmarshal(data, &updated); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	updated.UpdatedAt = time.Now().UTC()

	return phraseStore.Update(updated)
}

func phraseSetTag(idPrefix, tags string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}
	p.Tags = []string{}
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			p.Tags = append(p.Tags, t)
		}
	}
	p.UpdatedAt = time.Now().UTC()
	return phraseStore.Update(*p)
}

func phraseAddSynonym(idPrefix, s string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}
	for _, ex := range p.Synonyms {
		if strings.EqualFold(ex, s) {
			return nil
		}
	}
	p.Synonyms = append(p.Synonyms, s)
	p.UpdatedAt = time.Now().UTC()
	return phraseStore.Update(*p)
}

func phraseAddAdvanced(idPrefix, a string) error {
	p, err := phraseStore.Get(idPrefix)
	if err != nil {
		return err
	}
	for _, ex := range p.Advanced {
		if strings.EqualFold(ex, a) {
			return nil
		}
	}
	p.Advanced = append(p.Advanced, a)
	p.UpdatedAt = time.Now().UTC()
	return phraseStore.Update(*p)
}

func phraseList(args []string) error {
	var tagFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--tag" && i+1 < len(args) {
			tagFilter = args[i+1]; i++
		} else if strings.HasPrefix(args[i], "--tag=") {
			tagFilter = strings.TrimPrefix(args[i], "--tag=")
		}
	}

	var tags []string
	if tagFilter != "" {
		tags = strings.Split(tagFilter, ",")
	}

	phrases, err := phraseStore.Search(nil, tags)
	if err != nil {
		return err
	}

	for _, p := range phrases {
		def := p.Definition
		if len(def) > 60 {
			def = def[:57] + "..."
		}
		fmt.Printf("%-25s [%s]  %s  %s\n", p.Phrase, p.Type, strings.Join(p.Tags, ","), def)
	}
	fmt.Printf("\n%d phrases\n", len(phrases))
	return nil
}

func phraseSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo phrase search <keyword>")
	}

	phrases, err := phraseStore.Search(args, nil)
	if err != nil {
		return err
	}

	for _, p := range phrases {
		fmt.Printf("%-25s [%s]  %s\n", p.Phrase, p.Type, strings.Join(p.Tags, ","))
	}
	fmt.Printf("\n%d results\n", len(phrases))
	return nil
}

func formatPhrase(p *model.Phrase) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Phrase: %s\n", p.Phrase))
	if p.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", p.Type))
	}
	b.WriteString(fmt.Sprintf("ID: %s\n", p.ID))
	if p.Definition != "" {
		b.WriteString(fmt.Sprintf("Definition: %s\n", p.Definition))
	}
	if len(p.Words) > 0 {
		b.WriteString(fmt.Sprintf("Words: %s\n", strings.Join(p.Words, ", ")))
	}
	if len(p.Examples) > 0 {
		b.WriteString("Examples:\n")
		for _, e := range p.Examples {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	if len(p.Synonyms) > 0 {
		b.WriteString(fmt.Sprintf("Synonyms: %s\n", strings.Join(p.Synonyms, ", ")))
	}
	if len(p.Advanced) > 0 {
		b.WriteString(fmt.Sprintf("Advanced: %s\n", strings.Join(p.Advanced, ", ")))
	}
	if len(p.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(p.Tags, ", ")))
	}
	if p.ReviewCount > 0 {
		b.WriteString(fmt.Sprintf("Reviewed: %d times (next: %s, stability: %.1f d, difficulty: %.1f)\n",
			p.ReviewCount, p.NextReviewAt.Format("2006-01-02"), p.Stability, p.Difficulty))
	}
	if p.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", p.Notes))
	}
	return b.String()
}
