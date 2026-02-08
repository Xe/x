---
name: fizzbuzz
description: FizzBuzz classic programming exercise - counts from start to end, replacing multiples of 3 with "Fizz", multiples of 5 with "Buzz", and multiples of both with "FizzBuzz"
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
    results:
      type: array
      items:
        type: string
  required: [results]
---

# FizzBuzz

For each number from {{ .start }} to {{ .end }}, output:

- "FizzBuzz" if divisible by both 3 and 5
- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- The number itself otherwise

Return the results as an array of strings.
