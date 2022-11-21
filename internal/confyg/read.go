// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Module file parser.
// This is a simplified copy of Google's buildifier parser.

package confyg

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A Position describes the position between two bytes of input.
type Position struct {
	Line     int // line in input (starting at 1)
	LineRune int // rune in line (starting at 1)
	Byte     int // byte in input (starting at 0)
}

// add returns the position at the end of s, assuming it starts at p.
func (p Position) add(s string) Position {
	p.Byte += len(s)
	if n := strings.Count(s, "\n"); n > 0 {
		p.Line += n
		s = s[strings.LastIndex(s, "\n")+1:]
		p.LineRune = 1
	}
	p.LineRune += utf8.RuneCountInString(s)
	return p
}

// An Expr represents an input element.
type Expr interface {
	// Span returns the start and end position of the expression,
	// excluding leading or trailing comments.
	Span() (start, end Position)

	// Comment returns the comments attached to the expression.
	// This method would normally be named 'Comments' but that
	// would interfere with embedding a type of the same name.
	Comment() *Comments
}

// A Comment represents a single // comment.
type Comment struct {
	Start  Position
	Token  string // without trailing newline
	Suffix bool   // an end of line (not whole line) comment
}

// Comments collects the comments associated with an expression.
type Comments struct {
	Before []Comment // whole-line comments before this expression
	Suffix []Comment // end-of-line comments after this expression

	// For top-level expressions only, After lists whole-line
	// comments following the expression.
	After []Comment
}

// Comment returns the receiver. This isn't useful by itself, but
// a Comments struct is embedded into all the expression
// implementation types, and this gives each of those a Comment
// method to satisfy the Expr interface.
func (c *Comments) Comment() *Comments {
	return c
}

// A FileSyntax represents an entire go.mod file.
type FileSyntax struct {
	Name string // file path
	Comments
	Stmt []Expr
}

func (x *FileSyntax) Span() (start, end Position) {
	if len(x.Stmt) == 0 {
		return
	}
	start, _ = x.Stmt[0].Span()
	_, end = x.Stmt[len(x.Stmt)-1].Span()
	return start, end
}

// A CommentBlock represents a top-level block of comments separate
// from any rule.
type CommentBlock struct {
	Comments
	Start Position
}

func (x *CommentBlock) Span() (start, end Position) {
	return x.Start, x.Start
}

// A Line is a single line of tokens.
type Line struct {
	Comments
	Start Position
	Token []string
	End   Position
}

func (x *Line) Span() (start, end Position) {
	return x.Start, x.End
}

// A LineBlock is a factored block of lines, like
//
//	require (
//		"x"
//		"y"
//	)
//
type LineBlock struct {
	Comments
	Start  Position
	LParen LParen
	Token  []string
	Line   []*Line
	RParen RParen
}

func (x *LineBlock) Span() (start, end Position) {
	return x.Start, x.RParen.Pos.add(")")
}

// An LParen represents the beginning of a parenthesized line block.
// It is a place to store suffix comments.
type LParen struct {
	Comments
	Pos Position
}

func (x *LParen) Span() (start, end Position) {
	return x.Pos, x.Pos.add(")")
}

// An RParen represents the end of a parenthesized line block.
// It is a place to store whole-line (before) comments.
type RParen struct {
	Comments
	Pos Position
}

func (x *RParen) Span() (start, end Position) {
	return x.Pos, x.Pos.add(")")
}

// An input represents a single input file being parsed.
type input struct {
	// Lexing state.
	filename  string    // name of input file, for errors
	complete  []byte    // entire input
	remaining []byte    // remaining input
	token     []byte    // token being scanned
	lastToken string    // most recently returned token, for error messages
	pos       Position  // current input position
	comments  []Comment // accumulated comments
	endRule   int       // position of end of current rule

	// Parser state.
	file       *FileSyntax // returned top-level syntax tree
	parseError error       // error encountered during parsing

	// Comment assignment state.
	pre  []Expr // all expressions, in preorder traversal
	post []Expr // all expressions, in postorder traversal
}

func newInput(filename string, data []byte) *input {
	return &input{
		filename:  filename,
		complete:  data,
		remaining: data,
		pos:       Position{Line: 1, LineRune: 1, Byte: 0},
	}
}

// parse parses the input file.
func parse(file string, data []byte) (f *FileSyntax, err error) {
	in := newInput(file, data)
	// The parser panics for both routine errors like syntax errors
	// and for programmer bugs like array index errors.
	// Turn both into error returns. Catching bug panics is
	// especially important when processing many files.
	defer func() {
		if e := recover(); e != nil {
			if e == in.parseError {
				err = in.parseError
			} else {
				err = fmt.Errorf("%s:%d:%d: internal error: %v", in.filename, in.pos.Line, in.pos.LineRune, e)
			}
		}
	}()

	// Invoke the parser.
	in.parseFile()
	if in.parseError != nil {
		return nil, in.parseError
	}
	in.file.Name = in.filename

	// Assign comments to nearby syntax.
	in.assignComments()

	return in.file, nil
}

// Error is called to report an error.
// The reason s is often "syntax error".
// Error does not return: it panics.
func (in *input) Error(s string) {
	if s == "syntax error" && in.lastToken != "" {
		s += " near " + in.lastToken
	}
	in.parseError = fmt.Errorf("%s:%d:%d: %v", in.filename, in.pos.Line, in.pos.LineRune, s)
	panic(in.parseError)
}

// eof reports whether the input has reached end of file.
func (in *input) eof() bool {
	return len(in.remaining) == 0
}

// peekRune returns the next rune in the input without consuming it.
func (in *input) peekRune() int {
	if len(in.remaining) == 0 {
		return 0
	}
	r, _ := utf8.DecodeRune(in.remaining)
	return int(r)
}

// readRune consumes and returns the next rune in the input.
func (in *input) readRune() int {
	if len(in.remaining) == 0 {
		in.Error("internal lexer error: readRune at EOF")
	}
	r, size := utf8.DecodeRune(in.remaining)
	in.remaining = in.remaining[size:]
	if r == '\n' {
		in.pos.Line++
		in.pos.LineRune = 1
	} else {
		in.pos.LineRune++
	}
	in.pos.Byte += size
	return int(r)
}

type symType struct {
	pos    Position
	endPos Position
	text   string
}

// startToken marks the beginning of the next input token.
// It must be followed by a call to endToken, once the token has
// been consumed using readRune.
func (in *input) startToken(sym *symType) {
	in.token = in.remaining
	sym.text = ""
	sym.pos = in.pos
}

// endToken marks the end of an input token.
// It records the actual token string in sym.text if the caller
// has not done that already.
func (in *input) endToken(sym *symType) {
	if sym.text == "" {
		tok := string(in.token[:len(in.token)-len(in.remaining)])
		sym.text = tok
		in.lastToken = sym.text
	}
	sym.endPos = in.pos
}

// lex is called from the parser to obtain the next input token.
// It returns the token value (either a rune like '+' or a symbolic token _FOR)
// and sets val to the data associated with the token.
// For all our input tokens, the associated data is
// val.Pos (the position where the token begins)
// and val.Token (the input string corresponding to the token).
func (in *input) lex(sym *symType) int {
	// Skip past spaces, stopping at non-space or EOF.
	countNL := 0 // number of newlines we've skipped past
	for !in.eof() {
		// Skip over spaces. Count newlines so we can give the parser
		// information about where top-level blank lines are,
		// for top-level comment assignment.
		c := in.peekRune()
		if c == ' ' || c == '\t' || c == '\r' {
			in.readRune()
			continue
		}

		// Comment runs to end of line.
		if c == '#' {
			in.startToken(sym)

			// Is this comment the only thing on its line?
			// Find the last \n before this // and see if it's all
			// spaces from there to here.
			i := bytes.LastIndex(in.complete[:in.pos.Byte], []byte("\n"))
			suffix := len(bytes.TrimSpace(in.complete[i+1:in.pos.Byte])) > 0
			in.readRune()
			c = in.peekRune()
			if c != '#' {
				in.Error(fmt.Sprintf("unexpected input character %#q", c))
			}

			// Consume comment.
			for len(in.remaining) > 0 && in.readRune() != '\n' {
			}
			in.endToken(sym)

			sym.text = strings.TrimRight(sym.text, "\n")
			in.lastToken = "comment"

			// If we are at top level (not in a statement), hand the comment to
			// the parser as a _COMMENT token. The grammar is written
			// to handle top-level comments itself.
			if !suffix {
				// Not in a statement. Tell parser about top-level comment.
				return _COMMENT
			}

			// Otherwise, save comment for later attachment to syntax tree.
			if countNL > 1 {
				in.comments = append(in.comments, Comment{sym.pos, "", false})
			}
			in.comments = append(in.comments, Comment{sym.pos, sym.text, suffix})
			countNL = 1
			return _EOL
		}

		// Found non-space non-comment.
		break
	}

	// Found the beginning of the next token.
	in.startToken(sym)
	defer in.endToken(sym)

	// End of file.
	if in.eof() {
		in.lastToken = "EOF"
		return _EOF
	}

	// Punctuation tokens.
	switch c := in.peekRune(); c {
	case '\n':
		in.readRune()
		return c

	case '(':
		in.readRune()
		return c

	case ')':
		in.readRune()
		return c

	case '"', '`': // quoted string
		quote := c
		in.readRune()
		for {
			if in.eof() {
				in.pos = sym.pos
				in.Error("unexpected EOF in string")
			}
			if in.peekRune() == '\n' {
				in.Error("unexpected newline in string")
			}
			c := in.readRune()
			if c == quote {
				break
			}
			if c == '\\' && quote != '`' {
				if in.eof() {
					in.pos = sym.pos
					in.Error("unexpected EOF in string")
				}
				in.readRune()
			}
		}
		in.endToken(sym)
		return _STRING
	}

	// Checked all punctuation. Must be identifier token.
	if c := in.peekRune(); !isIdent(c) {
		in.Error(fmt.Sprintf("unexpected input character %#q", c))
	}

	// Scan over identifier.
	for isIdent(in.peekRune()) {
		in.readRune()
	}
	return _IDENT
}

// isIdent reports whether c is an identifier rune.
// We treat nearly all runes as identifier runes.
func isIdent(c int) bool {
	return c != 0 && !unicode.IsSpace(rune(c)) && c != '(' && c != ')' && c != '"' && c != '`'
}

// Comment assignment.
// We build two lists of all subexpressions, preorder and postorder.
// The preorder list is ordered by start location, with outer expressions first.
// The postorder list is ordered by end location, with outer expressions last.
// We use the preorder list to assign each whole-line comment to the syntax
// immediately following it, and we use the postorder list to assign each
// end-of-line comment to the syntax immediately preceding it.

// order walks the expression adding it and its subexpressions to the
// preorder and postorder lists.
func (in *input) order(x Expr) {
	if x != nil {
		in.pre = append(in.pre, x)
	}
	switch x := x.(type) {
	default:
		panic(fmt.Errorf("order: unexpected type %T", x))
	case nil:
		// nothing
	case *LParen, *RParen:
		// nothing
	case *CommentBlock:
		// nothing
	case *Line:
		// nothing
	case *FileSyntax:
		for _, stmt := range x.Stmt {
			in.order(stmt)
		}
	case *LineBlock:
		in.order(&x.LParen)
		for _, l := range x.Line {
			in.order(l)
		}
		in.order(&x.RParen)
	}
	if x != nil {
		in.post = append(in.post, x)
	}
}

// assignComments attaches comments to nearby syntax.
func (in *input) assignComments() {
	const debug = false

	// Generate preorder and postorder lists.
	in.order(in.file)

	// Split into whole-line comments and suffix comments.
	var line, suffix []Comment
	for _, com := range in.comments {
		if com.Suffix {
			suffix = append(suffix, com)
		} else {
			line = append(line, com)
		}
	}

	if debug {
		for _, c := range line {
			fmt.Fprintf(os.Stderr, "LINE %q :%d:%d #%d\n", c.Token, c.Start.Line, c.Start.LineRune, c.Start.Byte)
		}
	}

	// Assign line comments to syntax immediately following.
	for _, x := range in.pre {
		start, _ := x.Span()
		if debug {
			fmt.Printf("pre %T :%d:%d #%d\n", x, start.Line, start.LineRune, start.Byte)
		}
		xcom := x.Comment()
		for len(line) > 0 && start.Byte >= line[0].Start.Byte {
			if debug {
				fmt.Fprintf(os.Stderr, "ASSIGN LINE %q #%d\n", line[0].Token, line[0].Start.Byte)
			}
			xcom.Before = append(xcom.Before, line[0])
			line = line[1:]
		}
	}

	// Remaining line comments go at end of file.
	in.file.After = append(in.file.After, line...)

	if debug {
		for _, c := range suffix {
			fmt.Fprintf(os.Stderr, "SUFFIX %q :%d:%d #%d\n", c.Token, c.Start.Line, c.Start.LineRune, c.Start.Byte)
		}
	}

	// Assign suffix comments to syntax immediately before.
	for i := len(in.post) - 1; i >= 0; i-- {
		x := in.post[i]

		start, end := x.Span()
		if debug {
			fmt.Printf("post %T :%d:%d #%d :%d:%d #%d\n", x, start.Line, start.LineRune, start.Byte, end.Line, end.LineRune, end.Byte)
		}

		// Do not assign suffix comments to end of line block or whole file.
		// Instead assign them to the last element inside.
		switch x.(type) {
		case *FileSyntax:
			continue
		}

		// Do not assign suffix comments to something that starts
		// on an earlier line, so that in
		//
		//	x ( y
		//		z ) // comment
		//
		// we assign the comment to z and not to x ( ... ).
		if start.Line != end.Line {
			continue
		}
		xcom := x.Comment()
		for len(suffix) > 0 && end.Byte <= suffix[len(suffix)-1].Start.Byte {
			if debug {
				fmt.Fprintf(os.Stderr, "ASSIGN SUFFIX %q #%d\n", suffix[len(suffix)-1].Token, suffix[len(suffix)-1].Start.Byte)
			}
			xcom.Suffix = append(xcom.Suffix, suffix[len(suffix)-1])
			suffix = suffix[:len(suffix)-1]
		}
	}

	// We assigned suffix comments in reverse.
	// If multiple suffix comments were appended to the same
	// expression node, they are now in reverse. Fix that.
	for _, x := range in.post {
		reverseComments(x.Comment().Suffix)
	}

	// Remaining suffix comments go at beginning of file.
	in.file.Before = append(in.file.Before, suffix...)
}

// reverseComments reverses the []Comment list.
func reverseComments(list []Comment) {
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}

func (in *input) parseFile() {
	in.file = new(FileSyntax)
	var sym symType
	var cb *CommentBlock
	for {
		tok := in.lex(&sym)
		switch tok {
		case '\n':
			if cb != nil {
				in.file.Stmt = append(in.file.Stmt, cb)
				cb = nil
			}
		case _COMMENT:
			if cb == nil {
				cb = &CommentBlock{Start: sym.pos}
			}
			com := cb.Comment()
			com.Before = append(com.Before, Comment{Start: sym.pos, Token: sym.text})
		case _EOF:
			if cb != nil {
				in.file.Stmt = append(in.file.Stmt, cb)
			}
			return
		default:
			in.parseStmt(&sym)
			if cb != nil {
				in.file.Stmt[len(in.file.Stmt)-1].Comment().Before = cb.Before
				cb = nil
			}
		}
	}
}

func (in *input) parseStmt(sym *symType) {
	start := sym.pos
	end := sym.endPos
	token := []string{sym.text}
	for {
		tok := in.lex(sym)
		switch tok {
		case '\n', _EOF, _EOL:
			in.file.Stmt = append(in.file.Stmt, &Line{
				Start: start,
				Token: token,
				End:   end,
			})
			return
		case '(':
			in.file.Stmt = append(in.file.Stmt, in.parseLineBlock(start, token, sym))
			return
		default:
			token = append(token, sym.text)
			end = sym.endPos
		}
	}
}

func (in *input) parseLineBlock(start Position, token []string, sym *symType) *LineBlock {
	x := &LineBlock{
		Start:  start,
		Token:  token,
		LParen: LParen{Pos: sym.pos},
	}
	var comments []Comment
	for {
		tok := in.lex(sym)
		switch tok {
		case _EOL:
			// ignore
		case '\n':
			if len(comments) == 0 && len(x.Line) > 0 || len(comments) > 0 && comments[len(comments)-1].Token != "" {
				comments = append(comments, Comment{})
			}
		case _COMMENT:
			comments = append(comments, Comment{Start: sym.pos, Token: sym.text})
		case _EOF:
			in.Error(fmt.Sprintf("syntax error (unterminated block started at %s:%d:%d)", in.filename, x.Start.Line, x.Start.LineRune))
		case ')':
			x.RParen.Before = comments
			x.RParen.Pos = sym.pos
			tok = in.lex(sym)
			if tok != '\n' && tok != _EOF && tok != _EOL {
				in.Error("syntax error (expected newline after closing paren)")
			}
			return x
		default:
			l := in.parseLine(sym)
			x.Line = append(x.Line, l)
			l.Comment().Before = comments
			comments = nil
		}
	}
}

func (in *input) parseLine(sym *symType) *Line {
	start := sym.pos
	end := sym.endPos
	token := []string{sym.text}
	for {
		tok := in.lex(sym)
		switch tok {
		case '\n', _EOF, _EOL:
			return &Line{
				Start: start,
				Token: token,
				End:   end,
			}
		default:
			token = append(token, sym.text)
			end = sym.endPos
		}
	}
}

const (
	_EOF = -(1 + iota)
	_EOL
	_IDENT
	_STRING
	_COMMENT
)
