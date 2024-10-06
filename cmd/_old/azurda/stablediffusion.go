package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

func hallucinatePrompt(hash string) (string, int) {
	var sb strings.Builder
	if hash[0] > '0' && hash[0] <= '5' {
		fmt.Fprint(&sb, "1girl, ")
	} else {
		fmt.Fprint(&sb, "1guy, ")
	}

	switch hash[1] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "blonde, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "brown hair, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "red hair, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "black hair, ")
	default:
	}

	if hash[2] > '0' && hash[2] <= '5' {
		fmt.Fprint(&sb, "coffee shop, ")
	} else {
		fmt.Fprint(&sb, "landscape, outdoors, ")
	}

	if hash[3] > '0' && hash[3] <= '5' {
		fmt.Fprint(&sb, "hoodie, ")
	} else {
		fmt.Fprint(&sb, "sweatsuit, ")
	}

	switch hash[4] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "<lora:cdi:1>, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "breath of the wild, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "genshin impact, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "arknights, ")
	default:
	}

	if hash[5] > '0' && hash[5] <= '5' {
		fmt.Fprint(&sb, "watercolor, ")
	} else {
		fmt.Fprint(&sb, "matte painting, ")
	}

	switch hash[6] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "highly detailed, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "ornate, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "thick lines, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "3d render, ")
	default:
	}

	switch hash[7] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "short hair, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "long hair, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "ponytail, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "pigtails, ")
	default:
	}

	switch hash[8] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "smile, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "frown, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "laughing, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "angry, ")
	default:
	}

	switch hash[9] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "sweater, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "tshirt, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "suitjacket, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "armor, ")
	default:
	}

	switch hash[10] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "blue eyes, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "red eyes, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "brown eyes, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "hazel eyes, ")
	default:
	}

	if hash[11] == '0' {
		fmt.Fprint(&sb, "heterochromia, ")
	}

	switch hash[12] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "morning, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "afternoon, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "evening, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "nighttime, ")
	default:
	}

	if hash[13] == '0' {
		fmt.Fprint(&sb, "<lora:genshin:1>, genshin, ")
	}

	switch hash[14] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "vtuber, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "anime, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "studio ghibli, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "cloverworks, ")
	default:
	}

	seedPortion := hash[len(hash)-9 : len(hash)-1]
	seed, err := strconv.ParseInt(seedPortion, 16, 32)
	if err != nil {
		seed = int64(rand.Int())
	}

	fmt.Fprint(&sb, "pants")

	return sb.String(), int(seed)
}
