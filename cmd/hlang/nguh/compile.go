package nguh

import (
	"bytes"
	"fmt"
	"io"

	"github.com/eaburns/peggy/peg"
	"github.com/go-interpreter/wagon/wasm/leb128"
)

// Section is a WebAssembly binary section.
type Section struct {
	Kind uint32
	Len  uint32
	Data bytes.Buffer
}

// WebAssembly section types
const (
	SectionTypeType   = 0x01
	SectionTypeImport = 0x02
	SectionTypeFunc   = 0x03
	SectionTypeExport = 0x07
	SectionTypeCode   = 0x0a
)

// Commit turns a section into WebAssembly bytecode
func (s *Section) Commit(out io.Writer) error {
	if s.Len != uint32(s.Data.Len()) {
		s.Len = uint32(s.Data.Len())
	}

	if _, err := leb128.WriteVarUint32(out, s.Kind); err != nil {
		return err
	}
	if _, err := leb128.WriteVarUint32(out, s.Len); err != nil {
		return err
	}

	if _, err := io.Copy(out, &s.Data); err != nil {
		return err
	}

	return nil
}

const (
	ExportFunc  = 0x60
	WASMi32Type = 0x7f
)

func Compile(tree *peg.Node) ([]byte, error) {
	out := bytes.NewBuffer([]byte{
		// WebAssembly binary header / magic numbers
		0x00, 0x61, 0x73, 0x6d, // \0asm wasm magic number
		0x01, 0x00, 0x00, 0x00, // version 1
	})

	typeS := &Section{Kind: SectionTypeType}
	typeS.Data.Write([]byte{
		0x02,                                // 2 entries
		ExportFunc, 0x01, WASMi32Type, 0x00, // function type 0, 1 i32 param, 0 return
		ExportFunc, 0x00, 0x00, // function type 1, 0 param, 0 return
	})

	if err := typeS.Commit(out); err != nil {
		return nil, fmt.Errorf("can't write type section: %v", err)
	}

	importS := &Section{Kind: SectionTypeImport}
	importS.Data.Write([]byte{
		0x01,       // 1 entry
		0x01, 0x68, // module h
		0x01, 0x68, // name h
		0x00, // type index
		0x00, // function number
	})

	if err := importS.Commit(out); err != nil {
		return nil, fmt.Errorf("can't write import section: %v", err)
	}

	funcS := &Section{Kind: SectionTypeFunc}
	funcS.Data.Write([]byte{
		0x01, // function 1
		0x01, // type 1
	})

	if err := funcS.Commit(out); err != nil {
		return nil, fmt.Errorf("can't write func section: %v", err)
	}

	exportS := &Section{Kind: SectionTypeExport}
	exportS.Data.Write([]byte{
		0x01,       // 1 entry
		0x01, 0x68, // "h"
		0x00, 0x01, // function 1
	})

	if err := exportS.Commit(out); err != nil {
		return nil, fmt.Errorf("can't write export section: %v", err)
	}

	codeS := &Section{Kind: SectionTypeCode}
	codeS.Data.Write([]byte{
		0x01, // 1 entry
	})

	funcBuf := bytes.NewBuffer([]byte{
		0x01,       // 1 local declaration
		0x03, 0x7f, // 3 i32 values - (local i32 i32 i32)
		0x41, 0x0a, // i32.const 10 '\n'
		0x21, 0x00, // local.set 0
		0x41, 0xe8, 0x00, // i32.const 104 - 'h'
		0x21, 0x01, // local.set 1
		0x41, 0x27, // i32.const 39 '
		0x21, 0x02, // local.set 2
	})

	// compile AST to wasm
	if len(tree.Kids) == 0 {
		if err := compileOneNode(funcBuf, tree); err != nil {
			return nil, err
		}
	} else {
		for _, node := range tree.Kids {
			if err := compileOneNode(funcBuf, node); err != nil {
				return nil, err
			}
		}
	}

	// finally print newline
	funcBuf.Write([]byte{
		0x20, 0x00, // local.get 0
		0x10, 0x00, // call 0
		0x0b, // end of function
	})

	if _, err := leb128.WriteVarUint32(&codeS.Data, uint32(funcBuf.Len())); err != nil {
		return nil, err
	}

	if _, err := io.Copy(&codeS.Data, funcBuf); err != nil {
		return nil, err
	}

	if err := codeS.Commit(out); err != nil {
		return nil, fmt.Errorf("can't write code section: %v", err)
	}

	return out.Bytes(), nil
}

func compileOneNode(out io.Writer, node *peg.Node) error {
	switch node.Text {
	case "h":
		if _, err := out.Write([]byte{
			0x20, 0x01, // local.get 1
			0x10, 0x00, // call 0
		}); err != nil {
			return err
		}
	case "'":
		if _, err := out.Write([]byte{
			0x20, 0x02, // local.get 2
			0x10, 0x00, // call 0
		}); err != nil {
			return err
		}
	default:
		fmt.Errorf("h: le vi lerfu zo %q cu gentoldra", node.Text)
	}

	return nil
}
