package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

type fuzzyItem struct {
	Type   string // [W], [P], [S], [A]
	ID     string
	Text   string // word, phrase, sentence text, or article title
	Detail string // definition, translation, source
	Tags   string // comma-separated tags
}

// fuzzyScore computes a match score for query against s.
// Returns (score, matched positions). score < 0 means no match.
func fuzzyScore(query, s string) (int, []int) {
	if query == "" {
		return 0, nil
	}
	query = strings.ToLower(query)
	sLower := strings.ToLower(s)

	var positions []int
	qIdx := 0
	prevPos := -1
	score := 0

	for i := 0; i < len(sLower); i++ {
		if qIdx >= len(query) {
			break
		}
		if sLower[i] == query[qIdx] {
			positions = append(positions, i)
			// Bonus for consecutive matches.
			if prevPos >= 0 && i == prevPos+1 {
				score += 10
			}
			// Bonus for match at start or after a separator.
			if i == 0 || sLower[i-1] == ' ' || sLower[i-1] == '_' || sLower[i-1] == '-' {
				score += 5
			}
			score += 1
			prevPos = i
			qIdx++
		}
	}

	if qIdx < len(query) {
		return -1, nil
	}
	// Penalize longer strings.
	score -= len(s) / 20
	return score, positions
}

func highlight(s string, positions []int) string {
	if len(positions) == 0 {
		return s
	}
	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}
	var b strings.Builder
	runes := []rune(s)
	byteIdx := 0
	runesOut := make([]rune, 0, len(runes))
	for _, r := range runes {
		rLen := utf8.RuneLen(r)
		if posSet[byteIdx] {
			b.WriteString("\033[1;33m") // bold yellow
			b.WriteRune(r)
			b.WriteString("\033[0m")
		} else {
			b.WriteRune(r)
		}
		runesOut = append(runesOut, r)
		byteIdx += rLen
	}
	return b.String()
}

type scoredItem struct {
	item      fuzzyItem
	score     int
	positions []int
}

func fuzzyPicker(items []fuzzyItem) (*fuzzyItem, error) {
	if len(items) == 0 {
		fmt.Println("No items.")
		return nil, nil
	}

	// Restore terminal on exit.
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: simple line-based search.
		return linePicker(items)
	}
	defer term.Restore(fd, oldState)

	query := ""
	selected := 0

	termWidth, termHeight, _ := term.GetSize(fd)
	_ = termWidth

	render := func() {
		// Clear screen and move to top.
		fmt.Print("\033[2J\033[H")

		// Score and filter items.
		var scored []scoredItem
		for _, item := range items {
			searchText := item.Text + " " + item.Detail + " " + item.Tags
			score, pos := fuzzyScore(query, searchText)
			if score >= 0 {
				scored = append(scored, scoredItem{item, score, pos})
			}
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		// Show results.
		start := 0
		if selected >= len(scored) {
			selected = len(scored) - 1
		}
		if selected < 0 {
			selected = 0
		}
		// Scroll window.
		if selected >= start+termHeight-3 {
			start = selected - (termHeight - 4)
		}
		if start < 0 {
			start = 0
		}

		visible := termHeight - 3
		if visible > len(scored) {
			visible = len(scored)
		}
		for i := 0; i < visible && i < len(scored); i++ {
			idx := i
			if visible <= len(scored) {
				idx = start + i
			}
			if idx >= len(scored) {
				break
			}
			s := scored[idx]
			line := fmt.Sprintf("%s %s", s.item.Type, highlight(s.item.Text, s.positions))
			if s.item.Detail != "" {
				detail := s.item.Detail
				if len(detail) > 60 {
					detail = detail[:57] + "..."
				}
				line += fmt.Sprintf("  \033[2m%s\033[0m", detail)
			}
			if idx == selected {
				fmt.Print("\033[7m") // reverse video
			}
			fmt.Print(line)
			fmt.Print("\033[0m\033[K\n")
		}

		// Status line.
		matchCount := len(scored)
		fmt.Printf("\033[7m\033[K query: %s\033[0m  %d matches   \033[2m↑↓:nav ↵:select esc:quit\033[0m",
			query, matchCount)
		fmt.Print("\033[K")
	}

	render()

	var buf [16]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			break
		}

		// Handle escape sequences (arrow keys: \x1b[A, \x1b[B, etc.)
		if n == 3 && buf[0] == 0x1b && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up
				if selected > 0 {
					selected--
				}
			case 'B': // Down
				selected++
			}
			render()
			continue
		}

		// Handle single bytes.
		for i := 0; i < n; i++ {
			b := buf[i]

			switch b {
			case 0x1b: // Esc
				fmt.Print("\033[2J\033[H")
				return nil, nil
			case 0x03: // Ctrl-C
				fmt.Print("\033[2J\033[H")
				return nil, fmt.Errorf("cancelled")
			case 0x0d, 0x0a: // Enter
				fmt.Print("\033[2J\033[H")
				// Find selected item.
				var scored []scoredItem
				for _, item := range items {
					searchText := item.Text + " " + item.Detail + " " + item.Tags
					score, _ := fuzzyScore(query, searchText)
					if score >= 0 {
						scored = append(scored, scoredItem{item, score, nil})
					}
				}
				sort.Slice(scored, func(i, j int) bool {
					return scored[i].score > scored[j].score
				})
				if len(scored) > 0 && selected < len(scored) {
					return &scored[selected].item, nil
				}
				return nil, nil
			case 0x7f: // Backspace
				if len(query) > 0 {
					_, size := utf8.DecodeLastRuneInString(query)
					query = query[:len(query)-size]
					selected = 0
				}
			default:
				if b >= 32 && b < 127 {
					query += string(b)
					selected = 0
				}
			}
		}
		render()
	}

	return nil, nil
}

// linePicker is a fallback when terminal raw mode isn't available.
func linePicker(items []fuzzyItem) (*fuzzyItem, error) {
	fmt.Printf("%d items. Type to search, or Enter to list all.\n", len(items))
	scanner := &lineScanner{}
	for {
		fmt.Print("search> ")
		query, ok := scanner.Scan()
		if !ok {
			return nil, nil
		}
		query = strings.TrimSpace(query)
		if query == "" || query == "q" {
			return nil, nil
		}

		var scored []scoredItem
		for _, item := range items {
			searchText := item.Text + " " + item.Detail + " " + item.Tags
			score, _ := fuzzyScore(query, searchText)
			if score >= 0 {
				scored = append(scored, scoredItem{item, score, nil})
			}
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		if len(scored) == 0 {
			fmt.Println("No matches.")
			continue
		}

		limit := 20
		if len(scored) < limit {
			limit = len(scored)
		}
		for i := 0; i < limit; i++ {
			s := scored[i]
			detail := s.item.Detail
			if len(detail) > 60 {
				detail = detail[:57] + "..."
			}
			fmt.Printf("  %2d. %s %s  %s\n", i+1, s.item.Type, s.item.Text, detail)
		}

		fmt.Print("Select number (or new query): ")
		sel, ok := scanner.Scan()
		if !ok {
			return nil, nil
		}
		sel = strings.TrimSpace(sel)
		if n, found := parseSelection(sel); found && n > 0 && n <= limit {
			return &scored[n-1].item, nil
		}
		// Treat as new query.
	}
}

type lineScanner struct{}

func (s *lineScanner) Scan() (string, bool) {
	var buf [4096]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil || n == 0 {
		return "", false
	}
	return strings.TrimRight(string(buf[:n]), "\r\n"), true
}

func parseSelection(s string) (int, bool) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
