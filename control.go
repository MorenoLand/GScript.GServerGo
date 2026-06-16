package main

import "strings"

func rcChatPacket(message string) []byte {
	buf := NewBuffer()
	buf.WriteByte(PLO_RC_CHAT).Write([]byte(message))
	return buf.Bytes()
}

func rcCommandAccountPacket(account string) []byte {
	return NewBuffer().WriteGString(strings.TrimSpace(account)).Bytes()
}

func isRCOnlyPacket(packetId int) bool {
	if packetId >= PLI_RC_SERVEROPTIONSGET && packetId <= PLI_RC_FILEBROWSER_RENAME {
		return packetId != PLI_PROFILEGET && packetId != PLI_PROFILESET
	}
	return packetId == PLI_RC_FOLDERDELETE || packetId == PLI_RC_UNKNOWN162
}

func isNCOnlyPacket(packetId int) bool {
	return packetId == PLI_NC_LISTNPCS ||
		(packetId >= PLI_NC_NPCGET && packetId <= PLI_NC_CLASSDELETE) ||
		packetId == PLI_NC_LEVELLISTGET ||
		packetId == PLI_NC_LEVELLISTSET
}
