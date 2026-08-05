package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player/ddui"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDataStoreSerializationPreservesControlTypes(t *testing.T) {
	desc := ddui.FormDescriptor{
		Title:          "Settings",
		HasCloseButton: true,
		CloseButton:    ddui.ElementDescriptor{Visible: false, Label: "Dismiss"},
		Elements: []ddui.ElementDescriptor{
			{Kind: ddui.ElementDropdown, Options: []ddui.DropdownOption{{Label: "Easy", Description: "Low", Value: 3}}, IntValue: 3},
			{Kind: ddui.ElementSlider, Min: 0.25, Max: 2.75, Step: 0.25, FloatValue: 1.5},
		},
	}

	root := serializeCustomForm(desc)
	closeButton := mapValue(t, root, "closeButton")
	if got := boolValue(t, closeButton, "visible"); got {
		t.Fatal("hidden close button was serialized as visible")
	}

	layout := mapValue(t, root, "layout")
	dropdown := mapValue(t, layout, "0")
	items := mapValue(t, dropdown, "items")
	item := mapValue(t, items, "0")
	if got := intValue(t, item, "value"); got != 3 {
		t.Fatalf("dropdown item value = %d, want 3", got)
	}
	if got := stringValue(t, item, "description"); got != "Low" {
		t.Fatalf("dropdown item description = %q, want %q", got, "Low")
	}

	slider := mapValue(t, layout, "1")
	for _, key := range []string{"minValue", "maxValue", "step", "value"} {
		if got := propertyValue(t, slider, key).Type; got != protocol.DataStorePropertyTypeDouble {
			t.Fatalf("slider %s type = %d, want double", key, got)
		}
	}
}

func TestMessageBoxSerializationOmitsUnsetButtons(t *testing.T) {
	desc := ddui.NewMessageBox("Confirm", ddui.Body("Continue?")).Describe()
	root := serializeMessageBox(desc)
	if _, ok := findMapValue(root, "button1"); ok {
		t.Fatal("unset first button was serialized")
	}
	if _, ok := findMapValue(root, "button2"); ok {
		t.Fatal("unset second button was serialized")
	}
}

func TestDataStoreUpdateCarriesMonotonicCounts(t *testing.T) {
	af := &activeDDUIForm{
		propertyUpdateCount: 1,
		pathUpdateCounts:    make(map[string]uint32),
	}

	property, path := af.recordUpdate("layout[0].text")
	if property != 2 || path != 1 {
		t.Fatalf("first update counts = (%d, %d), want (2, 1)", property, path)
	}
	property, path = af.recordUpdate("layout[0].text")
	if property != 3 || path != 2 {
		t.Fatalf("second update counts = (%d, %d), want (3, 2)", property, path)
	}
	property, path = af.recordUpdate("title")
	if property != 4 || path != 1 {
		t.Fatalf("different-path update counts = (%d, %d), want (4, 1)", property, path)
	}

	update := serializeUpdate("custom_form_data_1", ddui.UpdateNotification{
		Path:  "title",
		Value: ddui.UpdateValue{Kind: ddui.UpdateKindString, String: "Updated"},
	}, property, path)
	if update.PropertyUpdateCount != 4 || update.PathUpdateCount != 1 {
		t.Fatalf("serialized counts = (%d, %d), want (4, 1)", update.PropertyUpdateCount, update.PathUpdateCount)
	}
}

func TestServerBoundDataStoreRejectsInvalidOwnershipAndTypes(t *testing.T) {
	value := ddui.NewObservable("before", true)
	form := ddui.New("Settings", ddui.TextField("Name", value))
	h := &DDUIFormHandler{forms: make(map[uint32]*activeDDUIForm)}
	h.forms[1] = &activeDDUIForm{
		form:             form,
		instanceID:       1,
		property:         "custom_form_data_1",
		pathUpdateCounts: make(map[string]uint32),
		unbind:           func() {},
	}
	handler := &ServerBoundDataStoreHandler{h: h}

	packetFor := func(property string, controlType uint32) *packet.ServerBoundDataStore {
		return &packet.ServerBoundDataStore{Update: protocol.DataStoreUpdate{
			DataStoreName: "minecraft",
			Property:      property,
			Path:          "layout[0].text",
			ControlType:   controlType,
			StringValue:   "changed",
		}}
	}

	if err := handler.Handle(packetFor("custom_form_data_1", 99), Nop, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := value.Get(); got != "before" {
		t.Fatalf("invalid control type changed value to %q", got)
	}

	if err := handler.Handle(packetFor("other_data_1", protocol.DataStoreControlString), Nop, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := value.Get(); got != "before" {
		t.Fatalf("wrong property changed value to %q", got)
	}

	if err := handler.Handle(packetFor("custom_form_data_1", protocol.DataStoreControlString), Nop, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := value.Get(); got != "changed" {
		t.Fatalf("valid update changed value to %q", got)
	}
}

func TestServerBoundDataStoreOnlyUpdatesTopForm(t *testing.T) {
	oldValue := ddui.NewObservable("old", true)
	newValue := ddui.NewObservable("new", true)
	oldForm := ddui.New("Old", ddui.TextField("Name", oldValue))
	newForm := ddui.New("New", ddui.TextField("Name", newValue))
	h := &DDUIFormHandler{forms: map[uint32]*activeDDUIForm{
		1: {form: oldForm, instanceID: 1, property: "custom_form_data_1", pathUpdateCounts: make(map[string]uint32)},
		2: {form: newForm, instanceID: 2, property: "custom_form_data_2", pathUpdateCounts: make(map[string]uint32)},
	}}
	handler := &ServerBoundDataStoreHandler{h: h}

	err := handler.Handle(&packet.ServerBoundDataStore{Update: protocol.DataStoreUpdate{
		DataStoreName: "minecraft",
		Property:      "custom_form_data_1",
		Path:          "layout[0].text",
		ControlType:   protocol.DataStoreControlString,
		StringValue:   "ignored",
	}}, Nop, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if oldValue.Get() != "old" || newValue.Get() != "new" {
		t.Fatalf("non-top form was updated: old=%q new=%q", oldValue.Get(), newValue.Get())
	}
}

func TestDataStoreControlConversionRejectsUnknownTypes(t *testing.T) {
	if _, ok := dataStoreControlToUpdateValue(protocol.DataStoreUpdate{ControlType: 99}); ok {
		t.Fatal("unknown data-store control type was accepted")
	}
}

func TestCloseDDUIFormsPublishesCleanupBeforeCallbacks(t *testing.T) {
	s := &Session{packets: make(chan packet.Packet, 4), closeBackground: make(chan struct{})}
	callbackQueueLength := 0
	form := ddui.New("Settings", ddui.Handler(func(int) {
		callbackQueueLength = len(s.packets)
	}))
	h := &DDUIFormHandler{forms: map[uint32]*activeDDUIForm{
		1: {
			form:                form,
			instanceID:          1,
			property:            "custom_form_data_1",
			propertyUpdateCount: 1,
			pathUpdateCounts:    make(map[string]uint32),
			unbind:              func() {},
		},
	}}

	h.CloseDDUIForms(s)
	if callbackQueueLength != 2 {
		t.Fatalf("callback observed %d queued packets, want close plus cleanup", callbackQueueLength)
	}
}

func TestMessageBoxSelectionPublishesCloseBeforeCallback(t *testing.T) {
	s := &Session{packets: make(chan packet.Packet, 4), closeBackground: make(chan struct{})}
	callbackQueueLength := 0
	box := ddui.NewMessageBox("Confirm", ddui.Button1("Yes"), ddui.Handler(func(int) {
		callbackQueueLength = len(s.packets)
	}))
	h := &DDUIFormHandler{forms: map[uint32]*activeDDUIForm{
		1: {
			form:                box,
			formID:              7,
			instanceID:          1,
			property:            "message_box_data_1",
			propertyUpdateCount: 1,
			pathUpdateCounts:    make(map[string]uint32),
			unbind:              func() {},
		},
	}}

	handler := &ServerBoundDataStoreHandler{h: h}
	err := handler.Handle(&packet.ServerBoundDataStore{Update: protocol.DataStoreUpdate{
		DataStoreName: "minecraft",
		Property:      "message_box_data_1",
		Path:          "button1.onClick",
		ControlType:   protocol.DataStoreControlDouble,
	}}, s, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if callbackQueueLength != 2 {
		t.Fatalf("callback observed %d queued packets, want close plus cleanup", callbackQueueLength)
	}
}

func propertyValue(t *testing.T, value protocol.DataStorePropertyValue, key string) protocol.DataStorePropertyValue {
	t.Helper()
	for _, entry := range value.MapValue {
		if entry.Key == key {
			return entry.Value
		}
	}
	t.Fatalf("data-store map is missing %q", key)
	return protocol.DataStorePropertyValue{}
}

func findMapValue(value protocol.DataStorePropertyValue, key string) (protocol.DataStorePropertyValue, bool) {
	for _, entry := range value.MapValue {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return protocol.DataStorePropertyValue{}, false
}

func mapValue(t *testing.T, value protocol.DataStorePropertyValue, key string) protocol.DataStorePropertyValue {
	t.Helper()
	value = propertyValue(t, value, key)
	if value.Type != protocol.DataStorePropertyTypeMap {
		t.Fatalf("data-store value %q has type %d, want map", key, value.Type)
	}
	return value
}

func boolValue(t *testing.T, value protocol.DataStorePropertyValue, key string) bool {
	t.Helper()
	value = propertyValue(t, value, key)
	if value.Type != protocol.DataStorePropertyTypeBool {
		t.Fatalf("data-store value %q has type %d, want bool", key, value.Type)
	}
	return value.BoolValue
}

func intValue(t *testing.T, value protocol.DataStorePropertyValue, key string) int64 {
	t.Helper()
	value = propertyValue(t, value, key)
	if value.Type != protocol.DataStorePropertyTypeInt64 {
		t.Fatalf("data-store value %q has type %d, want int64", key, value.Type)
	}
	return value.Int64Value
}

func stringValue(t *testing.T, value protocol.DataStorePropertyValue, key string) string {
	t.Helper()
	value = propertyValue(t, value, key)
	if value.Type != protocol.DataStorePropertyTypeString {
		t.Fatalf("data-store value %q has type %d, want string", key, value.Type)
	}
	return value.StringValue
}
