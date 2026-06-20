package main

import (
	"fmt"
	"strconv"
	"strings"

	nativegs2vm "github.com/MorenoLand/GScript.gs2vm-go"
)

type gs2VMResult struct {
	output         []string
	clientTriggers []string
	playerFlags    []gs2VMPlayerFlag
	playerMessages []gs2VMPlayerMessage
	err            string
}

type gs2VMPlayerFlag struct {
	account string
	name    string
	value   string
}

type gs2VMPlayerMessage struct {
	account string
	message string
}

func (s *Server) runServerSideGS2(scriptType, scriptName, eventName, script string, eventArgs ...string) gs2VMResult {
	return s.runServerSideGS2Native(scriptType, scriptName, eventName, script, nil, eventArgs...)
}

func (s *Server) runServerSideGS2Native(scriptType, scriptName, eventName, script string, playerContext map[string]string, eventArgs ...string) gs2VMResult {
	src := serversideGS2(script)
	if strings.TrimSpace(src) == "" {
		return gs2VMResult{}
	}
	if playerContext == nil {
		playerContext = make(map[string]string)
	}
	result := nativegs2vm.Run(nativegs2vm.Config{
		ScriptName:    scriptName,
		EventName:     eventName,
		Script:        src,
		Params:        eventArgs,
		Player:        playerContext,
		PlayerFlags:   s.snapshotGS2PlayerFlags(playerContext["account"]),
		Players:       s.snapshotGS2Players(),
		ServerFlags:   s.snapshotServerFlags(),
		ServerOptions: s.snapshotServerOptions(),
	})
	out := gs2VMResult{output: result.Output, err: result.Err}
	for _, trigger := range result.ClientTriggers {
		parts := []string{trigger.Name}
		parts = append(parts, trigger.Args...)
		out.clientTriggers = append(out.clientTriggers, "clientside,"+strings.Join(parts, ","))
	}
	for _, flag := range result.PlayerFlags {
		out.playerFlags = append(out.playerFlags, gs2VMPlayerFlag{account: flag.Account, name: flag.Name, value: flag.Value})
	}
	for _, message := range result.PlayerMessages {
		out.playerMessages = append(out.playerMessages, gs2VMPlayerMessage{account: message.Account, message: message.Message})
	}
	return out
}

func snapshotGS2Player(player *Player) map[string]string {
	out := make(map[string]string)
	if player == nil {
		return out
	}
	account := player.accountName
	if player.deviceId > 0 && (account == "" || strings.EqualFold(account, "guest")) {
		account = "pc:" + strconv.FormatInt(player.deviceId, 10)
	}
	out["account"] = account
	out["nick"] = player.character.nickName
	out["nickname"] = player.character.nickName
	out["level"] = player.levelName
	return out
}

func (s *Server) snapshotGS2PlayerFlags(account string) map[string]string {
	if s == nil || account == "" {
		return nil
	}
	if player := s.findGS2Player(account); player != nil {
		return copyStringMap(player.flagList)
	}
	return nil
}

func (s *Server) snapshotGS2Players() []nativegs2vm.PlayerContext {
	if s == nil {
		return nil
	}
	players := s.GetAllPlayers()
	out := make([]nativegs2vm.PlayerContext, 0, len(players))
	for _, player := range players {
		if player == nil || player.accountName == "" || player.playerType&PLTYPE_ANYCLIENT == 0 {
			continue
		}
		out = append(out, nativegs2vm.PlayerContext{Account: gs2PlayerAccount(player), Nick: player.character.nickName, Nickname: player.character.nickName, Level: player.levelName, Flags: copyStringMap(player.flagList)})
	}
	return out
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func gs2PlayerAccount(player *Player) string {
	if player == nil {
		return ""
	}
	if player.deviceId > 0 && (player.accountName == "" || strings.EqualFold(player.accountName, "guest")) {
		return "pc:" + strconv.FormatInt(player.deviceId, 10)
	}
	return player.accountName
}

func (s *Server) findGS2Player(account string) *Player {
	if s == nil || account == "" {
		return nil
	}
	for _, player := range s.GetAllPlayers() {
		if player == nil || player.playerType&PLTYPE_ANYCLIENT == 0 {
			continue
		}
		if strings.EqualFold(player.accountName, account) || strings.EqualFold(gs2PlayerAccount(player), account) || strings.EqualFold(player.character.nickName, account) {
			return player
		}
	}
	return nil
}

func (s *Server) snapshotServerFlags() map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	s.flagMu.RLock()
	defer s.flagMu.RUnlock()
	for key, value := range s.flags {
		out[key] = value
	}
	return out
}

func (s *Server) snapshotServerOptions() map[string]string {
	out := make(map[string]string)
	if s == nil || s.settings == nil {
		return out
	}
	s.settings.mu.RLock()
	defer s.settings.mu.RUnlock()
	for key, value := range s.settings.settings {
		out[key] = value
	}
	return out
}

func serversideGS2(script string) string {
	normalized := strings.ReplaceAll(script, "\r\n", "\n")
	lower := strings.ToLower(normalized)
	idx := strings.Index(lower, "//#clientside")
	if idx >= 0 {
		return strings.TrimSpace(normalized[:idx])
	}
	return normalized
}

func (s *Server) runServerSideWeaponEvent(weapon *Weapon, eventName string) {
	s.runServerSideWeaponEventForPlayer(weapon, eventName, nil)
}

func (s *Server) runServerSideWeaponEventForPlayer(weapon *Weapon, eventName string, player *Player, eventArgs ...string) {
	if s == nil || weapon == nil || weapon.script == "" {
		return
	}
	result := s.runServerSideGS2ForPlayer("weapon", weapon.name, eventName, weapon.script, player, eventArgs...)
	if result.err != "" {
		s.sendToNC(fmt.Sprintf("GS2 VM error for Weapon %s: %s", weapon.name, result.err))
		return
	}
	s.applyGS2VMResult(result)
	if player != nil {
		for _, action := range result.clientTriggers {
			player.sendPLO_TRIGGERACTION(0, 0, 0, 0, action)
		}
	}
	for _, line := range result.output {
		s.logger.Info("[GS2:%s] %s", weapon.name, line)
		s.sendToNC(line)
	}
}

func (s *Server) runServerSideGS2ForPlayer(scriptType, scriptName, eventName, script string, player *Player, eventArgs ...string) gs2VMResult {
	return s.runServerSideGS2Native(scriptType, scriptName, eventName, script, snapshotGS2Player(player), eventArgs...)
}

func (s *Server) applyGS2VMResult(result gs2VMResult) {
	if s == nil {
		return
	}
	for _, flag := range result.playerFlags {
		if player := s.findGS2Player(flag.account); player != nil {
			player.SetFlag(flag.name, flag.value)
			player.sendPLO_FLAGSET(flag.name, flag.value)
		}
	}
	for _, message := range result.playerMessages {
		if player := s.findGS2Player(message.account); player != nil {
			s.sendGS2PlayerPM(player, message.message)
		}
	}
}

func (s *Server) sendGS2PlayerPM(player *Player, message string) {
	if s == nil || player == nil {
		return
	}
	senderId := uint16(1)
	if npcServer := s.ensureNPCServer().Player(); npcServer != nil {
		senderId = npcServer.id
	}
	buf := NewBuffer()
	buf.WriteByte(PLO_PRIVATEMESSAGE).WriteGShort(senderId).Write([]byte("\"\",")).
		Write([]byte(gtokenizeText(message)))
	player.send(buf)
}
