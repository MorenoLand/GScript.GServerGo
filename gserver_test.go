package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestSendPacketGen5LengthExcludesLengthPrefix(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:          serverConn,
		server:        &Server{logger: NewLogger("", false)},
		encryption:    *NewEncryption(),
		outEncryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.Reset(0)

	done := make(chan error, 1)
	go func() {
		p.sendPacket([]byte{PLO_SIGNATURE, 73})
		done <- nil
	}()

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	frame := make([]byte, 5)
	if _, err := io.ReadFull(clientConn, frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("send packet: %v", err)
	}

	wireLen := int(frame[0]) | int(frame[1])<<8
	if wireLen != len(frame)-2 {
		t.Fatalf("GEN_5 length prefix = %d, want %d", wireLen, len(frame)-2)
	}
}

func TestSendPacketGen5EncodesOutgoingPacketID(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:          serverConn,
		server:        &Server{logger: NewLogger("", false)},
		encryption:    *NewEncryption(),
		outEncryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.Reset(0)

	done := make(chan error, 1)
	go func() {
		p.sendPacket([]byte{PLO_SIGNATURE, 73})
		done <- nil
	}()

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	frame := make([]byte, 5)
	if _, err := io.ReadFull(clientConn, frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("send packet: %v", err)
	}

	payload := append([]byte(nil), frame[3:]...)
	in := *NewEncryption()
	in.SetGen(ENCRYPT_GEN_5)
	in.Reset(0)
	in.LimitFromType(frame[2])
	in.Decrypt(payload)

	if payload[0] != PLO_SIGNATURE+32 {
		t.Fatalf("plaintext packet ID = 0x%02X, want encoded 0x%02X", payload[0], PLO_SIGNATURE+32)
	}
}

func TestSendCompressFlushesQueuedPacketsAsOneGen5Frame(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:          serverConn,
		server:        &Server{logger: NewLogger("", false)},
		encryption:    *NewEncryption(),
		outEncryption: *NewEncryption(),
		queueOutgoing: true,
	}
	p.encryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.Reset(0)

	p.sendPacket([]byte{PLO_SIGNATURE, 73, '\n'})
	p.sendPacket([]byte{PLO_CLEARWEAPONS, '\n'})

	done := make(chan struct{}, 1)
	go func() {
		p.sendCompress(true)
		done <- struct{}{}
	}()

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	header := make([]byte, 3)
	if _, err := io.ReadFull(clientConn, header); err != nil {
		t.Fatalf("read frame header: %v", err)
	}
	frameLen := int(header[0]) | int(header[1])<<8
	if frameLen != 1+len([]byte{PLO_SIGNATURE + 32, 73, '\n', PLO_CLEARWEAPONS + 32, '\n'}) {
		t.Fatalf("GEN_5 frame length = %d, want one uncompressed queued frame", frameLen)
	}
	encrypted := make([]byte, frameLen-1)
	if _, err := io.ReadFull(clientConn, encrypted); err != nil {
		t.Fatalf("read frame payload: %v", err)
	}
	<-done

	in := *NewEncryption()
	in.SetGen(ENCRYPT_GEN_5)
	in.Reset(0)
	in.LimitFromType(header[2])
	in.Decrypt(encrypted)

	want := []byte{PLO_SIGNATURE + 32, 73, '\n', PLO_CLEARWEAPONS + 32, '\n'}
	if string(encrypted) != string(want) {
		t.Fatalf("queued plaintext = % X, want % X", encrypted, want)
	}
}

func TestSendPacketGen5UsesBz2ForLargeFrames(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:          serverConn,
		server:        &Server{logger: NewLogger("", false)},
		encryption:    *NewEncryption(),
		outEncryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.SetGen(ENCRYPT_GEN_5)
	p.outEncryption.Reset(0)

	packet := append([]byte{PLO_RAWDATA}, make([]byte, 0x2001)...)
	done := make(chan struct{}, 1)
	go func() {
		p.sendPacket(packet)
		done <- struct{}{}
	}()

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	header := make([]byte, 3)
	if _, err := io.ReadFull(clientConn, header); err != nil {
		t.Fatalf("read frame header: %v", err)
	}
	if header[2] != COMPRESS_BZ2 {
		t.Fatalf("compression type = 0x%02X, want BZ2 0x%02X", header[2], COMPRESS_BZ2)
	}
	encrypted := make([]byte, (int(header[0])|int(header[1])<<8)-1)
	if _, err := io.ReadFull(clientConn, encrypted); err != nil {
		t.Fatalf("read frame payload: %v", err)
	}
	<-done

	in := *NewEncryption()
	in.SetGen(ENCRYPT_GEN_5)
	in.Reset(0)
	in.LimitFromType(header[2])
	in.Decrypt(encrypted)
	decompressed, err := Bz2Decompress(encrypted)
	if err != nil {
		t.Fatalf("decompress bz2 payload: %v", err)
	}
	if decompressed[0] != PLO_RAWDATA+32 || len(decompressed) != len(packet) {
		t.Fatalf("decompressed large frame len/id = %d/0x%02X, want %d/0x%02X", len(decompressed), decompressed[0], len(packet), PLO_RAWDATA+32)
	}
}

func TestHandlePacketDecodesGraalEncodedPacketID(t *testing.T) {
	p := &Player{
		server: &Server{logger: NewLogger("", false)},
	}

	if !p.handlePacket([]byte{byte(PLI_LANGUAGE + 32), 'E', 'n', 'g', 'l', 'i', 's', 'h'}) {
		t.Fatal("handlePacket returned false")
	}
	if p.language != "English" {
		t.Fatalf("language = %q, want English", p.language)
	}
}

func TestHandleDecompressedPacketsSplitsNewlineDelimitedClientPackets(t *testing.T) {
	p := &Player{
		server: &Server{logger: NewLogger("", false)},
	}

	p.handleDecompressedPackets([]byte{
		byte(PLI_LANGUAGE + 32), 'E', 'n', 'g', 'l', 'i', 's', 'h', '\n',
		byte(PLI_LANGUAGE + 32), 'F', 'r', 'e', 'n', 'c', 'h', '\n',
	})

	if p.language != "French" {
		t.Fatalf("language = %q, want French", p.language)
	}
}

func TestSendPropsWithArrayEncodesPropertyIDs(t *testing.T) {
	p := &Player{}
	p.character.nickName = "moondeath"

	var props [PROPCOUNT]bool
	props[PLPROP_NICKNAME] = true

	got := p.sendPropsWithArray(props)
	want := []byte{byte(PLPROP_NICKNAME + 32), byte(len("moondeath") + 32)}
	want = append(want, []byte("moondeath")...)

	if string(got) != string(want) {
		t.Fatalf("props bytes = % X, want % X", got, want)
	}
}

func TestLevelBoardPacketUsesEncodedPacketID(t *testing.T) {
	level := NewLevel()
	level.tiles[0] = &LevelTiles{width: 64, height: 64, tiles: make([]int16, 4096)}
	level.tiles[0].tiles[0] = 0x1234
	got := level.getBoardPacket()

	if len(got) != 8194 {
		t.Fatalf("board packet length = %d, want 8194", len(got))
	}
	if got[0] != PLO_BOARDPACKET+32 {
		t.Fatalf("board packet ID = 0x%02X, want encoded 0x%02X", got[0], PLO_BOARDPACKET+32)
	}
	if got[1] != 0x34 || got[2] != 0x12 {
		t.Fatalf("first board tile bytes = %02X %02X, want little-endian 34 12", got[1], got[2])
	}
	if got[len(got)-1] != '\n' {
		t.Fatalf("board packet missing newline terminator")
	}
}

func TestPlayerWarpUsesClientWireFormat(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_PLAYERWARP(32, 32, 0, "onlinestartlocal.nw")
		done <- struct{}{}
	}()

	want := append([]byte{PLO_PLAYERWARP + 32, 32*2 + 32, 32*2 + 32}, []byte("onlinestartlocal.nw\n")...)

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read warp packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("warp packet = % X, want % X", got, want)
	}
}

func TestNPCWeaponDelUsesRawNameWithoutNul(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_NPCWEAPONDEL("Bomb")
		done <- struct{}{}
	}()

	want := []byte{PLO_NPCWEAPONDEL + 32, 'B', 'o', 'm', 'b', '\n'}

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read weapon delete packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("weapon delete packet = % X, want % X", got, want)
	}
}

func TestLevelModTimeUsesGInt5WireFormat(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	modTime := int64(1712345678)
	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_LEVELMODTIME(modTime)
		done <- struct{}{}
	}()

	expectedTime := NewBuffer()
	expectedTime.WriteGInt5(uint64(modTime))
	want := append([]byte{PLO_LEVELMODTIME + 32}, expectedTime.Bytes()...)
	want = append(want, '\n')

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read level modtime packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("level modtime packet = % X, want % X", got, want)
	}
}

func TestNewWorldTimeUsesGInt4WireFormat(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	worldTime := uint(0x123456)
	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_NEWWORLDTIME(worldTime)
		done <- struct{}{}
	}()

	want := []byte{PLO_NEWWORLDTIME + 32, 0x20, 0x68, 0x88, 0x76, '\n'}

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read new world time packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("new world time packet = % X, want % X", got, want)
	}
}

func TestGInt4MatchesCStringWireFormat(t *testing.T) {
	buf := NewBuffer()
	buf.WriteGInt4(0xAB4)

	want := []byte{0x20, 0x20, 0x35, 0x54}
	if string(buf.Bytes()) != string(want) {
		t.Fatalf("GInt4 bytes = % X, want % X", buf.Bytes(), want)
	}

	got := NewBufferFromBytes(want).ReadGInt4()
	if got != 0xAB4 {
		t.Fatalf("ReadGInt4 = %X, want AB4", got)
	}
}

func TestGhostIconUsesSingleBytePayload(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_GHOSTICON(false)
		done <- struct{}{}
	}()

	want := []byte{PLO_GHOSTICON + 32, 0, '\n'}

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read ghost icon packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("ghost icon packet = % X, want % X", got, want)
	}
}

func TestRpgWindowUsesCStringCompatibleTextPayload(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	message := "\"Welcome to Orion.\",\"Go Code GServer.\""
	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_RPGWINDOW(message)
		done <- struct{}{}
	}()

	want := append([]byte{PLO_RPGWINDOW + 32}, []byte(message)...)
	want = append(want, '\n')

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read rpg window packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("rpg window packet = % X, want % X", got, want)
	}
}

func TestStartMessageUsesRawConfiguredMessage(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	p := &Player{
		conn:       serverConn,
		server:     &Server{logger: NewLogger("", false)},
		encryption: *NewEncryption(),
	}
	p.encryption.SetGen(ENCRYPT_GEN_1)

	message := "<html><body>Welcome</body></html>"
	done := make(chan struct{}, 1)
	go func() {
		p.sendPLO_STARTMESSAGE(message)
		done <- struct{}{}
	}()

	want := append([]byte{PLO_STARTMESSAGE + 32}, []byte(message)...)
	want = append(want, '\n')

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read start message packet: %v", err)
	}
	<-done

	if string(got) != string(want) {
		t.Fatalf("start message packet = % X, want % X", got, want)
	}
}
