package h

import (
	"github.com/eaburns/peggy/peg"
)

const (
	_sep   int = 0
	_space int = 1
	_h     int = 2
	_H     int = 3

	_N int = 4
)

type _Parser struct {
	text     string
	deltaPos [][_N]int32
	deltaErr [][_N]int32
	node     map[_key]*peg.Node
	fail     map[_key]*peg.Fail
	act      map[_key]interface{}
	lastFail int
	data     interface{}
}

type _key struct {
	start int
	rule  int
}

type tooBigError struct{}

func (tooBigError) Error() string { return "input is too big" }

func _NewParser(text string) (*_Parser, error) {
	n := len(text) + 1
	if n < 0 {
		return nil, tooBigError{}
	}
	p := &_Parser{
		text:     text,
		deltaPos: make([][_N]int32, n),
		deltaErr: make([][_N]int32, n),
		node:     make(map[_key]*peg.Node),
		fail:     make(map[_key]*peg.Fail),
		act:      make(map[_key]interface{}),
	}
	return p, nil
}

func _max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func _memoize(parser *_Parser, rule, start, pos, perr int) (int, int) {
	parser.lastFail = perr
	derr := perr - start
	parser.deltaErr[start][rule] = int32(derr + 1)
	if pos >= 0 {
		dpos := pos - start
		parser.deltaPos[start][rule] = int32(dpos + 1)
		return dpos, derr
	}
	parser.deltaPos[start][rule] = -1
	return -1, derr
}

func _memo(parser *_Parser, rule, start int) (int, int, bool) {
	dp := parser.deltaPos[start][rule]
	if dp == 0 {
		return 0, 0, false
	}
	if dp > 0 {
		dp--
	}
	de := parser.deltaErr[start][rule] - 1
	return int(dp), int(de), true
}

func _failMemo(parser *_Parser, rule, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start][rule]
	de := parser.deltaErr[start][rule]
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	f := parser.fail[_key{start: start, rule: rule}]
	if dp < 0 && f != nil {
		return -1, f
	}
	if dp > 0 && f != nil {
		return start + int(dp-1), f
	}
	return start, nil
}

func _accept(parser *_Parser, f func(*_Parser, int) (int, int), pos, perr *int) bool {
	dp, de := f(parser, *pos)
	*perr = _max(*perr, *pos+de)
	if dp < 0 {
		return false
	}
	*pos += dp
	return true
}

func _node(parser *_Parser, f func(*_Parser, int) (int, *peg.Node), node *peg.Node, pos *int) bool {
	p, kid := f(parser, *pos)
	if kid == nil {
		return false
	}
	node.Kids = append(node.Kids, kid)
	*pos = p
	return true
}

func _fail(parser *_Parser, f func(*_Parser, int, int) (int, *peg.Fail), errPos int, node *peg.Fail, pos *int) bool {
	p, kid := f(parser, *pos, errPos)
	if kid.Want != "" || len(kid.Kids) > 0 {
		node.Kids = append(node.Kids, kid)
	}
	if p < 0 {
		return false
	}
	*pos = p
	return true
}

func _next(parser *_Parser, pos int) (rune, int) {
	r, w := peg.DecodeRuneInString(parser.text[pos:])
	return r, w
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

// A no-op function to mark a variable as used.
func use(interface{}) {}

func _sepAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _sep, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// space+ h
	// space+
	// space
	if !_accept(parser, _spaceAccepts, &pos, &perr) {
		goto fail
	}
	for {
		pos2 := pos
		// space
		if !_accept(parser, _spaceAccepts, &pos, &perr) {
			goto fail4
		}
		continue
	fail4:
		pos = pos2
		break
	}
	// h
	if !_accept(parser, _hAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _sep, start, pos, perr)
fail:
	return _memoize(parser, _sep, start, -1, perr)
}

func _sepNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_sep]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _sep}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "sep"}
	// space+ h
	// space+
	// space
	if !_node(parser, _spaceNode, node, &pos) {
		goto fail
	}
	for {
		nkids1 := len(node.Kids)
		pos2 := pos
		// space
		if !_node(parser, _spaceNode, node, &pos) {
			goto fail4
		}
		continue
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		break
	}
	// h
	if !_node(parser, _hNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _sepFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _sep, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "sep",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _sep}
	// space+ h
	// space+
	// space
	if !_fail(parser, _spaceFail, errPos, failure, &pos) {
		goto fail
	}
	for {
		pos2 := pos
		// space
		if !_fail(parser, _spaceFail, errPos, failure, &pos) {
			goto fail4
		}
		continue
	fail4:
		pos = pos2
		break
	}
	// h
	if !_fail(parser, _hFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _sepAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_sep]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _sep}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// space+ h
	{
		var node0 string
		// space+
		{
			var node3 string
			// space
			if p, n := _spaceAction(parser, pos); n == nil {
				goto fail
			} else {
				node3 = *n
				pos = p
			}
			node0 += node3
		}
		for {
			pos2 := pos
			var node3 string
			// space
			if p, n := _spaceAction(parser, pos); n == nil {
				goto fail4
			} else {
				node3 = *n
				pos = p
			}
			node0 += node3
			continue
		fail4:
			pos = pos2
			break
		}
		node, node0 = node+node0, ""
		// h
		if p, n := _hAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _spaceAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _space, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// " "
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	return _memoize(parser, _space, start, pos, perr)
fail:
	return _memoize(parser, _space, start, -1, perr)
}

func _spaceNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_space]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _space}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "space"}
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
	pos, failure := _failMemo(parser, _space, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "space",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _space}
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

func _spaceAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_space]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _space}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// " "
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
		goto fail
	}
	node = parser.text[pos : pos+1]
	pos++
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _hAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _h, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// "h"/"'"
	{
		pos3 := pos
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			perr = _max(perr, pos)
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos3
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos++
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _h, start, pos, perr)
fail:
	return _memoize(parser, _h, start, -1, perr)
}

func _hNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_h]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _h}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "h"}
	// "h"/"'"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			goto fail4
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			goto fail5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
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
	pos, failure := _failMemo(parser, _h, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "h",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _h}
	// "h"/"'"
	{
		pos3 := pos
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"h\"",
				})
			}
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos3
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"'\"",
				})
			}
			goto fail5
		}
		pos++
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _hAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_h]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _h}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// "h"/"'"
	{
		pos3 := pos
		var node2 string
		// "h"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "h" {
			goto fail4
		}
		node = parser.text[pos : pos+1]
		pos++
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// "'"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "'" {
			goto fail5
		}
		node = parser.text[pos : pos+1]
		pos++
		goto ok0
	fail5:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _HAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _H, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// h+ sep+/h
	{
		pos3 := pos
		// h+ sep+
		// h+
		// h
		if !_accept(parser, _hAccepts, &pos, &perr) {
			goto fail4
		}
		for {
			pos7 := pos
			// h
			if !_accept(parser, _hAccepts, &pos, &perr) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		// sep+
		// sep
		if !_accept(parser, _sepAccepts, &pos, &perr) {
			goto fail4
		}
		for {
			pos11 := pos
			// sep
			if !_accept(parser, _sepAccepts, &pos, &perr) {
				goto fail13
			}
			continue
		fail13:
			pos = pos11
			break
		}
		goto ok0
	fail4:
		pos = pos3
		// h
		if !_accept(parser, _hAccepts, &pos, &perr) {
			goto fail14
		}
		goto ok0
	fail14:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _H, start, pos, perr)
fail:
	return _memoize(parser, _H, start, -1, perr)
}

func _HNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_H]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _H}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "H"}
	// h+ sep+/h
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// h+ sep+
		// h+
		// h
		if !_node(parser, _hNode, node, &pos) {
			goto fail4
		}
		for {
			nkids6 := len(node.Kids)
			pos7 := pos
			// h
			if !_node(parser, _hNode, node, &pos) {
				goto fail9
			}
			continue
		fail9:
			node.Kids = node.Kids[:nkids6]
			pos = pos7
			break
		}
		// sep+
		// sep
		if !_node(parser, _sepNode, node, &pos) {
			goto fail4
		}
		for {
			nkids10 := len(node.Kids)
			pos11 := pos
			// sep
			if !_node(parser, _sepNode, node, &pos) {
				goto fail13
			}
			continue
		fail13:
			node.Kids = node.Kids[:nkids10]
			pos = pos11
			break
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// h
		if !_node(parser, _hNode, node, &pos) {
			goto fail14
		}
		goto ok0
	fail14:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
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
	pos, failure := _failMemo(parser, _H, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "H",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _H}
	// h+ sep+/h
	{
		pos3 := pos
		// h+ sep+
		// h+
		// h
		if !_fail(parser, _hFail, errPos, failure, &pos) {
			goto fail4
		}
		for {
			pos7 := pos
			// h
			if !_fail(parser, _hFail, errPos, failure, &pos) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		// sep+
		// sep
		if !_fail(parser, _sepFail, errPos, failure, &pos) {
			goto fail4
		}
		for {
			pos11 := pos
			// sep
			if !_fail(parser, _sepFail, errPos, failure, &pos) {
				goto fail13
			}
			continue
		fail13:
			pos = pos11
			break
		}
		goto ok0
	fail4:
		pos = pos3
		// h
		if !_fail(parser, _hFail, errPos, failure, &pos) {
			goto fail14
		}
		goto ok0
	fail14:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _HAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_H]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _H}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// h+ sep+/h
	{
		pos3 := pos
		var node2 string
		// h+ sep+
		{
			var node5 string
			// h+
			{
				var node8 string
				// h
				if p, n := _hAction(parser, pos); n == nil {
					goto fail4
				} else {
					node8 = *n
					pos = p
				}
				node5 += node8
			}
			for {
				pos7 := pos
				var node8 string
				// h
				if p, n := _hAction(parser, pos); n == nil {
					goto fail9
				} else {
					node8 = *n
					pos = p
				}
				node5 += node8
				continue
			fail9:
				pos = pos7
				break
			}
			node, node5 = node+node5, ""
			// sep+
			{
				var node12 string
				// sep
				if p, n := _sepAction(parser, pos); n == nil {
					goto fail4
				} else {
					node12 = *n
					pos = p
				}
				node5 += node12
			}
			for {
				pos11 := pos
				var node12 string
				// sep
				if p, n := _sepAction(parser, pos); n == nil {
					goto fail13
				} else {
					node12 = *n
					pos = p
				}
				node5 += node12
				continue
			fail13:
				pos = pos11
				break
			}
			node, node5 = node+node5, ""
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// h
		if p, n := _hAction(parser, pos); n == nil {
			goto fail14
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail14:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}
