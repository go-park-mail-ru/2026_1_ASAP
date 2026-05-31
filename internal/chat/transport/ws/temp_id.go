package ws

import dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"

func stripTempIDForViewer(presented *dto.ResponseSendMessage, viewerID, senderID int64) {
	if presented == nil {
		return
	}
	if viewerID != senderID {
		presented.TempID = ""
	}
}
