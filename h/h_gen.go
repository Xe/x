package h

import (
	"github.com/eaburns/peggy/peg"
)

type _Parser struct {
	text     string
	deltaPos []_Rules
	deltaErr []_Rules
	node     map[_key]*peg.Node
	fail     map[_key]*peg.Fail
	lastFail int
	data     interface{}
}

type _key struct {
	start int
	name  string
}

func _NewParser(text string) *_Parser {
	return &_Parser{
		text:     text,
		deltaPos: make([]_Rules, len(text)+1),
		deltaErr: make([]_Rules, len(text)+1),
		node:     make(map[_key]*peg.Node),
		fail:     make(map[_key]*peg.Fail),
	}
}

type _Rules struct {
	sep   int32
	space int32
	h     int32
	H     int32
}

func _max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func _next(parser *_Parser, pos int) (rune, int) {
	r, w := peg.DecodeRuneInString(parser.text[pos:])
	return r, w
}

func _node(name string) *peg.Node {
	return &peg.Node{Name: name}
}

func _sub(parser *_Parser, start, end int, kids []*peg.Node) *peg.Node {
	node := &peg.Node{
		Text: parser.text[start:end],
		Kids: make([]*peg.Node, len(kids)),
	}
	copy(node.Kids, kids)
	return node
}

func _leaf(parser *_Parser, start, end int) *peg.Node {
	return &peg.Node{Text: parser.text[start:end]}
}

func _sepAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp := parser.deltaPos[start].sep; dp != 0 {
		de := parser.deltaErr[start].sep - 1
		if dp > 0 {
			dp--
		}
		return int(dp), int(de)
	}
	pos, perr := start, -1
	// space+ h
	// space+
	// space
	if dp, de := _spaceAccepts(parser, pos); dp < 0 {
		perr = _max(perr, pos+de)
		goto fail
	} else {
		perr = _max(perr, pos+de)
		pos += dp
	}
	for {
		pos1 := pos
		// space
		if dp, de := _spaceAccepts(parser, pos); dp < 0 {
			perr = _max(perr, pos+de)
			goto fail2
		} else {
			perr = _max(perr, pos+de)
			pos += dp
		}
		continue
	fail2:
		pos = pos1
		break
	}
	// h
	if dp, de := _hAccepts(parser, pos); dp < 0 {
		perr = _max(perr, pos+de)
		goto fail
	} else {
		perr = _max(perr, pos+de)
		pos += dp
	}
	parser.deltaPos[start].sep = int32(pos-start) + 1
	parser.deltaErr[start].sep = int32(perr-start) + 1
	parser.lastFail = perr
	return pos - start, perr - start
fail:
	parser.deltaPos[start].sep = -1
	parser.deltaErr[start].sep = int32(perr-start) + 1
	parser.lastFail = perr
	return -1, perr - start
}

func _sepNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start].sep
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, name: "sep"}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = _node("sep")
	// space+ h
	// space+
	// space
	if p, kid := _spaceNode(parser, pos); kid == nil {
		goto fail
	} else {
		node.Kids = append(node.Kids, kid)
		pos = p
	}
	for {
		nkids0 := len(node.Kids)
		pos1 := pos
		// space
		if p, kid := _spaceNode(parser, pos); kid == nil {
			goto fail2
		} else {
			node.Kids = append(node.Kids, kid)
			pos = p
		}
		continue
	fail2:
		node.Kids = node.Kids[:nkids0]
		pos = pos1
		break
	}
	// h
	if p, kid := _hNode(parser, pos); kid == nil {
		goto fail
	} else {
		node.Kids = append(node.Kids, kid)
		pos = p
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _sepFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start].sep
	de := parser.deltaErr[start].sep
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	key := _key{start: start, name: "sep"}
	failure := parser.fail[key]
	if dp < 0 && failure != nil {
		return -1, failure
	}
	if dp > 0 && failure != nil {
		return start + int(dp-1), failure
	}
	pos := start
	failure = &peg.Fail{
		Name: "sep",
		Pos:  int(start),
	}
	// space+ h
	// space+
	// space
	{
		p, kid := _spaceFail(parser, pos, errPos)
		if kid.Want != "" || len(kid.Kids) > 0 {
			failure.Kids = append(failure.Kids, kid)
		}
		if p < 0 {
			goto fail
		}
		pos = p
	}
	for {
		pos1 := pos
		// space
		{
			p, kid := _spaceFail(parser, pos, errPos)
			if kid.Want != "" || len(kid.Kids) > 0 {
				failure.Kids = append(failure.Kids, kid)
			}
			if p < 0 {
				goto fail2
			}
			pos = p
		}
		continue
	fail2:
		pos = pos1
		break
	}
	// h
	{
		p, kid := _hFail(parser, pos, errPos)
		if kid.Want != "" || len(kid.Kids) > 0 {
			failure.Kids = append(failure.Kids, kid)
		}
		if p < 0 {
			goto fail
		}
		pos = p
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _spaceAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp := parser.deltaPos[start].space; dp != 0 {
		de := parser.deltaErr[start].space - 1
		if dp > 0 {
			dp--
		}
		return int(dp), int(de)
	}
	pos, perr := start, -1
	// " "
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	parser.deltaPos[start].space = int32(pos-start) + 1
	parser.deltaErr[start].space = int32(perr-start) + 1
	parser.lastFail = perr
	return pos - start, perr - start
fail:
	parser.deltaPos[start].space = -1
	parser.deltaErr[start].space = int32(perr-start) + 1
	parser.lastFail = perr
	return -1, perr - start
}

func _spaceNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start].space
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, name: "space"}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = _node("space")
	// " "
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _spaceFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start].space
	de := parser.deltaErr[start].space
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	key := _key{start: start, name: "space"}
	failure := parser.fail[key]
	if dp < 0 && failure != nil {
		return -1, failure
	}
	if dp > 0 && failure != nil {
		return start + int(dp-1), failure
	}
	pos := start
	failure = &peg.Fail{
		Name: "space",
		Pos:  int(start),
	}
	// " "
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\" \"",
			})
		}
		goto fail
	}
	pos++
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _hAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp := parser.deltaPos[start].h; dp != 0 {
		de := parser.deltaErr[start].h - 1
		if dp > 0 {
			dp--
		}
		return int(dp), int(de)
	}
	pos, perr := start, -1
	// "h"/"'"
	{
		pos2 := pos
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			perr = _max(perr, pos)
			goto fail3
		}
		pos++
		goto ok0
	fail3:
		pos = pos2
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			perr = _max(perr, pos)
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos2
		goto fail
	ok0:
	}
	parser.deltaPos[start].h = int32(pos-start) + 1
	parser.deltaErr[start].h = int32(perr-start) + 1
	parser.lastFail = perr
	return pos - start, perr - start
fail:
	parser.deltaPos[start].h = -1
	parser.deltaErr[start].h = int32(perr-start) + 1
	parser.lastFail = perr
	return -1, perr - start
}

func _hNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start].h
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, name: "h"}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = _node("h")
	// "h"/"'"
	{
		pos2 := pos
		nkids1 := len(node.Kids)
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			goto fail3
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			goto fail4
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _hFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start].h
	de := parser.deltaErr[start].h
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	key := _key{start: start, name: "h"}
	failure := parser.fail[key]
	if dp < 0 && failure != nil {
		return -1, failure
	}
	if dp > 0 && failure != nil {
		return start + int(dp-1), failure
	}
	pos := start
	failure = &peg.Fail{
		Name: "h",
		Pos:  int(start),
	}
	// "h"/"'"
	{
		pos2 := pos
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"h\"",
				})
			}
			goto fail3
		}
		pos++
		goto ok0
	fail3:
		pos = pos2
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"'\"",
				})
			}
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos2
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _HAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp := parser.deltaPos[start].H; dp != 0 {
		de := parser.deltaErr[start].H - 1
		if dp > 0 {
			dp--
		}
		return int(dp), int(de)
	}
	pos, perr := start, -1
	// h+ sep+/h
	{
		pos2 := pos
		// h+ sep+
		// h+
		// h
		if dp, de := _hAccepts(parser, pos); dp < 0 {
			perr = _max(perr, pos+de)
			goto fail3
		} else {
			perr = _max(perr, pos+de)
			pos += dp
		}
		for {
			pos5 := pos
			// h
			if dp, de := _hAccepts(parser, pos); dp < 0 {
				perr = _max(perr, pos+de)
				goto fail6
			} else {
				perr = _max(perr, pos+de)
				pos += dp
			}
			continue
		fail6:
			pos = pos5
			break
		}
		// sep+
		// sep
		if dp, de := _sepAccepts(parser, pos); dp < 0 {
			perr = _max(perr, pos+de)
			goto fail3
		} else {
			perr = _max(perr, pos+de)
			pos += dp
		}
		for {
			pos8 := pos
			// sep
			if dp, de := _sepAccepts(parser, pos); dp < 0 {
				perr = _max(perr, pos+de)
				goto fail9
			} else {
				perr = _max(perr, pos+de)
				pos += dp
			}
			continue
		fail9:
			pos = pos8
			break
		}
		goto ok0
	fail3:
		pos = pos2
		// h
		if dp, de := _hAccepts(parser, pos); dp < 0 {
			perr = _max(perr, pos+de)
			goto fail10
		} else {
			perr = _max(perr, pos+de)
			pos += dp
		}
		goto ok0
	fail10:
		pos = pos2
		goto fail
	ok0:
	}
	parser.deltaPos[start].H = int32(pos-start) + 1
	parser.deltaErr[start].H = int32(perr-start) + 1
	parser.lastFail = perr
	return pos - start, perr - start
fail:
	parser.deltaPos[start].H = -1
	parser.deltaErr[start].H = int32(perr-start) + 1
	parser.lastFail = perr
	return -1, perr - start
}

func _HNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start].H
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, name: "H"}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = _node("H")
	// h+ sep+/h
	{
		pos2 := pos
		nkids1 := len(node.Kids)
		// h+ sep+
		// h+
		// h
		if p, kid := _hNode(parser, pos); kid == nil {
			goto fail3
		} else {
			node.Kids = append(node.Kids, kid)
			pos = p
		}
		for {
			nkids4 := len(node.Kids)
			pos5 := pos
			// h
			if p, kid := _hNode(parser, pos); kid == nil {
				goto fail6
			} else {
				node.Kids = append(node.Kids, kid)
				pos = p
			}
			continue
		fail6:
			node.Kids = node.Kids[:nkids4]
			pos = pos5
			break
		}
		// sep+
		// sep
		if p, kid := _sepNode(parser, pos); kid == nil {
			goto fail3
		} else {
			node.Kids = append(node.Kids, kid)
			pos = p
		}
		for {
			nkids7 := len(node.Kids)
			pos8 := pos
			// sep
			if p, kid := _sepNode(parser, pos); kid == nil {
				goto fail9
			} else {
				node.Kids = append(node.Kids, kid)
				pos = p
			}
			continue
		fail9:
			node.Kids = node.Kids[:nkids7]
			pos = pos8
			break
		}
		goto ok0
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		// h
		if p, kid := _hNode(parser, pos); kid == nil {
			goto fail10
		} else {
			node.Kids = append(node.Kids, kid)
			pos = p
		}
		goto ok0
	fail10:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _HFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start].H
	de := parser.deltaErr[start].H
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	key := _key{start: start, name: "H"}
	failure := parser.fail[key]
	if dp < 0 && failure != nil {
		return -1, failure
	}
	if dp > 0 && failure != nil {
		return start + int(dp-1), failure
	}
	pos := start
	failure = &peg.Fail{
		Name: "H",
		Pos:  int(start),
	}
	// h+ sep+/h
	{
		pos2 := pos
		// h+ sep+
		// h+
		// h
		{
			p, kid := _hFail(parser, pos, errPos)
			if kid.Want != "" || len(kid.Kids) > 0 {
				failure.Kids = append(failure.Kids, kid)
			}
			if p < 0 {
				goto fail3
			}
			pos = p
		}
		for {
			pos5 := pos
			// h
			{
				p, kid := _hFail(parser, pos, errPos)
				if kid.Want != "" || len(kid.Kids) > 0 {
					failure.Kids = append(failure.Kids, kid)
				}
				if p < 0 {
					goto fail6
				}
				pos = p
			}
			continue
		fail6:
			pos = pos5
			break
		}
		// sep+
		// sep
		{
			p, kid := _sepFail(parser, pos, errPos)
			if kid.Want != "" || len(kid.Kids) > 0 {
				failure.Kids = append(failure.Kids, kid)
			}
			if p < 0 {
				goto fail3
			}
			pos = p
		}
		for {
			pos8 := pos
			// sep
			{
				p, kid := _sepFail(parser, pos, errPos)
				if kid.Want != "" || len(kid.Kids) > 0 {
					failure.Kids = append(failure.Kids, kid)
				}
				if p < 0 {
					goto fail9
				}
				pos = p
			}
			continue
		fail9:
			pos = pos8
			break
		}
		goto ok0
	fail3:
		pos = pos2
		// h
		{
			p, kid := _hFail(parser, pos, errPos)
			if kid.Want != "" || len(kid.Kids) > 0 {
				failure.Kids = append(failure.Kids, kid)
			}
			if p < 0 {
				goto fail10
			}
			pos = p
		}
		goto ok0
	fail10:
		pos = pos2
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}
