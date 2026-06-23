package main

import (
	"crypto/rand"
	"strings"
	"time"

	gs2parser "github.com/MorenoLand/GScript.gs2parser-go"
)

type gs2CompileResult struct {
	bytecode    []byte
	errText     string
	warningText string
}

func (s *Server) compileGS2ForFeedback(scriptType, scriptName, script string) gs2CompileResult {
	if s == nil {
		return gs2CompileResult{}
	}
	src, ok := clientsideGS2(script)
	if !ok {
		return gs2CompileResult{}
	}
	res := gs2parser.CompileDetailed(src)
	if len(res.Diagnostics) != 0 {
		return gs2CompileResult{errText: gs2DiagnosticsText(res.Diagnostics)}
	}
	return gs2CompileResult{bytecode: gs2BytecodeWithHeader(res.Bytecode, scriptType, scriptName, true)}
}

func gs2DiagnosticsText(diagnostics []gs2parser.Diagnostic) string {
	lines := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		lines = append(lines, diagnostic.Error())
	}
	return strings.Join(lines, "\n")
}

func gs2BytecodeWithHeader(bytecode []byte, scriptType, scriptName string, saveToDisk bool) []byte {
	if len(bytecode) == 0 {
		return nil
	}
	if _, ok := gs2BytecodeHeader(bytecode); ok {
		return bytecode
	}
	headerLen := len(scriptType) + len(scriptName) + 14
	buf := NewBuffer()
	buf.WriteGShort(uint16(headerLen))
	buf.Write([]byte(scriptType))
	buf.WriteByte(',')
	buf.Write([]byte(scriptName))
	buf.WriteByte(',')
	if saveToDisk {
		buf.WriteByte('1')
	} else {
		buf.WriteByte('0')
	}
	buf.WriteByte(',')
	key := gs2HeaderKey()
	buf.Write(key[:])
	buf.Write(bytecode)
	return buf.Bytes()
}

func gs2HeaderKey() [10]byte {
	var key [10]byte
	if _, err := rand.Read(key[:]); err != nil {
		seed := uint64(time.Now().UnixNano())
		for i := range key {
			seed = seed*1664525 + 1013904223
			key[i] = byte(seed % 255)
		}
	}
	for i := range key {
		key[i] %= 255
		if key[i] < 223 {
			key[i] += 32
		}
	}
	return key
}

func clientsideGS2(script string) (string, bool) {
	if clientsideScriptIsGS1(script) {
		return "", false
	}
	const marker = "//#CLIENTSIDE"
	idx := strings.Index(strings.ToUpper(script), marker)
	if idx < 0 {
		return "", false
	}
	return strings.TrimSpace(script[idx+len(marker):]), true
}
