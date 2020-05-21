package main

import (
	"bytes"
	"flag"
	"github.com/ci123chain/wasm-util/disasm"
	"github.com/ci123chain/wasm-util/wasm"
	"io/ioutil"
	"log"
)

func main() {
	file := flag.String("i", "", "input filename")
	out := flag.String("o", "", "output filename")
	flag.Parse()

	raw, err := ioutil.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}

	r := bytes.NewReader(raw)
	m, pos, err := wasm.DecodeModuleAddGas(r)
	if err != nil {
		log.Fatal(err)
	}
	if m.Code == nil {
		log.Fatal(err)
	}

	for i := 0; i < len(m.Code.Bodies); i++ {
		d, err := disasm.DisassembleAddGas(m.Code.Bodies[i].Code, pos)
		if err != nil {
			log.Fatal(err)
		}
		code, err := disasm.Assemble(d)
		if err != nil {
			log.Fatal(err)
		}
		m.Code.Bodies[i].Code = code
	}

	buf := new(bytes.Buffer)
	err = wasm.EncodeModule(buf, m)

	m, err = wasm.DecodeModule(bytes.NewReader(buf.Bytes()))
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(*out, buf.Bytes(),0644)
	if err != nil {
		log.Fatal(err)
	}
}
