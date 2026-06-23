package main

import (
	"strconv"
	"strings"
	"time"
)

type cachedListserverServer struct {
	Name         string
	Type         string
	PlayerCount  int
	Language     string
	Description  string
	URL          string
	Version      string
	GameVersions string
	Latency      int
	Updated      time.Time
}

type listserverEndpoint struct {
	host string
	port string
}

func (s *Server) configureServerLists() {
	enabled := s.settings.GetBool("listserver", true)
	endpoints := s.listserverEndpoints()
	if len(endpoints) == 0 {
		endpoints = []listserverEndpoint{{host: "listserver.graal.in", port: "14900"}}
	}
	s.serverLists = make([]*ServerList, 0, len(endpoints))
	for _, endpoint := range endpoints {
		serverList := NewServerListEndpoint(s, endpoint.host, endpoint.port)
		serverList.enabled = enabled
		s.serverLists = append(s.serverLists, serverList)
	}
	s.serverList = s.serverLists[0]
}

func (s *Server) listserverEndpoints() []listserverEndpoint {
	hosts := splitCommaList(s.settings.Get("listip"))
	ports := splitCommaList(s.settings.Get("listport"))
	if len(hosts) == 0 {
		return nil
	}
	if len(ports) == 0 {
		ports = []string{"14900"}
	}
	endpoints := make([]listserverEndpoint, 0, len(hosts))
	for i, host := range hosts {
		port := ports[0]
		if i < len(ports) {
			port = ports[i]
		}
		endpoints = append(endpoints, listserverEndpoint{host: host, port: port})
	}
	return endpoints
}

func (s *Server) sendPlayerTextToListservers(packetId byte, playerID uint16, text string) bool {
	if s == nil {
		return false
	}
	sent := false
	seen := make(map[*ServerList]bool)
	for _, serverList := range s.serverLists {
		if serverList == nil || seen[serverList] {
			continue
		}
		seen[serverList] = true
		if !serverList.connected {
			continue
		}
		serverList.SendPlayerTextPacket(packetId, playerID, text)
		sent = true
	}
	if s.serverList != nil && !seen[s.serverList] && s.serverList.connected {
		s.serverList.SendPlayerTextPacket(packetId, playerID, text)
		sent = true
	}
	return sent
}

func (s *Server) sendTextToListservers(packetId byte, text string) bool {
	if s == nil {
		return false
	}
	sent := false
	seen := make(map[*ServerList]bool)
	for _, serverList := range s.serverLists {
		if serverList == nil || seen[serverList] {
			continue
		}
		seen[serverList] = true
		if !serverList.connected {
			continue
		}
		serverList.SendTextPacket(packetId, text)
		sent = true
	}
	if s.serverList != nil && !seen[s.serverList] && s.serverList.connected {
		s.serverList.SendTextPacket(packetId, text)
		sent = true
	}
	return sent
}

func (s *Server) sendLoginPacketToListservers(player *Player, password, identity string) bool {
	if s == nil || player == nil {
		return false
	}
	sent := false
	seen := make(map[*ServerList]bool)
	for _, serverList := range s.serverLists {
		if serverList == nil || seen[serverList] {
			continue
		}
		seen[serverList] = true
		if !serverList.connected {
			continue
		}
		serverList.SendLoginPacketForPlayer(player, password, identity)
		sent = true
	}
	if s.serverList != nil && !seen[s.serverList] && s.serverList.connected {
		s.serverList.SendLoginPacketForPlayer(player, password, identity)
		sent = true
	}
	return sent
}

func (s *Server) addPlayerToListservers(player *Player) {
	if s == nil || player == nil {
		return
	}
	seen := make(map[*ServerList]bool)
	for _, serverList := range s.serverLists {
		if serverList == nil || seen[serverList] {
			continue
		}
		seen[serverList] = true
		serverList.AddPlayer(player)
	}
	if s.serverList != nil && !seen[s.serverList] {
		s.serverList.AddPlayer(player)
	}
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (s *Server) cacheListserverText(data []byte) {
	if s == nil || len(data) == 0 {
		return
	}
	text := strings.TrimSpace(strings.TrimRight(string(data), "\x00"))
	if text == "" {
		return
	}
	if strings.HasPrefix(strings.ToLower(text), "listserver,modify,server,") {
		s.cacheListserverModify(text)
		return
	}
	for _, record := range listserverRecords(text) {
		s.cacheListserverRecord(record)
	}
}

func (s *Server) cacheListserverModify(text string) {
	fields := strings.Split(text, ",")
	if len(fields) < 4 {
		return
	}
	name := strings.TrimSpace(fields[3])
	if name == "" {
		return
	}
	server := cachedListserverServer{Name: name, Updated: time.Now()}
	s.listserverMu.RLock()
	if s.listserverCache != nil {
		if existing, ok := s.listserverCache[strings.ToLower(name)]; ok {
			server = existing
			server.Updated = time.Now()
		}
	}
	s.listserverMu.RUnlock()
	for _, field := range fields[4:] {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok {
			continue
		}
		applyListserverServerField(&server, key, value)
	}
	s.storeListserverServer(server)
}

func (s *Server) cacheListserverRecord(record string) {
	fields := splitListserverFields(record)
	if len(fields) < 3 {
		return
	}
	name := strings.TrimSpace(fields[0])
	if name == "" || strings.EqualFold(name, "Listserver") || strings.EqualFold(name, "GraalEngine") {
		return
	}
	server := cachedListserverServer{Name: name, Updated: time.Now()}
	if len(fields) > 1 {
		server.Type = fields[1]
	}
	if len(fields) > 2 {
		server.PlayerCount, _ = strconv.Atoi(strings.TrimSpace(fields[2]))
	}
	if len(fields) > 3 {
		server.Language = fields[3]
	}
	if len(fields) > 4 {
		server.Description = fields[4]
	}
	if len(fields) > 5 {
		server.URL = fields[5]
	}
	if len(fields) > 6 {
		server.Version = fields[6]
	}
	if len(fields) > 7 {
		server.GameVersions = fields[7]
	}
	if len(fields) > 8 {
		server.Latency, _ = strconv.Atoi(strings.TrimSpace(fields[8]))
	}
	s.storeListserverServer(server)
}

func (s *Server) storeListserverServer(server cachedListserverServer) {
	if strings.TrimSpace(server.Name) == "" {
		return
	}
	s.listserverMu.Lock()
	defer s.listserverMu.Unlock()
	if s.listserverCache == nil {
		s.listserverCache = make(map[string]cachedListserverServer)
	}
	s.listserverCache[strings.ToLower(server.Name)] = server
}

func listserverRecords(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if strings.Contains(text, "\x01") {
		text = strings.ReplaceAll(text, "\x01", "\n")
	}
	parts := strings.Split(text, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitListserverFields(record string) []string {
	if strings.Contains(record, "\x01") {
		return splitAndTrim(record, "\x01")
	}
	if strings.Contains(record, ",") {
		return splitAndTrim(guntokenizeText(record), "\n")
	}
	return nil
}

func splitAndTrim(value, sep string) []string {
	parts := strings.Split(value, sep)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func applyListserverServerField(server *cachedListserverServer, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "name":
		server.Name = value
	case "type":
		server.Type = value
	case "players", "playercount":
		server.PlayerCount, _ = strconv.Atoi(strings.TrimSpace(value))
	case "language":
		server.Language = value
	case "description", "desc":
		server.Description = value
	case "url", "website":
		server.URL = value
	case "version", "serverversion":
		server.Version = value
	case "gameversions", "allowedversions":
		server.GameVersions = value
	case "latency", "ping":
		server.Latency, _ = strconv.Atoi(strings.TrimSpace(value))
	}
}
