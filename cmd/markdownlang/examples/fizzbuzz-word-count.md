---
name: fizzbuzz-word-count
description: Runs FizzBuzz on a range and then counts the total words across all results, demonstrating agent composition
input:
  type: object
  properties:
    start:
      type: integer
      minimum: 1
    end:
      type: integer
      minimum: 1
  required: [start, end]
output:
  type: object
  properties:
    total_words:
      type: integer
    total_characters:
      type: integer
    results:
      type: array
      items:
        type: string
  required: [total_words, total_characters, results]
imports:
  - ./fizzbuzz.md
  - ./word-count.md
---

First, run fizzbuzz to get all results from {{ .start }} to {{ .end }}.

Then, for each result, count its words and characters using the word-count tool.

Finally, sum up all the word counts and character counts into totals with python.

Return the totals along with the original fizzbuzz results array.
