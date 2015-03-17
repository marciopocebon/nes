package nes

import "log"

type Mapper4 struct {
	*Cartridge
	nes        *NES
	register   byte
	registers  [8]byte
	prgMode    byte
	chrMode    byte
	prgOffsets [4]int
	chrOffsets [8]int
	latch      byte
	counter    byte
	reload     bool
	irq        bool
}

func NewMapper4(nes *NES, cartridge *Cartridge) Mapper {
	m := Mapper4{Cartridge: cartridge, nes: nes}
	m.prgOffsets[0] = m.prgBankOffset(0)
	m.prgOffsets[1] = m.prgBankOffset(1)
	m.prgOffsets[2] = m.prgBankOffset(-2)
	m.prgOffsets[3] = m.prgBankOffset(-1)
	return &m
}

func (m *Mapper4) Step() {
	ppu := m.nes.PPU
	if ppu.Cycle != 260 {
		return
	}
	if ppu.ScanLine > 239 && ppu.ScanLine < 261 {
		return
	}
	if ppu.flagShowBackground == 0 && ppu.flagShowSprites == 0 {
		return
	}
	m.HandleScanLine()
}

func (m *Mapper4) HandleScanLine() {
	if m.reload || m.counter == 0 {
		m.counter = m.latch
		m.reload = false
	} else {
		m.counter--
		if m.counter == 0 && m.irq {
			m.nes.CPU.triggerIRQ()
		}
	}
}

func (m *Mapper4) Read(address uint16) byte {
	switch {
	case address < 0x2000:
		bank := address / 0x0400
		offset := address % 0x0400
		return m.CHR[m.chrOffsets[bank]+int(offset)]
	case address >= 0x8000:
		address = address - 0x8000
		bank := address / 0x2000
		offset := address % 0x2000
		return m.PRG[m.prgOffsets[bank]+int(offset)]
	case address >= 0x6000:
		return m.SRAM[int(address)-0x6000]
	default:
		log.Fatalf("unhandled mapper4 read at address: 0x%04X", address)
	}
	return 0
}

func (m *Mapper4) Write(address uint16, value byte) {
	switch {
	case address < 0x2000:
		bank := address / 0x0400
		offset := address % 0x0400
		m.CHR[m.chrOffsets[bank]+int(offset)] = value
	case address >= 0x8000:
		m.writeRegister(address, value)
	case address >= 0x6000:
		m.SRAM[int(address)-0x6000] = value
	default:
		log.Fatalf("unhandled mapper4 write at address: 0x%04X", address)
	}
}

func (m *Mapper4) writeRegister(address uint16, value byte) {
	switch {
	case address <= 0x9FFF && address%2 == 0:
		m.writeBankSelect(value)
	case address <= 0x9FFF && address%2 == 1:
		m.writeBankData(value)
	case address <= 0xBFFF && address%2 == 0:
		m.writeMirror(value)
	case address <= 0xBFFF && address%2 == 1:
		m.writeProtect(value)
	case address <= 0xDFFF && address%2 == 0:
		m.writeIRQLatch(value)
	case address <= 0xDFFF && address%2 == 1:
		m.writeIRQReload(value)
	case address <= 0xFFFF && address%2 == 0:
		m.writeIRQDisable(value)
	case address <= 0xFFFF && address%2 == 1:
		m.writeIRQEnable(value)
	}
}

func (m *Mapper4) writeBankSelect(value byte) {
	m.prgMode = (value >> 6) & 1
	m.chrMode = (value >> 7) & 1
	m.register = value & 7
	m.updateOffsets()
}

func (m *Mapper4) writeBankData(value byte) {
	m.registers[m.register] = value
	m.updateOffsets()
}

func (m *Mapper4) writeMirror(value byte) {
	switch value & 1 {
	case 0:
		m.Cartridge.Mirror = MirrorVertical
	case 1:
		m.Cartridge.Mirror = MirrorHorizontal
	}
}

func (m *Mapper4) writeProtect(value byte) {
}

func (m *Mapper4) writeIRQLatch(value byte) {
	m.latch = value
}

func (m *Mapper4) writeIRQReload(value byte) {
	m.reload = true
	m.counter |= 0x80
}

func (m *Mapper4) writeIRQDisable(value byte) {
	m.irq = false
}

func (m *Mapper4) writeIRQEnable(value byte) {
	m.irq = true
}

func (m *Mapper4) prgBankOffset(index int) int {
	offset := index * 0x2000
	if offset < 0 {
		offset += len(m.PRG)
	}
	return offset
}

func (m *Mapper4) chrBankOffset(index int) int {
	offset := index * 0x0400
	if offset < 0 {
		offset += len(m.CHR)
	}
	return offset
}

func (m *Mapper4) updateOffsets() {
	switch m.prgMode {
	case 0:
		m.prgOffsets[0] = m.prgBankOffset(int(m.registers[6]))
		m.prgOffsets[1] = m.prgBankOffset(int(m.registers[7]))
		m.prgOffsets[2] = m.prgBankOffset(-2)
		m.prgOffsets[3] = m.prgBankOffset(-1)
	case 1:
		m.prgOffsets[0] = m.prgBankOffset(-2)
		m.prgOffsets[1] = m.prgBankOffset(int(m.registers[7]))
		m.prgOffsets[2] = m.prgBankOffset(int(m.registers[6]))
		m.prgOffsets[3] = m.prgBankOffset(-1)
	}
	switch m.chrMode {
	case 0:
		m.chrOffsets[0] = m.chrBankOffset(int(m.registers[0] & 0xFE))
		m.chrOffsets[1] = m.chrBankOffset(int(m.registers[0] | 0x01))
		m.chrOffsets[2] = m.chrBankOffset(int(m.registers[1] & 0xFE))
		m.chrOffsets[3] = m.chrBankOffset(int(m.registers[1] | 0x01))
		m.chrOffsets[4] = m.chrBankOffset(int(m.registers[2]))
		m.chrOffsets[5] = m.chrBankOffset(int(m.registers[3]))
		m.chrOffsets[6] = m.chrBankOffset(int(m.registers[4]))
		m.chrOffsets[7] = m.chrBankOffset(int(m.registers[5]))
	case 1:
		m.chrOffsets[0] = m.chrBankOffset(int(m.registers[2]))
		m.chrOffsets[1] = m.chrBankOffset(int(m.registers[3]))
		m.chrOffsets[2] = m.chrBankOffset(int(m.registers[4]))
		m.chrOffsets[3] = m.chrBankOffset(int(m.registers[5]))
		m.chrOffsets[4] = m.chrBankOffset(int(m.registers[0] & 0xFE))
		m.chrOffsets[5] = m.chrBankOffset(int(m.registers[0] | 0x01))
		m.chrOffsets[6] = m.chrBankOffset(int(m.registers[1] & 0xFE))
		m.chrOffsets[7] = m.chrBankOffset(int(m.registers[1] | 0x01))
	}
}