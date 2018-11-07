// Command nasin-nasa is insanity. There's a fine line between genius and insanity.
// I have erased this line.
//
// This compiles toki pona toki into webassembly modules. This will be a proof of
// concept for lojban in the future.
//
// o sewi suli, "sina pali e ilo suli e ilo pona e ilo sona. ken la jan mute li jo e wawa sina. jan pali o, pali e ilo suli e ilo pona e ilo sona. sina ken pali e ilo ni. sina jo wawa mute. sina jo sona mute. sina o, pali e ilo."
package main

import (
	"flag"
	"log"

	"github.com/Xe/x/web/tokiponatokens"
	"github.com/kr/pretty"
)

var (
	verbs = []string{
		"toki",   // log at info level
		"pakala", // panic
		"pali",   // create nimi x1 with value x2
	}
)

var (
	apiURL = flag.String("tokiponatokens-api-url", "https://us-central1-golden-cove-408.cloudfunctions.net/function-1", "API url for package tokiponatokens")
)

const program = `ilo o, ona li toki e toki toki.`

func main() {
	flag.Parse()
	data, err := tokiponatokens.Tokenize(*apiURL, program)
	if err != nil {
		log.Fatal(err)
	}

	pretty.Println(data)
}

// Program is the AST of the program parsed from toki pona.
type Program struct {
	Steps           []Instruction
	Variables       map[string]Value
	LastUsedPointer int32
}

/*
func NewProgram(st []tokiponatokens.Sentence) (*Program, error) {
	p := &Program{
		Variables: map[string]Value{
			"pakala": Value{
				Str: "pakala",
				Ptr: 0x256,
			},
		},
		LastUsedPointer: 0x1024,
	}

	for _, sentence := range st {
		var (
			hasAddress bool
			subject    string
			verb       string
			object     []string
		)

	}
}
*/

func (p *Program) paliToki(nimi, toki string) (Instruction, bool) {
	_, ok := p.Variables[nimi]
	if !ok {
		return Instruction{}, false
	}

	lenToki := int32(len(toki))
	ptr := p.allocate(lenToki)

	val := Value{
		Str: toki,
		Ptr: ptr,
	}

	return Instruction{Verb: verbs[2], Object: val}, true
}

func (p *Program) allocate(amt int32) int32 {
	result := p.LastUsedPointer + 4
	p.LastUsedPointer = p.LastUsedPointer + 8 + amt
	return result
}

// Value is a string or number.
type Value struct {
	IsNum bool
	Int   int32
	Str   string
	Ptr   int32 // where this should be written to in ram
}

// Instruction is a single "VM" instruction.
type Instruction struct {
	Verb   string
	Object Value
}
