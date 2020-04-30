// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disasm_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-interpreter/wagon/disasm"
	"github.com/go-interpreter/wagon/wasm"
)

func TestDisassemble(t *testing.T) {
	for _, dir := range testPaths {
		fnames, err := filepath.Glob(filepath.Join(dir, "*.wasm"))
		if err != nil {
			t.Fatal(err)
		}
		for _, fname := range fnames {
			name := fname
			t.Run(filepath.Base(name), func(t *testing.T) {
				raw, err := ioutil.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}

				r := bytes.NewReader(raw)
				m, err := wasm.ReadModule(r, nil)
				if err != nil {
					t.Fatalf("error reading module %v", err)
				}
				for _, f := range m.FunctionIndexSpace {
					a, err := disasm.NewDisassembly(f, m)
					fmt.Println(a.Code)
					fmt.Println()
					if err != nil {
						t.Fatalf("disassemble failed: %v", err)
					}
				}
			})
		}
	}
}

func TestDis(t *testing.T) {
	name := "../wasm/testdata/addgas.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		raw, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		r := bytes.NewReader(raw)
		m, err := wasm.ReadModule(r, nil)
		if err != nil {
			t.Fatalf("error reading module %v", err)
		}
		for _, f := range m.FunctionIndexSpace {
			a, err := disasm.NewDisassembly(f, m)
			if err != nil {
				t.Fatalf("disassemble failed: %v", err)
			}
			for _,v := range a.Code {
				fmt.Println("Op:----------------------------")
				fmt.Println(v.Op.Code)
				fmt.Println(v.Op.Name)
				fmt.Println(v.Op.Args)
				fmt.Println(v.Op.Returns)
				fmt.Println("Immediates:--------------------")
				fmt.Println(v.Immediates)
				fmt.Println("Block:--------------------")
				fmt.Println(v.Block)
				fmt.Println("===============================")
			}
		}
	})
}

