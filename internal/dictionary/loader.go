package dictionary

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Dictionary holds the word list and frequency data.
type Dictionary struct {
	entries map[string]int64 // word → frequency
}

// Load reads a dictionary file with word+frequency format.
// Each line should be: "word frequency" (space-separated).
func Load(dictPath string) (*Dictionary, error) {
	d := &Dictionary{
		entries: make(map[string]int64),
	}

	file, err := os.Open(dictPath)
	if err != nil {
		return nil, fmt.Errorf("opening dictionary file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}

		word := strings.ToLower(parts[0])
		freq := int64(1) // default frequency

		// If there's a second field, try to parse it as frequency
		if len(parts) >= 2 {
			if f, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				freq = f
			}
		}

		d.entries[word] = freq
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning dictionary: %w", err)
	}

	return d, nil
}

func (d *Dictionary) WordCount() int {
	return len(d.entries)
}

func (d *Dictionary) Entries() map[string]int64 {
	return d.entries
}
