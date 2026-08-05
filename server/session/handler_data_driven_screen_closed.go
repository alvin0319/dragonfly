package session

import (
	"github.com/df-mc/dragonfly/server/player/ddui"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// DataDrivenScreenClosedHandler handles the ServerBoundDataDrivenScreenClosed packet.
type DataDrivenScreenClosedHandler struct {
	h *DDUIFormHandler
}

func (d *DataDrivenScreenClosedHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, _ Controllable) error {
	pk := p.(*packet.ServerBoundDataDrivenScreenClosed)

	d.h.mu.Lock()
	var af *activeDDUIForm
	for _, f := range d.h.forms {
		if f.formID == pk.FormID {
			af = f
			delete(d.h.forms, f.instanceID)
			break
		}
	}
	d.h.mu.Unlock()

	if af == nil {
		return nil
	}
	if !af.claim() {
		return nil
	}

	af.unbind()
	sendDataStoreCleanup(s, af)
	af.form.OnClose(closeReasonToDDUI(pk.CloseReason))
	return nil
}

func closeReasonToDDUI(reason string) int {
	switch reason {
	case packet.DataDrivenScreenCloseReasonProgrammaticClose:
		return ddui.CloseReasonProgrammatic
	case packet.DataDrivenScreenCloseReasonProgrammaticCloseAll:
		return ddui.CloseReasonProgrammaticAll
	case packet.DataDrivenScreenCloseReasonClientCanceled:
		return ddui.Closed
	case packet.DataDrivenScreenCloseReasonUserBusy:
		return ddui.Busy
	case packet.DataDrivenScreenCloseReasonInvalidForm:
		return ddui.Invalid
	default:
		return ddui.Closed
	}
}
