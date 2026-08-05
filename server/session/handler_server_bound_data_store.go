package session

import (
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/player/ddui"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type ServerBoundDataStoreHandler struct {
	h *DDUIFormHandler
}

func (d *ServerBoundDataStoreHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, _ Controllable) error {
	pk := p.(*packet.ServerBoundDataStore)
	if pk.Update.DataStoreName != "minecraft" {
		return nil
	}

	property := pk.Update.Property
	const sep = "_data_"
	i := strings.LastIndex(property, sep)
	if i < 0 {
		return nil
	}
	id, err := strconv.ParseUint(property[i+len(sep):], 10, 32)
	if err != nil {
		return nil
	}

	d.h.mu.Lock()
	af := d.h.forms[uint32(id)]
	active := true
	for instanceID := range d.h.forms {
		if instanceID > uint32(id) {
			active = false
			break
		}
	}
	if af == nil || property != af.property {
		active = false
	}
	d.h.mu.Unlock()

	if !active {
		return nil
	}

	value, valid := dataStoreControlToUpdateValue(pk.Update)
	if !valid {
		return nil
	}

	result, accepted := af.handleUpdate(pk.Update.Path, value)
	if !accepted {
		return nil
	}
	if result.Close {
		af.sendMu.Lock()
		af.sendMu.Unlock()
		d.h.mu.Lock()
		if d.h.forms[af.instanceID] == af {
			delete(d.h.forms, af.instanceID)
		}
		d.h.mu.Unlock()
		af.unbind()
		s.writePacket(&packet.ClientBoundDataDrivenUICloseScreen{
			FormID: protocol.Option(af.formID),
		})
		sendDataStoreCleanup(s, af)
		if result.Complete == nil {
			af.form.OnClose(ddui.Closed)
		} else {
			result.Complete()
		}
	} else if result.Complete != nil {
		result.Complete()
	}
	return nil
}

func dataStoreControlToUpdateValue(u protocol.DataStoreUpdate) (ddui.UpdateValue, bool) {
	switch u.ControlType {
	case protocol.DataStoreControlDouble:
		return ddui.UpdateValue{Kind: ddui.UpdateKindFloat, Float: u.DoubleValue}, true
	case protocol.DataStoreControlBoolean:
		return ddui.UpdateValue{Kind: ddui.UpdateKindBool, Bool: u.BoolValue}, true
	case protocol.DataStoreControlString:
		return ddui.UpdateValue{Kind: ddui.UpdateKindString, String: u.StringValue}, true
	default:
		return ddui.UpdateValue{}, false
	}
}
