package message

import (
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

func (a MessageAttachmentDTO) toWire() messageAttachmentWire {
	wire := messageAttachmentWire{
		Type:             a.Type,
		URL:              a.URL,
		FileName:         a.FileName,
		MimeType:         a.MimeType,
		FileSize:         a.FileSize,
		DurationMs:       a.DurationMs,
		Waveform:         a.Waveform,
		Transcript:       a.Transcript,
		CanTranscribe:    a.CanTranscribe,
		ContactUserID:    a.ContactUserID,
		ContactFirstName: a.ContactFirstName,
		ContactLastName:  a.ContactLastName,
		ContactAvatarURL: a.ContactAvatarURL,
	}
	if a.Type == "photo" {
		blur := a.IsBlur
		wire.IsBlur = &blur
	}
	return wire
}

func (w messageAttachmentWire) toDTO() MessageAttachmentDTO {
	dto := MessageAttachmentDTO{
		Type:             w.Type,
		URL:              w.URL,
		FileName:         w.FileName,
		MimeType:         w.MimeType,
		FileSize:         w.FileSize,
		DurationMs:       w.DurationMs,
		Waveform:         w.Waveform,
		Transcript:       w.Transcript,
		CanTranscribe:    w.CanTranscribe,
		ContactUserID:    w.ContactUserID,
		ContactFirstName: w.ContactFirstName,
		ContactLastName:  w.ContactLastName,
		ContactAvatarURL: w.ContactAvatarURL,
	}
	if w.IsBlur != nil {
		dto.IsBlur = *w.IsBlur
	}
	return dto
}

func (a MessageAttachmentDTO) MarshalEasyJSON(w *jwriter.Writer) {
	a.toWire().MarshalEasyJSON(w)
}

func (a *MessageAttachmentDTO) UnmarshalEasyJSON(l *jlexer.Lexer) {
	var wire messageAttachmentWire
	wire.UnmarshalEasyJSON(l)
	*a = wire.toDTO()
}

func (a MessageAttachmentDTO) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	a.MarshalEasyJSON(&w)
	return w.Buffer.BuildBytes(), w.Error
}

func (a *MessageAttachmentDTO) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	a.UnmarshalEasyJSON(&r)
	return r.Error()
}
