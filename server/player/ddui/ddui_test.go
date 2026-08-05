package ddui

import (
	"math"
	"testing"
)

func TestObservableListenCanBeRemoved(t *testing.T) {
	obs := NewObservable(0, false)
	calls := 0
	remove := obs.Listen(func(int) { calls++ })

	obs.Set(1)
	remove()
	obs.Set(2)

	if calls != 1 {
		t.Fatalf("listener called %d times, want 1", calls)
	}
}

func TestClientUpdatesDoNotEchoToBoundForm(t *testing.T) {
	value := NewObservable("before", true)
	form := New("Settings", TextField("Name", value))
	updates := make([]UpdateNotification, 0, 2)
	remove := form.BindSend(func(update UpdateNotification) {
		updates = append(updates, update)
	})
	defer remove()

	form.HandleUpdate("layout[0].text", UpdateValue{Kind: UpdateKindString, String: "from client"})
	if got := value.Get(); got != "from client" {
		t.Fatalf("value after client update = %q, want %q", got, "from client")
	}
	if len(updates) != 0 {
		t.Fatalf("client update echoed %d times, want 0", len(updates))
	}

	value.Set("from server")
	if len(updates) != 1 {
		t.Fatalf("server update produced %d notifications, want 1", len(updates))
	}
}

func TestClientUpdatesOnlySkipTheirOriginBinding(t *testing.T) {
	value := NewObservable("before", true)
	firstForm := New("First", TextField("Name", value))
	secondForm := New("Second", TextField("Name", value))
	first, second := 0, 0
	removeFirst := firstForm.BindSendFrom(1, func(UpdateNotification) { first++ })
	defer removeFirst()
	removeSecond := secondForm.BindSendFrom(2, func(UpdateNotification) { second++ })
	defer removeSecond()

	firstForm.HandleUpdateFrom(1, "layout[0].text", UpdateValue{Kind: UpdateKindString, String: "from client"})
	if first != 0 || second != 1 {
		t.Fatalf("binding notifications = (%d, %d), want (0, 1)", first, second)
	}

	siblingValue := NewObservable("before", true)
	siblingForm := New("Sibling", TextField("First", siblingValue), TextField("Second", siblingValue))
	siblingFirst, siblingSecond := 0, 0
	removeSibling := siblingForm.BindSendFrom(3, func(update UpdateNotification) {
		if update.Path == "layout[0].text" {
			siblingFirst++
		} else if update.Path == "layout[1].text" {
			siblingSecond++
		}
	})
	defer removeSibling()
	siblingForm.HandleUpdateFrom(3, "layout[0].text", UpdateValue{Kind: UpdateKindString, String: "sibling"})
	if siblingFirst != 0 || siblingSecond != 1 {
		t.Fatalf("sibling notifications = (%d, %d), want (0, 1)", siblingFirst, siblingSecond)
	}
}

func TestCustomFormRejectsInvalidControlValues(t *testing.T) {
	dropdownValue := NewObservable(10, true)
	form := New("Settings", Dropdown("Mode", dropdownValue, []DropdownOption{
		{Label: "Ten", Value: 10},
		{Label: "Twenty", Value: 20},
	}))

	form.HandleUpdate("layout[0].value", UpdateValue{Kind: UpdateKindFloat, Float: 10.5})
	if got := dropdownValue.Get(); got != 10 {
		t.Fatalf("dropdown accepted fractional value %d", got)
	}

	form.HandleUpdate("layout[0].value", UpdateValue{Kind: UpdateKindFloat, Float: 20})
	if got := dropdownValue.Get(); got != 20 {
		t.Fatalf("dropdown value = %d, want 20", got)
	}

	sliderValue := NewObservable(0.0, true)
	slider := New("Settings", Slider("Volume", sliderValue, 0.0, 2.0, WithStep(0.5)))
	slider.HandleUpdate("layout[0].value", UpdateValue{Kind: UpdateKindFloat, Float: 1.25})
	if got := sliderValue.Get(); got != 0 {
		t.Fatalf("slider accepted off-step value %v", got)
	}

	slider.HandleUpdate("layout[0].value", UpdateValue{Kind: UpdateKindFloat, Float: 1.5})
	if got := sliderValue.Get(); got != 1.5 {
		t.Fatalf("slider value = %v, want 1.5", got)
	}
}

func TestCustomFormButtonStaysOpen(t *testing.T) {
	clicked := 0
	form := New("Actions", Button("Run", func() { clicked++ }))

	result := form.HandleUpdate("layout[0].onClick", UpdateValue{Kind: UpdateKindFloat})
	if result.Close || result.Complete == nil {
		t.Fatal("button update should complete a callback without closing the form")
	}
	result.Complete()
	if clicked != 1 {
		t.Fatalf("button callback count = %d, want 1", clicked)
	}
}

func TestMessageBoxButtonClosesWithSelection(t *testing.T) {
	selection := 0
	box := NewMessageBox("Confirm", Button1("Yes"), Button2("No"), Handler(func(value int) {
		selection = value
	}))

	result := box.HandleUpdate("button2.onClick", UpdateValue{Kind: UpdateKindFloat})
	if !result.Close || result.Complete == nil {
		t.Fatal("message box button should close and complete")
	}
	result.Complete()
	if selection != 2 {
		t.Fatalf("selection = %d, want 2", selection)
	}

	invalid := box.HandleUpdate("button1.onClick", UpdateValue{Kind: UpdateKindString, String: "click"})
	if invalid.Close || invalid.Complete != nil {
		t.Fatal("message box accepted a non-numeric click update")
	}
	invalid = box.HandleUpdate("button1.onClick", UpdateValue{Kind: UpdateKindFloat, Float: math.NaN()})
	if invalid.Close || invalid.Complete != nil {
		t.Fatal("message box accepted a non-finite click update")
	}
}
