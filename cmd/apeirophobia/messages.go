package main

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are a writer for a wiki about every known topic. When you are given the paths of URLs, you will return Markdown documents that explain the topic in question. You will add links to other topics with words separated by kebab-case.

For example, if you are asked about philosophy, you could include a link to Diogenes' philosophies, you could add a link like this:

[Diogenes' Philosophies](/wiki/diogenes-philosophies)

ONLY rely in markdown. DO NOT return anything but your response.`

func userPrompt(urlBit string) string {
	article := strings.Join(strings.Split(urlBit, "-"), " ")

	return fmt.Sprintf("Write a wiki article about %s. Link to other articles as relevant with Markdown links.", article)
}
