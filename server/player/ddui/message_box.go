package ddui

// MessageBoxOption configures a MessageBox.
type MessageBoxOption interface {
	applyMessageBox(m *MessageBox)
}

type bodyOption struct{ obs *Observable[string] }

func (b bodyOption) applyMessageBox(m *MessageBox) { m.body = b.obs }

// Body sets the message box body.
func Body[T string | *Observable[string]](text T) MessageBoxOption {
	return bodyOption{obs: toStringObs(text)}
}

type button1Option struct {
	label *Observable[string]
	opts  []MessageBoxButtonOption
}

func (b button1Option) applyMessageBox(m *MessageBox) {
	m.btn1.label = b.label
	for _, o := range b.opts {
		o.applyMessageBoxButton(&m.btn1)
	}
}

// Button1 sets the first button.
func Button1[T string | *Observable[string]](label T, opts ...MessageBoxButtonOption) MessageBoxOption {
	return button1Option{label: toStringObs(label), opts: opts}
}

type button2Option struct {
	label *Observable[string]
	opts  []MessageBoxButtonOption
}

func (b button2Option) applyMessageBox(m *MessageBox) {
	m.btn2.label = b.label
	for _, o := range b.opts {
		o.applyMessageBoxButton(&m.btn2)
	}
}

// Button2 sets the second button.
func Button2[T string | *Observable[string]](label T, opts ...MessageBoxButtonOption) MessageBoxOption {
	return button2Option{label: toStringObs(label), opts: opts}
}

// MessageBox is a two-button confirmation dialog. The Handler is called with
// the selected button (1 or 2) when the form closes, or 0 if cancelled.
type MessageBox struct {
	title      *Observable[string]
	body       *Observable[string]
	btn1, btn2 messageBoxButton
	handler    func(selection int)
}

type messageBoxButton struct {
	label   *Observable[string]
	tooltip *Observable[string]
}

// MessageBoxButtonOption configures optional properties of a MessageBox button.
type MessageBoxButtonOption interface{ applyMessageBoxButton(*messageBoxButton) }

func (o tooltipOption) applyMessageBoxButton(button *messageBoxButton) { button.tooltip = o.value }

// WithTooltip sets the button tooltip.
func WithTooltip[T string | *Observable[string]](tooltip T) tooltipOption {
	return tooltipOption{value: toStringObs(tooltip)}
}

// NewMessageBox creates a MessageBox.
func NewMessageBox[T string | *Observable[string]](title T, opts ...MessageBoxOption) *MessageBox {
	m := &MessageBox{title: toStringObs(title)}
	for _, o := range opts {
		o.applyMessageBox(m)
	}
	return m
}

func (m *MessageBox) ScreenID() string { return "minecraft:message_box" }

func (m *MessageBox) Describe() FormDescriptor {
	desc := FormDescriptor{Title: m.title.Get()}
	if m.body != nil {
		desc.Body = m.body.Get()
	}
	if m.btn1.label != nil {
		desc.HasButton1 = true
		desc.Button1.Label = m.btn1.label.Get()
	}
	if m.btn1.tooltip != nil {
		desc.Button1.Tooltip = m.btn1.tooltip.Get()
	}
	if m.btn2.label != nil {
		desc.HasButton2 = true
		desc.Button2.Label = m.btn2.label.Get()
	}
	if m.btn2.tooltip != nil {
		desc.Button2.Tooltip = m.btn2.tooltip.Get()
	}
	return desc
}

func (m *MessageBox) HandleUpdate(path string, value UpdateValue) UpdateResult {
	return m.HandleUpdateFrom(0, path, value)
}

// HandleUpdateFrom processes a client change associated with a form binding.
func (m *MessageBox) HandleUpdateFrom(_ uint64, path string, value UpdateValue) UpdateResult {
	if !validEventFloat(value) {
		return UpdateResult{}
	}
	selection := 0
	switch path {
	case "button1.onClick":
		if m.btn1.label == nil {
			return UpdateResult{}
		}
		selection = 1
	case "button2.onClick":
		if m.btn2.label == nil {
			return UpdateResult{}
		}
		selection = 2
	default:
		return UpdateResult{}
	}
	return UpdateResult{Close: true, Complete: func() {
		if m.handler != nil {
			m.handler(selection)
		}
	}}
}

func (m *MessageBox) BindSend(fn func(UpdateNotification)) func() {
	return m.BindSendFrom(0, fn)
}

// BindSendFrom binds server updates to a specific form screen.
func (m *MessageBox) BindSendFrom(_ uint64, fn func(UpdateNotification)) func() {
	unbinds := make([]func(), 0, 6)
	bind := func(obs *Observable[string], path string) {
		if obs == nil {
			return
		}
		unbinds = append(unbinds, obs.bindSendFrom(nil, func(v string) {
			fn(UpdateNotification{Path: path, Value: UpdateValue{Kind: UpdateKindString, String: v}})
		}))
	}
	bind(m.title, "title")
	bind(m.body, "body")
	bind(m.btn1.label, "button1.label")
	bindTooltip := func(obs *Observable[string], path string) {
		if obs == nil {
			return
		}
		unbinds = append(unbinds, obs.bindSendFrom(nil, func(value string) {
			fn(UpdateNotification{Path: path + ".tooltip", Value: UpdateValue{Kind: UpdateKindString, String: value}})
			fn(UpdateNotification{Path: path + ".tooltip_visible", Value: UpdateValue{Kind: UpdateKindBool, Bool: value != ""}})
		}))
	}
	bindTooltip(m.btn1.tooltip, "button1")
	bind(m.btn2.label, "button2.label")
	bindTooltip(m.btn2.tooltip, "button2")
	return func() {
		for _, unbind := range unbinds {
			unbind()
		}
	}
}

func (m *MessageBox) OnClose(_ int) {
	if m.handler != nil {
		m.handler(0)
	}
}

func toStringObs[T string | *Observable[string]](v T) *Observable[string] {
	switch val := any(v).(type) {
	case string:
		return NewObservable(val, false)
	case *Observable[string]:
		return val
	}
	panic("unreachable")
}
