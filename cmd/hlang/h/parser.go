package h

//go:generate go run github.com/eaburns/peggy@v1.0.2 -o h_gen.go h.peg

import (
	"embed"
	"fmt"

	"github.com/eaburns/peggy/peg"
	"within.website/x/langs/jbo/namcu"
)

var (
	//go:embed h.peg
	Grammar embed.FS
)

func (p *_Parser) Parse() (int, bool) {
	pos, perr := _HAccepts(p, 0)
	return perr, pos >= 0
}

func (p *_Parser) ErrorTree(minPos int) *peg.Fail {
	p.fail = make(map[_key]*peg.Fail) // reset fail memo table
	_, tree := _HFail(p, 0, minPos)
	return tree
}

func (p *_Parser) ParseTree() *peg.Node {
	_, tree := _HNode(p, 0)
	return tree
}

// Parse parses h.
// On success, the parseTree is returned.
// On failure, both the word-level and the raw, morphological errors are returned.
func Parse(text string) (*peg.Node, error) {
	p, err := _NewParser(text)
	if err != nil {
		return nil, err
	}
	if perr, ok := p.Parse(); !ok {
		return nil, fmt.Errorf("h: gentoldra fi'o zvati fe li %s", namcu.Lerfu(perr))
	}

	tree := p.ParseTree()
	RemoveSpace(tree)
	CollapseLists(tree)

	return tree, nil
}
