// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/ci123chain/wasm-util/wasm/internal/readpos"
)

var ErrInvalidMagic = errors.New("wasm: Invalid magic number")

const (
	Magic   uint32 = 0x6d736100
	Version uint32 = 0x1
)

// Function represents an entry in the function index space of a module.
type Function struct {
	Sig  *FunctionSig
	Body *FunctionBody
	Host reflect.Value
	Name string
}

// IsHost indicates whether this function is a host function as defined in:
//  https://webassembly.github.io/spec/core/exec/modules.html#host-functions
func (fct *Function) IsHost() bool {
	return fct.Host != reflect.Value{}
}

// Module represents a parsed WebAssembly module:
// http://webassembly.org/docs/modules/
type Module struct {
	Version  uint32
	Sections []Section

	Types    *SectionTypes
	Import   *SectionImports
	Function *SectionFunctions
	Table    *SectionTables
	Memory   *SectionMemories
	Global   *SectionGlobals
	Export   *SectionExports
	Start    *SectionStartFunction
	Elements *SectionElements
	Code     *SectionCode
	Data     *SectionData
	Customs  []*SectionCustom

	// The function index space of the module
	FunctionIndexSpace []Function
	GlobalIndexSpace   []GlobalEntry

	// function indices into the global function space
	// the limit of each table is its capacity (cap)
	TableIndexSpace        [][]TableEntry
	LinearMemoryIndexSpace [][]byte

	imports struct {
		Funcs    []uint32
		Globals  int
		Tables   int
		Memories int
	}
}

// TableEntry represents a table index and tracks its initialized state.
type TableEntry struct {
	Index       uint32
	Initialized bool
}

// Custom returns a custom section with a specific name, if it exists.
func (m *Module) Custom(name string) *SectionCustom {
	for _, s := range m.Customs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// NewModule creates a new empty module
func NewModule() *Module {
	return &Module{
		Types:    &SectionTypes{},
		Import:   &SectionImports{},
		Table:    &SectionTables{},
		Memory:   &SectionMemories{},
		Global:   &SectionGlobals{},
		Export:   &SectionExports{},
		Start:    &SectionStartFunction{},
		Elements: &SectionElements{},
		Data:     &SectionData{},
	}
}

// ResolveFunc is a function that takes a module name and
// returns a valid resolved module.
type ResolveFunc func(name string) (*Module, error)

// DecodeModule is the same as ReadModule, but it only decodes the module without
// initializing the index space or resolving imports.
func DecodeModule(r io.Reader) (*Module, error) {
	reader := &readpos.ReadPos{
		R:      r,
		CurPos: 0,
	}
	m := &Module{}
	magic, err := readU32(reader)
	if err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, ErrInvalidMagic
	}
	if m.Version, err = readU32(reader); err != nil {
		return nil, err
	}
	if m.Version != Version {
		return nil, fmt.Errorf("wasm: unknown binary version: %d", m.Version)
	}

	err = newSectionsReader(m).readSections(reader)
	if err != nil {
		return nil, err
	}
	return m, nil
}


func DecodeModuleAddGas(r io.Reader) (*Module, int, error) {
	reader := &readpos.ReadPos{
		R:      r,
		CurPos: 0,
	}
	m := &Module{}
	magic, err := readU32(reader)
	if err != nil {
		return nil, 0, err
	}
	if magic != Magic {
		return nil, 0, ErrInvalidMagic
	}
	if m.Version, err = readU32(reader); err != nil {
		return nil, 0, err
	}
	if m.Version != Version {
		return nil, 0, fmt.Errorf("wasm: unknown binary version: %d", m.Version)
	}

	err = newSectionsReader(m).readSections(reader)
	if err != nil {
		return nil, 0, err
	}

	//判断type是否存在
	hasType := false
	var tPos int
	if m.Types != nil {
		for k,v := range m.Types.Entries{
			if len(v.ParamTypes) == 1 && v.ParamTypes[0] == ValueTypeI32 && len(v.ReturnTypes) == 0{
				hasType = true
				tPos = k
				break
			}
		}
	} else {
		m.Types = &SectionTypes{
			RawSection: RawSection{
				Start: 0,
				End:   0,
				ID:    SectionID(uint8(SectionIDType)),
				Bytes: nil,
			},
			Entries:    []FunctionSig{},
		}
		m.Sections = append([]Section{m.Types}, m.Sections...)
	}

	if !hasType {
		entry := FunctionSig{
			Form:        TypeFunc,
			ParamTypes:  []ValueType{ValueTypeI32},
			ReturnTypes: nil,
		}
		m.Types.Entries = append(m.Types.Entries, entry)
		tPos = len(m.Types.Entries) - 1
	}

	//判断sectoinImport(02) 是否存在, 不存在就实例化
	if m.Import == nil {
		m.Import = &SectionImports{
			RawSection: RawSection{
				Start: 0,
				End:   0,
				ID:    SectionID(uint8(SectionIDImport)),
				Bytes: nil,
			},
			Entries:    []ImportEntry{},
		}
		temp := append([]Section{}, m.Sections[1:]...)
		m.Sections = append(m.Sections[:1], m.Import)
		m.Sections = append(m.Sections, temp...)
	}

	//添加 addgas function
	entry := ImportEntry{
		ModuleName: "env",
		FieldName:  "addgas",
		Type:       FuncImport{Type:uint32(tPos)},
	}
	m.Import.Entries = append(m.Import.Entries,entry)

	mp := make(map[string]ExportEntry)
	for k,v := range m.Export.Entries {
		if v.Kind == ExternalFunction {
			 v.Index++
		}
		mp[k] = v
	}
	m.Export.Entries = mp

	if m.Elements != nil {
		for i := 0; i < len(m.Elements.Entries[0].Elems); i++ {
			m.Elements.Entries[0].Elems[i]++
		}
	}

	return m, len(m.Import.Entries) - 1, nil
}

// ReadModule reads a module from the reader r. resolvePath must take a string
// and a return a reader to the module pointed to by the string.
func ReadModule(r io.Reader, resolvePath ResolveFunc) (*Module, error) {
	m, err := DecodeModule(r)
	if err != nil {
		return nil, err
	}

	m.LinearMemoryIndexSpace = make([][]byte, 1)
	if m.Table != nil {
		m.TableIndexSpace = make([][]TableEntry, int(len(m.Table.Entries)))
	}

	if m.Import != nil && resolvePath != nil {
		if m.Code == nil {
			m.Code = &SectionCode{}
		}

		err := m.resolveImports(resolvePath)
		if err != nil {
			return nil, err
		}
	}

	for _, fn := range []func() error{
		m.populateGlobals,
		m.populateFunctions,
		m.populateTables,
		m.populateLinearMemory,
	} {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	logger.Printf("There are %d entries in the function index space.", len(m.FunctionIndexSpace))
	return m, nil
}
