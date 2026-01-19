package main

import ("bytes"; "compress/zlib"; "encoding/base64"; "io"; "net"; "sync"; "time")

type Buffer struct{ data []byte; read int; write int }

func NewBuffer() *Buffer {
	return &Buffer{data: make([]byte, 0, 256), read: 0, write: 0}
}

func NewBufferFromBytes(data []byte) *Buffer {
	return &Buffer{data: data, read: 0, write: len(data)}
}

func (b *Buffer) Write(data []byte) *Buffer {
	b.data = append(b.data, data...)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteByte(v byte) *Buffer {
	b.data = append(b.data, v)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGChar(v byte) *Buffer {
	if v > 223 { v = 223 }
	b.WriteByte(v + 32)
	return b
}

func (b *Buffer) WriteChar(v int8) *Buffer { return b.WriteByte(byte(v)) }

func (b *Buffer) WriteShort(v int16) *Buffer {
	b.data = append(b.data, byte(v>>8), byte(v))
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteInt(v int32) *Buffer {
	b.data = append(b.data, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteInt64(v int64) *Buffer {
	b.data = append(b.data, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteInt3(v int32) *Buffer {
	b.data = append(b.data, byte(v>>16), byte(v>>8), byte(v))
	b.write = len(b.data)
	return b
}

// Graal encoding
func (b *Buffer) WriteGByte(v uint8) *Buffer {
	b.data = append(b.data, v)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGShort(v uint16) *Buffer {
	if v < 128 {
		b.data = append(b.data, byte(v))
	} else {
		b.data = append(b.data, byte((v>>8)|0x80), byte(v))
	}
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGInt(v uint32) *Buffer {
	if v < 0x80 {
		b.data = append(b.data, byte(v))
	} else if v < 0x4000 {
		b.data = append(b.data, byte((v>>8)|0x80), byte(v&0xFF))
	} else {
		b.data = append(b.data, byte((v>>16)|0x80), byte((v>>8)&0xFF), byte(v&0xFF))
	}
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGInt4(v uint32) *Buffer {
	b.data = append(b.data, byte((v>>24)|0x80), byte((v>>16)&0xFF), byte((v>>8)&0xFF), byte(v&0xFF))
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGInt5(v uint64) *Buffer {
	b.data = append(b.data, byte((v>>32)&0xFF), byte((v>>24)&0xFF), byte((v>>16)&0xFF), byte((v>>8)&0xFF), byte(v&0xFF))
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteGString(s string) *Buffer {
	strBytes := []byte(s)
	b.WriteGInt(uint32(len(strBytes)))
	b.data = append(b.data, strBytes...)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteString8(s string) *Buffer {
	strBytes := []byte(s)
	b.WriteByte(byte(len(strBytes)))
	b.data = append(b.data, strBytes...)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteString8Encoded(s string) *Buffer {
	strBytes := []byte(s)
	b.WriteGChar(byte(len(strBytes)))
	b.data = append(b.data, strBytes...)
	b.write = len(b.data)
	return b
}

func (b *Buffer) WriteString(s string) *Buffer {
	b.data = append(b.data, []byte(s)...)
	b.data = append(b.data, 0)
	b.write = len(b.data)
	return b
}

// Reading
func (b *Buffer) ReadByte() byte {
	if b.read >= len(b.data) { return 0 }
	v := b.data[b.read]
	b.read++
	return v
}

func (b *Buffer) ReadChar() int8 { return int8(b.ReadByte()) }

func (b *Buffer) ReadShort() int16 {
	if b.read+2 > len(b.data) { return 0 }
	v := int16(b.data[b.read])<<8 | int16(b.data[b.read+1])
	b.read += 2
	return v
}

func (b *Buffer) ReadInt() int32 {
	if b.read+4 > len(b.data) { return 0 }
	v := int32(b.data[b.read])<<24 | int32(b.data[b.read+1])<<16 | int32(b.data[b.read+2])<<8 | int32(b.data[b.read+3])
	b.read += 4
	return v
}

func (b *Buffer) ReadInt3() int32 {
	if b.read+3 > len(b.data) { return 0 }
	v := int32(b.data[b.read])<<16 | int32(b.data[b.read+1])<<8 | int32(b.data[b.read+2])
	b.read += 3
	return v
}

func (b *Buffer) ReadGByte() uint8 { return uint8(b.ReadByte()) }

func (b *Buffer) ReadGShort() uint16 {
	if b.read >= len(b.data) { return 0 }
	first := b.data[b.read]
	b.read++
	if first&0x80 == 0 { return uint16(first) }
	if b.read >= len(b.data) { return 0 }
	second := b.data[b.read]
	b.read++
	return uint16(first&0x7F)<<8 | uint16(second)
}

func (b *Buffer) ReadGInt() uint32 {
	if b.read >= len(b.data) { return 0 }
	first := b.data[b.read]
	b.read++
	if first&0x80 == 0 { return uint32(first) }
	if b.read >= len(b.data) { return 0 }
	second := b.data[b.read]
	b.read++
	if second&0x80 == 0 { return uint32(first&0x7F)<<8 | uint32(second) }
	if b.read >= len(b.data) { return 0 }
	third := b.data[b.read]
	b.read++
	return uint32(first&0x7F)<<16 | uint32(second)<<8 | uint32(third)
}

func (b *Buffer) ReadGInt4() uint32 {
	if b.read+4 > len(b.data) { return 0 }
	b.read++
	v := uint32(b.data[b.read])<<16 | uint32(b.data[b.read+1])<<8 | uint32(b.data[b.read+2])
	b.read += 3
	return v
}

func (b *Buffer) ReadGInt5() uint64 {
	if b.read+5 > len(b.data) { return 0 }
	v := uint64(b.data[b.read])<<32 | uint64(b.data[b.read+1])<<24 | uint64(b.data[b.read+2])<<16 | uint64(b.data[b.read+3])<<8 | uint64(b.data[b.read+4])
	b.read += 5
	return v
}

func (b *Buffer) ReadGString() string {
	strLen := b.ReadGInt()
	if b.read+int(strLen) > len(b.data) { return "" }
	start := b.read
	b.read += int(strLen)
	return string(b.data[start:b.read])
}

func (b *Buffer) ReadString() string {
	start := b.read
	for b.read < len(b.data) && b.data[b.read] != 0 { b.read++ }
	s := string(b.data[start:b.read])
	if b.read < len(b.data) { b.read++ }
	return s
}

func (b *Buffer) Bytes() []byte { return b.data }
func (b *Buffer) Len() int { return len(b.data) }
func (b *Buffer) Remaining() int { rem := len(b.data) - b.read; if rem < 0 { return 0 }; return rem }
func (b *Buffer) BytesLeft() int { return b.Remaining() }
func (b *Buffer) ReadGChar() uint8 {
	v := b.ReadGByte()
	if v < 32 { return 0 }
	return v - 32
}
func (b *Buffer) ReadBytes(count int) []byte {
	result := make([]byte, count)
	for i := 0; i < count; i++ {
		result[i] = b.ReadByte()
	}
	return result
}
func (b *Buffer) Reset() { b.read = 0 }
func (b *Buffer) Clear() { b.data = b.data[:0]; b.read = 0; b.write = 0 }

func (b *Buffer) Base64Encode() *Buffer {
	encoded := base64.StdEncoding.EncodeToString(b.data)
	b.data = []byte(encoded)
	b.read = 0
	b.write = len(b.data)
	return b
}

func (b *Buffer) Base64Decode() *Buffer {
	decoded, err := base64.StdEncoding.DecodeString(string(b.data))
	if err != nil { return b }
	b.data = decoded
	b.read = 0
	b.write = len(b.data)
	return b
}

// Encryption
const (
	ENCRYPT_GEN_1 = 0; ENCRYPT_GEN_2 = 1; ENCRYPT_GEN_3 = 2
	ENCRYPT_GEN_4 = 3; ENCRYPT_GEN_5 = 4; ENCRYPT_GEN_6 = 5
	COMPRESS_UNCOMPRESSED = 0x02; COMPRESS_ZLIB = 0x04; COMPRESS_BZ2 = 0x06
)

var ITERATOR_START = [6]uint32{0, 0, 0x04A80B38, 0x4A80B38, 0x4A80B38, 0}

type Encryption struct { key byte; iterator uint32; limit int32; gen uint32 }

func NewEncryption() *Encryption {
	return &Encryption{key: 0, limit: -1, gen: ENCRYPT_GEN_3, iterator: ITERATOR_START[ENCRYPT_GEN_3]}
}

func (e *Encryption) Reset(key byte) {
	e.key = key
	e.iterator = ITERATOR_START[e.gen]
	e.limit = -1
}

func (e *Encryption) SetGen(gen uint32) {
	if e.gen > 6 { e.gen = 6 } else { e.gen = gen }
	e.iterator = ITERATOR_START[e.gen]
}

func (e *Encryption) GetGen() uint32 { return e.gen }
func (e *Encryption) SetLimit(limit int32) { e.limit = limit }

func (e *Encryption) LimitFromType(packetType byte) int {
	limits := []int{0x02, 0x0C, 0x04, 0x04, 0x06, 0x04}
	for i := 0; i < len(limits); i += 2 {
		if limits[i] == int(packetType) { e.limit = int32(limits[i+1]); return 0 }
	}
	return 1
}

func (e *Encryption) Decrypt(data []byte) {
	if len(data) == 0 { return }
	switch e.gen {
	case ENCRYPT_GEN_1, ENCRYPT_GEN_2:
		return
	case ENCRYPT_GEN_3:
		e.iterator = e.iterator*0x8088405 + uint32(e.key)
		pos := (e.iterator & 0xFFFF) % uint32(len(data))
		copy(data[pos:], data[pos+1:])
		data = data[:len(data)-1]
	case ENCRYPT_GEN_4, ENCRYPT_GEN_5:
		for i := 0; i < len(data); i++ {
			if i%4 == 0 {
				if e.limit == 0 { return }
				e.iterator = e.iterator*0x8088405 + uint32(e.key)
				if e.limit > 0 { e.limit-- }
			}
			iteratorBytes := []byte{byte(e.iterator), byte(e.iterator>>8), byte(e.iterator>>16), byte(e.iterator>>24)}
			data[i] ^= iteratorBytes[i%4]
		}
	case ENCRYPT_GEN_6:
		return
	}
}

func (e *Encryption) Encrypt(data []byte) []byte {
	if len(data) == 0 { return data }
	result := make([]byte, len(data))
	copy(result, data)
	switch e.gen {
	case ENCRYPT_GEN_1, ENCRYPT_GEN_2:
		return result
	case ENCRYPT_GEN_3:
		e.iterator = e.iterator*0x8088405 + uint32(e.key)
		pos := (e.iterator & 0xFFFF) % uint32(len(result))
		result = append(result[:pos+1], result[pos:]...)
		result[pos] = ')'
		return result
	case ENCRYPT_GEN_4, ENCRYPT_GEN_5:
		for i := 0; i < len(result); i++ {
			if i%4 == 0 {
				if e.limit == 0 { return result }
				e.iterator = e.iterator*0x8088405 + uint32(e.key)
				if e.limit > 0 { e.limit-- }
			}
			iteratorBytes := []byte{byte(e.iterator), byte(e.iterator>>8), byte(e.iterator>>16), byte(e.iterator>>24)}
			result[i] ^= iteratorBytes[i%4]
		}
		return result
	}
	return result
}

// Socket management
type SocketStub interface {
	OnRecv() bool; OnSend() bool; OnRegister() bool; OnUnregister()
	CanRecv() bool; CanSend() bool
}

type SocketManager struct { stubs map[net.Conn]SocketStub; mu sync.RWMutex; running bool }

func NewSocketManager() *SocketManager { return &SocketManager{stubs: make(map[net.Conn]SocketStub), running: false} }

func (sm *SocketManager) Register(conn net.Conn, stub SocketStub) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stubs[conn] = stub
	stub.OnRegister()
}

func (sm *SocketManager) Unregister(conn net.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if stub, ok := sm.stubs[conn]; ok { stub.OnUnregister(); delete(sm.stubs, conn); conn.Close() }
}

func (sm *SocketManager) Update(timeout time.Duration) bool {
	sm.mu.RLock()
	if len(sm.stubs) == 0 { sm.mu.RUnlock(); time.Sleep(timeout); return false }
	sm.mu.RUnlock()
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	handled := false
	for _, stub := range sm.stubs {
		if stub.CanRecv() { stub.OnRecv(); handled = true }
		if stub.CanSend() { stub.OnSend(); handled = true }
	}
	return handled
}

func (sm *SocketManager) Cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for conn, stub := range sm.stubs { stub.OnUnregister(); conn.Close(); delete(sm.stubs, conn) }
}

func ZlibDecompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil { return nil, err }
	defer reader.Close()
	return io.ReadAll(reader)
}
func ZlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes(), nil
}
