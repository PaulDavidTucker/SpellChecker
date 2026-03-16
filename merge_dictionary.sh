# merge_dictionary.sh
# Produces a single file: word<space>frequency
# Words from en_GB that have frequency data get their real frequency.
# Words without frequency data get a default of 1.

#!/bin/bash
WORDS="/tmp/en_gb_words.txt"
FREQ="/tmp/en_50k.txt"
OUTPUT="dictionaries/en_gb.txt"

awk 'NR==FNR {words[$1]; next} ($1 in words)' \
    "$WORDS" "$FREQ" > "$OUTPUT"

awk 'NR==FNR {seen[$1]; next} !($1 in seen) {print $1, 1}' \
    "$OUTPUT" "$WORDS" >> "$OUTPUT"

echo "Dictionary written to $OUTPUT"
wc -l "$OUTPUT"
