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

var wordStore *store.WordStore

func InitWord(s *store.WordStore) {
	wordStore = s
}

func init() {
	register("word", runWord)
	register("words", runWord)
}

func runWord(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo word <word|id> [flags]")
	}

	first := args[0]

	switch first {
	case "add":
		return wordAdd(args[1:])
	case "list", "ls":
		return wordList(args[1:])
	case "search":
		return wordSearch(args[1:])
	}

	var editFlag, deleteFlag bool
	var tagFlag, synonymFlag, advancedFlag, noteFlag string

	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--edit" || a == "-e":
			editFlag = true
		case a == "--delete" || a == "-d":
			deleteFlag = true
		case strings.HasPrefix(a, "--tag="):
			tagFlag = strings.TrimPrefix(a, "--tag=")
		case strings.HasPrefix(a, "--synonym="):
			synonymFlag = strings.TrimPrefix(a, "--synonym=")
		case strings.HasPrefix(a, "--advanced="):
			advancedFlag = strings.TrimPrefix(a, "--advanced=")
		case strings.HasPrefix(a, "--note="):
			noteFlag = strings.TrimPrefix(a, "--note=")
		case a == "--tag" && i+1 < len(args):
			tagFlag = args[i+1]
			i++
		case a == "--synonym" && i+1 < len(args):
			synonymFlag = args[i+1]
			i++
		case a == "--advanced" && i+1 < len(args):
			advancedFlag = args[i+1]
			i++
		}
	}

	if deleteFlag {
		return wordDelete(first)
	}
	if tagFlag != "" {
		return wordSetTag(first, tagFlag)
	}
	if synonymFlag != "" {
		return wordAddSynonym(first, synonymFlag)
	}
	if advancedFlag != "" {
		return wordAddAdvanced(first, advancedFlag)
	}
	if noteFlag != "" {
		return wordSetNote(first, noteFlag)
	}
	if editFlag {
		return wordEdit(first)
	}
	return wordShow(first)
}

func wordAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo word add <word> [-m]")
	}

	word := args[0]
	manual := false
	for _, a := range args[1:] {
		if a == "--manual" || a == "-m" {
			manual = true
		}
	}

	fmt.Printf("Adding word: %s\n", word)

	now := time.Now().UTC()
	stability, difficulty, state := review.NewCardState(review.DefaultWeights)

	w := model.Word{
		ID:           store.NewID("wd"),
		Word:         word,
		Definitions:  []model.Definition{},
		Examples:     []string{},
		Inflections:  []model.Inflection{},
		Synonyms:     []string{},
		Advanced:     []string{},
		Tags:         []string{},
		Stability:    stability,
		Difficulty:   difficulty,
		State:        state,
		NextReviewAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if !manual {
		info, err := LookupWord(word)
		if err != nil {
			fmt.Printf("Dictionary lookup failed: %v\n", err)
			fmt.Println("Entering manual mode.")
			manual = true
		} else {
			w.Phonetic = info.Phonetic
			w.Definitions = info.Definitions
			w.Examples = info.Examples
			w.Inflections = info.Inflections


			fmt.Println("\n--- Dictionary Result ---")
			fmt.Println(formatWord(&w))
		}
	}

	if manual {
		return manualWordAdd(&w)
	}

	fmt.Print("\nSave? [Y/n]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := wordStore.Add(w); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	fmt.Printf("Saved: %s (%s)\n", w.Word, w.ID)
	return nil
}

func manualWordAdd(w *model.Word) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Phonetic: ")
	if scanner.Scan() {
		w.Phonetic = strings.TrimSpace(scanner.Text())
	}

	fmt.Println("Definitions (empty line to finish, format: pos meaning):")
	for {
		fmt.Print("  > ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			w.Definitions = append(w.Definitions, model.Definition{Pos: parts[0], Meaning: parts[1]})
		} else {
			w.Definitions = append(w.Definitions, model.Definition{Meaning: line})
		}
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
		w.Examples = append(w.Examples, line)
	}

	fmt.Print("Tags (comma-separated): ")
	if scanner.Scan() {
		tags := strings.TrimSpace(scanner.Text())
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				w.Tags = append(w.Tags, strings.TrimSpace(t))
			}
		}
	}

	fmt.Print("Notes: ")
	if scanner.Scan() {
		w.Notes = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("\nSave? [Y/n]: ")
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	return wordStore.Add(*w)
}

func wordShow(idPrefix string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Print(formatWord(w))
	return nil
}

func wordDelete(idPrefix string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Printf("Delete '%s' (%s)? [y/N]: ", w.Word, w.ID)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return wordStore.Delete(idPrefix)
}

func wordEdit(idPrefix string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}

	tmpFile := os.TempDir() + "/lingo_edit_word.json"
	pretty, _ := prettyJSON(w)
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

	var updated model.Word
	if err := jsonUnmarshal(data, &updated); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	updated.UpdatedAt = time.Now().UTC()

	return wordStore.Update(updated)
}

func wordSetTag(idPrefix, tags string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	w.Tags = []string{}
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			w.Tags = append(w.Tags, t)
		}
	}
	w.UpdatedAt = time.Now().UTC()
	return wordStore.Update(*w)
}

func wordAddSynonym(idPrefix, s string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	for _, ex := range w.Synonyms {
		if strings.EqualFold(ex, s) {
			return nil
		}
	}
	w.Synonyms = append(w.Synonyms, s)
	w.UpdatedAt = time.Now().UTC()
	return wordStore.Update(*w)
}

func wordAddAdvanced(idPrefix, a string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	for _, ex := range w.Advanced {
		if strings.EqualFold(ex, a) {
			return nil
		}
	}
	w.Advanced = append(w.Advanced, a)
	w.UpdatedAt = time.Now().UTC()
	return wordStore.Update(*w)
}

func wordSetNote(idPrefix, note string) error {
	w, err := wordStore.Get(idPrefix)
	if err != nil {
		return err
	}
	w.Notes = note
	w.UpdatedAt = time.Now().UTC()
	return wordStore.Update(*w)
}

func wordList(args []string) error {
	var tagFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--tag" && i+1 < len(args) {
			tagFilter = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--tag=") {
			tagFilter = strings.TrimPrefix(args[i], "--tag=")
		}
	}

	var tags []string
	if tagFilter != "" {
		tags = strings.Split(tagFilter, ",")
	}

	words, err := wordStore.Search(nil, tags)
	if err != nil {
		return err
	}

	for _, w := range words {
		def := ""
		if len(w.Definitions) > 0 {
			def = w.Definitions[0].Meaning
			if len(def) > 60 {
				def = def[:57] + "..."
			}
		}
		fmt.Printf("%-20s  %s  %s\n", w.Word, strings.Join(w.Tags, ","), def)
	}
	fmt.Printf("\n%d words\n", len(words))
	return nil
}

func wordSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo word search <keyword>")
	}

	words, err := wordStore.Search(args, nil)
	if err != nil {
		return err
	}

	for _, w := range words {
		fmt.Printf("%-20s  %s\n", w.Word, strings.Join(w.Tags, ","))
	}
	fmt.Printf("\n%d results\n", len(words))
	return nil
}

func formatWord(w *model.Word) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Word: %s\n", w.Word))
	if w.Phonetic != "" {
		b.WriteString(fmt.Sprintf("Phonetic: %s\n", w.Phonetic))
	}
	b.WriteString(fmt.Sprintf("ID: %s\n", w.ID))

	if len(w.Definitions) > 0 {
		b.WriteString("Definitions:\n")
		for _, d := range w.Definitions {
			pos := d.Pos
			if pos == "" {
				pos = "-"
			}
			b.WriteString(fmt.Sprintf("  [%s] %s\n", pos, d.Meaning))
		}
	}

	if len(w.Examples) > 0 {
		b.WriteString("Examples:\n")
		for _, e := range w.Examples {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	if len(w.Inflections) > 0 {
		b.WriteString("Inflections:\n")
		for _, i := range w.Inflections {
			b.WriteString(fmt.Sprintf("  %s: %s\n", i.Form, i.Value))
		}
	}

	if len(w.Synonyms) > 0 {
		b.WriteString(fmt.Sprintf("Synonyms: %s\n", strings.Join(w.Synonyms, ", ")))
	}
	if len(w.Advanced) > 0 {
		b.WriteString(fmt.Sprintf("Advanced: %s\n", strings.Join(w.Advanced, ", ")))
	}
	if len(w.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(w.Tags, ", ")))
	}

	if w.ReviewCount > 0 {
		b.WriteString(fmt.Sprintf("Reviewed: %d times (next: %s, stability: %.1f d, difficulty: %.1f)\n",
			w.ReviewCount, w.NextReviewAt.Format("2006-01-02"), w.Stability, w.Difficulty))
	}

	if w.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", w.Notes))
	}
	return b.String()
}
