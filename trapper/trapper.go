package trapper

import (
	"fmt"

	"github.com/murkland/tango/mgba"
)

type Trapper struct {
	core  *mgba.Core
	traps map[uint32]trap
}

const trapOpcode = 0xbeef // bkpt 0xef

type trap struct {
	original uint16
	handler  func()
}

func New(core *mgba.Core) *Trapper {
	return &Trapper{core, map[uint32]trap{}}
}

func (s *Trapper) Add(addr uint32, handler func()) {
	if _, ok := s.traps[addr]; ok {
		panic(fmt.Sprintf("trap at 0x%08x already exists", addr))
	}
	t := trap{s.core.RawRead16(addr, -1), handler}
	s.core.RawWrite16(addr, -1, trapOpcode)
	s.traps[addr] = t
}

func (s *Trapper) BeefHandler() {
	const wordSizeThumb = 2
	caller := s.core.GBA().Register(15) - wordSizeThumb*2

	trap := s.traps[caller]
	if trap.handler == nil {
		panic(fmt.Sprintf("unhandled trap at 0x%08x", caller))
	}

	s.core.GBA().ARMRunFake(trap.original)
	trap.handler()
}
