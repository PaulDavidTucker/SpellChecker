#!/usr/bin/env python3
"""
Expand dictionary with additional words from various sources.
Preserves frequency information from original and adds reasonable frequencies to new words.
"""

import sys
from collections import OrderedDict

def load_dict_with_freq(filename):
    """Load dictionary with word frequencies. Returns dict of word->freq."""
    words = {}
    try:
        with open(filename, 'r') as f:
            for line in f:
                parts = line.strip().split()
                if len(parts) >= 2:
                    word = parts[0].lower()
                    try:
                        freq = int(parts[1])
                        words[word] = freq
                    except ValueError:
                        words[word] = 1
                elif len(parts) == 1:
                    words[parts[0].lower()] = 1
    except FileNotFoundError:
        print(f"Warning: {filename} not found")
    return words

def load_word_list(filename):
    """Load simple word list (one word per line, no frequencies)."""
    words = []
    try:
        with open(filename, 'r') as f:
            for line in f:
                word = line.strip().lower()
                if word and len(word) > 1:  # Skip single letters except 'a' and 'i'
                    words.append(word)
    except FileNotFoundError:
        print(f"Warning: {filename} not found")
    return words

def assign_frequencies(base_words, additional_words, min_freq=100):
    """
    Assign frequencies to additional words.
    Base words keep their original frequencies.
    Additional words get decreasing frequencies.
    """
    result = dict(base_words)
    
    # Get minimum frequency in base to start below it
    if base_words:
        base_min = min(base_words.values())
        current_freq = max(min_freq, base_min - 1000)
    else:
        current_freq = 100000
    
    # Decay factor - words get less frequent as we go down the list
    decay = 0.9999
    
    for word in additional_words:
        if word not in result:  # Don't overwrite existing words
            result[word] = int(current_freq)
            current_freq = max(current_freq * decay, 1)
    
    return result

def save_dict(words_dict, filename):
    """Save dictionary in word frequency format, sorted by frequency desc."""
    # Sort by frequency (desc), then alphabetically for same frequency
    sorted_words = sorted(words_dict.items(), key=lambda x: (-x[1], x[0]))
    
    with open(filename, 'w') as f:
        for word, freq in sorted_words:
            f.write(f"{word} {freq}\n")
    
    print(f"Saved {len(sorted_words)} words to {filename}")

def main():
    base_dict = "dictionaries/en_gb_medium.txt"
    
    # Load base dictionary
    print(f"Loading base dictionary from {base_dict}...")
    base_words = load_dict_with_freq(base_dict)
    print(f"  Loaded {len(base_words)} words")
    
    # Create LARGE dictionary (base + filtered additional words)
    print("\nCreating LARGE dictionary...")
    alpha_words = load_word_list("dictionaries/english_words_alpha.txt")
    # Filter: only add words that look like real words (no crazy letter combinations)
    # and limit to first 50K additional words
    filtered_alpha = [w for w in alpha_words[:50000] if len(w) <= 20 and len(w) >= 2]
    large_words = assign_frequencies(base_words, filtered_alpha, min_freq=50)
    save_dict(large_words, "dictionaries/en_gb_large.txt")
    
    # Create HUGE dictionary (base + all additional words)
    print("\nCreating HUGE dictionary...")
    all_alpha = [w for w in alpha_words if len(w) <= 25 and len(w) >= 2]
    huge_words = assign_frequencies(base_words, all_alpha, min_freq=10)
    save_dict(huge_words, "dictionaries/en_gb_huge.txt")
    
    print("\nDictionary expansion complete!")
    print(f"  Medium (base): {len(base_words):,} words")
    print(f"  Large: {len(large_words):,} words")
    print(f"  Huge: {len(huge_words):,} words")

if __name__ == "__main__":
    main()
