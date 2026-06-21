package main

import (
	"net"
	"strconv"
	"strings"
	"sync"

	nativegs2vm "github.com/MorenoLand/GScript.gs2vm-go"
)

type GS2SocketManager struct {
	server  *Server
	mu      sync.Mutex
	sockets map[string]*GS2Socket
}

type GS2Socket struct {
	key              string
	name             string
	id               string
	port             int
	conn             net.Conn
	listener         net.Listener
	packageDelimiter string
	buffer           string
	result           gs2VMResult
	closed           bool
}

func NewGS2SocketManager(server *Server) *GS2SocketManager {
	return &GS2SocketManager{server: server, sockets: make(map[string]*GS2Socket)}
}

func (m *GS2SocketManager) Apply(result gs2VMResult) {
	if m == nil {
		return
	}
	for _, update := range result.socketUpdates {
		m.update(update)
	}
	for _, action := range result.socketActions {
		switch strings.ToLower(action.action) {
		case "bind":
			m.bind(result, action)
		case "close":
			m.close(result, action)
		case "send":
			m.send(result, action)
		}
	}
}

func (m *GS2SocketManager) bind(result gs2VMResult, action gs2VMSocketAction) {
	if action.port <= 0 || action.name == "" {
		return
	}
	key := m.key(result, action.name, "")
	m.closeKey(key)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(action.port))
	if err != nil {
		m.fire(result, action.name, "", "onBindFailed", socketState(action.name, "", "", action.port, action.packageDelimiter, "", false))
		return
	}
	socket := &GS2Socket{key: key, name: action.name, port: action.port, listener: listener, packageDelimiter: action.packageDelimiter, result: result}
	m.mu.Lock()
	m.sockets[key] = socket
	m.mu.Unlock()
	m.fire(result, action.name, "", "onBind", socketState(action.name, "", "", action.port, action.packageDelimiter, "", true))
	go m.acceptLoop(socket)
}

func (m *GS2SocketManager) acceptLoop(serverSocket *GS2Socket) {
	for {
		conn, err := serverSocket.listener.Accept()
		if err != nil {
			return
		}
		id := conn.RemoteAddr().String()
		host, _, _ := net.SplitHostPort(id)
		clientKey := serverSocket.key + ":" + id
		client := &GS2Socket{key: clientKey, name: serverSocket.name, id: id, conn: conn, port: serverSocket.port, result: serverSocket.result}
		m.mu.Lock()
		m.sockets[clientKey] = client
		m.mu.Unlock()
		m.fire(serverSocket.result, serverSocket.name, id, "onNewClient", socketState(serverSocket.name, id, host, serverSocket.port, "", "", true))
		go m.readLoop(client)
	}
}

func (m *GS2SocketManager) readLoop(socket *GS2Socket) {
	buf := make([]byte, 4096)
	for {
		n, err := socket.conn.Read(buf)
		if err != nil || n == 0 {
			m.closeSocket(socket)
			m.fire(socket.result, socket.name, socket.id, "onClose", socketState(socket.name, socket.id, "", socket.port, socket.packageDelimiter, socket.buffer, false))
			return
		}
		chunk := string(buf[:n])
		m.mu.Lock()
		socket.buffer += chunk
		delimiter := socket.packageDelimiter
		m.mu.Unlock()
		if delimiter == "" {
			m.fire(socket.result, socket.name, socket.id, "onReceiveData", socketState(socket.name, socket.id, "", socket.port, delimiter, socket.buffer, true), chunk)
			continue
		}
		for {
			m.mu.Lock()
			idx := strings.Index(socket.buffer, delimiter)
			if idx < 0 {
				m.mu.Unlock()
				break
			}
			packet := socket.buffer[:idx]
			socket.buffer = socket.buffer[idx+len(delimiter):]
			m.mu.Unlock()
			m.fire(socket.result, socket.name, socket.id, "onReceiveDataPackage", socketState(socket.name, socket.id, "", socket.port, delimiter, socket.buffer, true), packet)
		}
	}
}

func (m *GS2SocketManager) fire(base gs2VMResult, name, id, event string, socket nativegs2vm.SocketContext, params ...string) {
	if m == nil || m.server == nil {
		return
	}
	this := copyAnyMap(base.this)
	this["name"] = name
	this["port"] = socket.Port
	result := m.server.runServerSideGS2NativeWithStateAndSocket(base.scriptType, base.scriptName, event, base.script, this, base.playerContext, base.npcID, &socket, params...)
	if result.err != "" {
		m.server.sendGS2VMErrorToNC(base.scriptType+" "+base.scriptName, result.err)
		return
	}
	m.server.applyGS2VMResult(result)
	m.server.emitGS2VMOutput(result)
}

func (m *GS2SocketManager) update(update gs2VMSocketUpdate) {
	if update.id == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, socket := range m.sockets {
		if socket.id == update.id {
			socket.packageDelimiter = update.packageDelimiter
			socket.buffer = update.data
		}
	}
}

func (m *GS2SocketManager) send(result gs2VMResult, action gs2VMSocketAction) {
	socket := m.find(result, action.name, action.id)
	if socket == nil || socket.conn == nil {
		return
	}
	_, _ = socket.conn.Write([]byte(action.data))
}

func (m *GS2SocketManager) close(result gs2VMResult, action gs2VMSocketAction) {
	if action.id == "" {
		m.closeKey(m.key(result, action.name, ""))
		return
	}
	if socket := m.find(result, action.name, action.id); socket != nil {
		m.closeSocket(socket)
	}
}

func (m *GS2SocketManager) closeKey(key string) {
	m.mu.Lock()
	socket := m.sockets[key]
	delete(m.sockets, key)
	m.mu.Unlock()
	if socket != nil {
		m.closeSocket(socket)
	}
}

func (m *GS2SocketManager) closeSocket(socket *GS2Socket) {
	m.mu.Lock()
	if socket.closed {
		m.mu.Unlock()
		return
	}
	socket.closed = true
	delete(m.sockets, socket.key)
	m.mu.Unlock()
	if socket.conn != nil {
		_ = socket.conn.Close()
	}
	if socket.listener != nil {
		_ = socket.listener.Close()
	}
}

func (m *GS2SocketManager) find(result gs2VMResult, name, id string) *GS2Socket {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		return m.sockets[m.key(result, name, "")]
	}
	for _, socket := range m.sockets {
		if socket.name == name && socket.id == id {
			return socket
		}
	}
	return nil
}

func (m *GS2SocketManager) key(result gs2VMResult, name, id string) string {
	return result.scriptType + ":" + result.scriptName + ":" + name + ":" + id
}

func (m *GS2SocketManager) CloseAll() {
	m.mu.Lock()
	sockets := make([]*GS2Socket, 0, len(m.sockets))
	for _, socket := range m.sockets {
		sockets = append(sockets, socket)
	}
	m.sockets = make(map[string]*GS2Socket)
	m.mu.Unlock()
	for _, socket := range sockets {
		if socket.conn != nil {
			_ = socket.conn.Close()
		}
		if socket.listener != nil {
			_ = socket.listener.Close()
		}
	}
}

func socketState(name, id, ip string, port int, delimiter, data string, connected bool) nativegs2vm.SocketContext {
	return nativegs2vm.SocketContext{Name: name, ID: id, IPAddress: ip, Port: port, PackageDelimiter: delimiter, Data: data, IsConnected: connected}
}

func copyAnyMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
