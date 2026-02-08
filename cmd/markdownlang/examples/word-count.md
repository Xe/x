---
name: word-count
description: Count word count, character count, and find the longest word in the given text
input:
  type: object
  properties:
    text:
      type: string
  required: [text]
output:
  type: object
  properties:
    word_count:
      type: integer
    character_count:
      type: integer
    longest_word:
      type: string
  required: [word_count, character_count, longest_word]
---

Count the statistics of the given text:

1. Count the number of words (separated by whitespace)
2. Count the number of characters (including spaces)
3. Find the longest word

Text: {{ .text }}
