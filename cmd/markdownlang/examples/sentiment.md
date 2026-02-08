---
name: sentiment-analysis
description: Analyze the sentiment of the given text (positive, negative, or neutral) with confidence score and explanation
input:
  type: object
  properties:
    text:
      type: string
      minLength: 1
  required: [text]
output:
  type: object
  properties:
    sentiment:
      type: string
      enum: [positive, negative, neutral]
    confidence:
      type: number
      minimum: 0
      maximum: 1
    explanation:
      type: string
  required: [sentiment, confidence, explanation]
mcp_servers:
  - name: python-interpreter
    command: uvx
    args: ["mcp-server-python-interpreter"]
---

Analyze the sentiment of the following text. Use the Python interpreter for any calculations.

Text to analyze: {{ .text }}

Return:

- sentiment: "positive", "negative", or "neutral"
- confidence: A number between 0 and 1
- explanation: A brief explanation of your analysis
