package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"lingo/model"
	"lingo/store"
)

var sentenceStore *store.SentenceStore

func InitSentence(s *store.SentenceStore) {
	sentenceStore = s
}

func init() {
	register("sentence", runSentence)
	register("sent", runSentence)
}

func runSentence(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo sent <id|subcommand>")
	}

	first := args[0]
	switch first {
	case "add":
		return sentenceAdd(args[1:])
	case "list", "ls":
		return sentenceList(args[1:])
	case "search":
		return sentenceSearch(args[1:])
	}

	var editFlag, deleteFlag bool
	var tagFlag string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--edit", "-e":
			editFlag = true
		case "--delete", "-d":
			deleteFlag = true
		default:
			if strings.HasPrefix(args[i], "--tag=") {
				tagFlag = strings.TrimPrefix(args[i], "--tag=")
			} else if args[i] == "--tag" && i+1 < len(args) {
				tagFlag = args[i+1]; i++
			}
		}
	}

	if deleteFlag {
		return sentenceDelete(first)
	}
	if tagFlag != "" {
		return sentenceSetTag(first, tagFlag)
	}
	if editFlag {
		return sentenceEdit(first)
	}
	return sentenceShow(first)
}

func sentenceAdd(args []string) error {
	fromFile := false
	var filePath string
	for i := 0; i < len(args); i++ {
		if args[i] == "--file" && i+1 < len(args) {
			fromFile = true
			filePath = args[i+1]
			i++
		}
	}

	if fromFile {
		return sentenceAddFromFile(filePath)
	}

	now := time.Now().UTC()
	st := model.Sentence{
		ID:        store.NewID("st"),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Text: ")
	if scanner.Scan() {
		st.Text = strings.TrimSpace(scanner.Text())
	}
	if st.Text == "" {
		return fmt.Errorf("text is required")
	}

	fmt.Print("Source (optional): ")
	if scanner.Scan() {
		st.Source = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Source URL (optional): ")
	if scanner.Scan() {
		st.SourceURL = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Author (optional): ")
	if scanner.Scan() {
		st.Author = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Translation (optional): ")
	if scanner.Scan() {
		st.Translation = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Tags (comma-separated): ")
	if scanner.Scan() {
		tags := strings.TrimSpace(scanner.Text())
		if tags != "" {
			for _, t := range strings.Split(tags, ",") {
				st.Tags = append(st.Tags, strings.TrimSpace(t))
			}
		}
	}

	fmt.Print("Notes: ")
	if scanner.Scan() {
		st.Notes = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("\nSave? [Y/n]: ")
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "n" || answer == "no" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	return sentenceStore.Add(st)
}

func sentenceAddFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	// Split by double newline for multiple sentences.
	blocks := strings.Split(string(data), "\n\n")

	var added int
	for _, block := range blocks {
		text := strings.TrimSpace(block)
		if text == "" {
			continue
		}
		st := model.Sentence{
			ID:        store.NewID("st"),
			Text:      text,
			Tags:      []string{},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := sentenceStore.Add(st); err != nil {
			if err == store.ErrExists {
				continue
			}
			return err
		}
		added++
	}
	fmt.Printf("Imported %d sentences\n", added)
	return nil
}

func sentenceShow(idPrefix string) error {
	st, err := sentenceStore.Get(idPrefix)
	if err != nil {
		return err
	}
	fmt.Print(formatSentence(st))
	return nil
}

func sentenceDelete(idPrefix string) error {
	st, err := sentenceStore.Get(idPrefix)
	if err != nil {
		return err
	}
	preview := st.Text
	if len(preview) > 40 {
		preview = preview[:37] + "..."
	}
	fmt.Printf("Delete '%s' (%s)? [y/N]: ", preview, st.ID)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return sentenceStore.Delete(idPrefix)
}

func sentenceEdit(idPrefix string) error {
	st, err := sentenceStore.Get(idPrefix)
	if err != nil {
		return err
	}

	tmpFile := os.TempDir() + "/lingo_edit_sentence.json"
	pretty, _ := prettyJSON(st)
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

	var updated model.Sentence
	if err := jsonUnmarshal(data, &updated); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	updated.UpdatedAt = time.Now().UTC()

	return sentenceStore.Update(updated)
}

func sentenceSetTag(idPrefix, tags string) error {
	st, err := sentenceStore.Get(idPrefix)
	if err != nil {
		return err
	}
	st.Tags = []string{}
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			st.Tags = append(st.Tags, t)
		}
	}
	st.UpdatedAt = time.Now().UTC()
	return sentenceStore.Update(*st)
}

func sentenceList(args []string) error {
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

	sentences, err := sentenceStore.Search(nil, tags)
	if err != nil {
		return err
	}

	for _, st := range sentences {
		text := st.Text
		if len(text) > 70 {
			text = text[:67] + "..."
		}
		fmt.Printf("%s  %s  %s\n", st.ID, strings.Join(st.Tags, ","), text)
	}
	fmt.Printf("\n%d sentences\n", len(sentences))
	return nil
}

func sentenceSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lingo sent search <keyword>")
	}

	sentences, err := sentenceStore.Search(args, nil)
	if err != nil {
		return err
	}

	for _, st := range sentences {
		text := st.Text
		if len(text) > 70 {
			text = text[:67] + "..."
		}
		fmt.Printf("%s  %s\n", st.ID, text)
	}
	fmt.Printf("\n%d results\n", len(sentences))
	return nil
}

func formatSentence(st *model.Sentence) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", st.ID))
	b.WriteString(fmt.Sprintf("Text: %s\n", st.Text))
	if st.Source != "" {
		b.WriteString(fmt.Sprintf("Source: %s", st.Source))
		if st.Author != "" {
			b.WriteString(fmt.Sprintf(" (%s)", st.Author))
		}
		b.WriteString("\n")
	}
	if st.SourceURL != "" {
		b.WriteString(fmt.Sprintf("URL: %s\n", st.SourceURL))
	}
	if st.Translation != "" {
		b.WriteString(fmt.Sprintf("Translation: %s\n", st.Translation))
	}
	if len(st.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(st.Tags, ", ")))
	}
	if st.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", st.Notes))
	}
	b.WriteString(fmt.Sprintf("Created: %s\n", st.CreatedAt.Format("2006-01-02 15:04")))
	return b.String()
}
