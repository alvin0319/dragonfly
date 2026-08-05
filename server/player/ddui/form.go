package ddui

const (
	// CloseReasonProgrammatic is used when a DDUI screen is closed by a server.
	CloseReasonProgrammatic = iota
	// CloseReasonProgrammaticAll is used when server closed all DDUI screens.
	CloseReasonProgrammaticAll
	// Closed is used when a DDUI screen is closed by the client.
	Closed
	// Busy is used when client is busy and unable to accept a DDUI screen.
	Busy
	// Invalid is used when a DDUI screen is invalid.
	Invalid
)

// UpdateKind identifies the type of a value received from the client.
type UpdateKind int8

const (
	UpdateKindInvalid UpdateKind = iota
	UpdateKindFloat
	UpdateKindBool
	UpdateKindString
)

// UpdateValue holds a typed value from a client field change.
type UpdateValue struct {
	Kind   UpdateKind
	Float  float64
	Bool   bool
	String string
}

// UpdateNotification carries a path-based value change from a server-side Observable
// to the session so it can be sent to the client.
type UpdateNotification struct {
	Path  string
	Value UpdateValue
}

// UpdateResult describes the action a form takes after a client update.
type UpdateResult struct {
	Close    bool
	Complete func()
}

// ElementKind identifies the type of a form element.
type ElementKind int8

const (
	ElementSpacer ElementKind = iota
	ElementDivider
	ElementLabel
	ElementHeader
	ElementTextField
	ElementDropdown
	ElementToggle
	ElementSlider
	ElementButton
)

// ElementDescriptor carries the render data for a single form element.
type ElementDescriptor struct {
	Kind           ElementKind
	Label          string
	Description    string
	Tooltip        string
	StringValue    string
	IntValue       int
	FloatValue     float64
	BoolValue      bool
	Visible        bool
	Disabled       bool
	Min, Max, Step float64
	Options        []DropdownOption
}

// ButtonDescriptor carries the label and optional tooltip for a MessageBox button.
type ButtonDescriptor struct {
	Label   string
	Tooltip string
}

// FormDescriptor captures the complete static structure of a form for session serialization.
type FormDescriptor struct {
	Title          string
	HasCloseButton bool
	CloseButton    ElementDescriptor
	Elements       []ElementDescriptor
	// MessageBox fields:
	Body       string
	HasButton1 bool
	Button1    ButtonDescriptor
	HasButton2 bool
	Button2    ButtonDescriptor
}

// Form is implemented by any type that can be shown to a player as a data-driven UI.
type Form interface {
	// OnClose is called when this form closes without completing a form action.
	// reason is one of the CloseReason constants.
	OnClose(reason int)
	// ScreenID returns the Bedrock data-driven UI screen identifier (e.g. "minecraft:custom_form").
	ScreenID() string
	// Describe returns the form's complete structure for serialization by the session.
	Describe() FormDescriptor
	// HandleUpdate processes a client-initiated field change at path.
	// It returns the action that should be taken after the update.
	HandleUpdate(path string, value UpdateValue) UpdateResult
	// BindSend registers the callback the session uses to receive server-side Observable changes and returns a function that removes the binding.
	BindSend(fn func(UpdateNotification)) func()
}

// BindingForm is implemented by built-in forms that can suppress updates only
// for the screen that originated a client change.
type BindingForm interface {
	Form
	BindSendFrom(bindingID uint64, fn func(UpdateNotification)) func()
	HandleUpdateFrom(bindingID uint64, path string, value UpdateValue) UpdateResult
}

// HandlerOption is returned by Handler and satisfies both FormOption and MessageBoxOption.
type HandlerOption struct{ fn func(int) }

func (h HandlerOption) applyForm(f *CustomForm)       { f.closeHandler = h.fn }
func (h HandlerOption) applyMessageBox(m *MessageBox) { m.handler = h.fn }

// Handler sets the function called when a form closes or a MessageBox button is selected.
func Handler(fn func(int)) HandlerOption { return HandlerOption{fn: fn} }
