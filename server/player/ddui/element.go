package ddui

import "math"

// DropdownOption is a selectable item in a Dropdown element.
type DropdownOption struct {
	Label       string
	Description string
	Value       int
}

type elementConfig struct {
	visible, disabled    *Observable[bool]
	description, tooltip *Observable[string]
}

func defaultElementConfig() elementConfig {
	return elementConfig{
		visible:     NewObservable(true, false),
		disabled:    NewObservable(false, false),
		description: NewObservable("", false),
		tooltip:     NewObservable("", false),
	}
}

// TextOption configures a Label or Header element.
type TextOption interface{ applyText(*elementConfig) }

// TextFieldOption configures a TextField element.
type TextFieldOption interface{ applyTextField(*elementConfig) }

// DropdownOptionConfig configures a Dropdown element.
type DropdownOptionConfig interface{ applyDropdown(*elementConfig) }

// ToggleOption configures a Toggle element.
type ToggleOption interface{ applyToggle(*elementConfig) }

// SliderOption configures a Slider element.
type SliderOption interface{ applySlider(*sliderElement) }

// ButtonOption configures a Button element.
type ButtonOption interface{ applyButton(*elementConfig) }

// DividerOption configures a Divider element.
type DividerOption interface{ applyDivider(*elementConfig) }

// SpacerOption configures a Spacer element.
type SpacerOption interface{ applySpacer(*elementConfig) }

type visibleOption struct{ value *Observable[bool] }

func (o visibleOption) applyText(c *elementConfig)            { c.visible = o.value }
func (o visibleOption) applyTextField(c *elementConfig)       { c.visible = o.value }
func (o visibleOption) applyDropdown(c *elementConfig)        { c.visible = o.value }
func (o visibleOption) applyToggle(c *elementConfig)          { c.visible = o.value }
func (o visibleOption) applySlider(s *sliderElement)          { s.config.visible = o.value }
func (o visibleOption) applyButton(c *elementConfig)          { c.visible = o.value }
func (o visibleOption) applyDivider(c *elementConfig)         { c.visible = o.value }
func (o visibleOption) applySpacer(c *elementConfig)          { c.visible = o.value }
func (o visibleOption) applyCloseButton(c *closeButtonOption) { c.visible = o.value }

// WithVisible controls whether an element is visible.
func WithVisible[T bool | *Observable[bool]](value T) visibleOption {
	return visibleOption{value: toBoolObs(value)}
}

type disabledOption struct{ value *Observable[bool] }

func (o disabledOption) applyTextField(c *elementConfig) { c.disabled = o.value }
func (o disabledOption) applyDropdown(c *elementConfig)  { c.disabled = o.value }
func (o disabledOption) applyToggle(c *elementConfig)    { c.disabled = o.value }
func (o disabledOption) applySlider(s *sliderElement)    { s.config.disabled = o.value }
func (o disabledOption) applyButton(c *elementConfig)    { c.disabled = o.value }

// WithDisabled controls whether an interactive element is disabled.
func WithDisabled[T bool | *Observable[bool]](value T) disabledOption {
	return disabledOption{value: toBoolObs(value)}
}

type descriptionOption struct{ value *Observable[string] }

func (o descriptionOption) applyTextField(c *elementConfig) { c.description = o.value }
func (o descriptionOption) applyDropdown(c *elementConfig)  { c.description = o.value }
func (o descriptionOption) applyToggle(c *elementConfig)    { c.description = o.value }
func (o descriptionOption) applySlider(s *sliderElement)    { s.config.description = o.value }

// WithDescription sets the descriptive text shown with a value element.
func WithDescription[T string | *Observable[string]](value T) descriptionOption {
	return descriptionOption{value: toStringObs(value)}
}

type tooltipOption struct{ value *Observable[string] }

func (o tooltipOption) applyButton(c *elementConfig) { c.tooltip = o.value }

type stepOption struct{ value *Observable[float64] }

func (o stepOption) applySlider(s *sliderElement) { s.step = o.value }

// WithStep sets the increment between slider values.
func WithStep[T int | float64 | *Observable[float64]](value T) stepOption {
	return stepOption{value: toFloatObs(value)}
}

type element interface {
	describe() ElementDescriptor
	handleUpdate(property string, value UpdateValue, origin *sendOrigin) func()
	bindSend(pathPrefix string, origin *sendOrigin, fn func(UpdateNotification)) func()
}

type spacerElement struct{ config elementConfig }

func (s *spacerElement) describe() ElementDescriptor {
	return ElementDescriptor{Kind: ElementSpacer, Visible: s.config.visible.Get()}
}
func (s *spacerElement) handleUpdate(_ string, _ UpdateValue, _ *sendOrigin) func() { return nil }
func (s *spacerElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return bindVisible(s.config.visible, path, "spacer_visible", origin, fn)
}
func (s *spacerElement) applyForm(f *CustomForm) { f.elements = append(f.elements, s) }

// Spacer adds a blank vertical spacer element.
func Spacer(opts ...SpacerOption) FormOption {
	e := &spacerElement{config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applySpacer(&e.config)
	}
	return e
}

type dividerElement struct{ config elementConfig }

func (d *dividerElement) describe() ElementDescriptor {
	return ElementDescriptor{Kind: ElementDivider, Visible: d.config.visible.Get()}
}
func (d *dividerElement) handleUpdate(_ string, _ UpdateValue, _ *sendOrigin) func() { return nil }
func (d *dividerElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return bindVisible(d.config.visible, path, "divider_visible", origin, fn)
}
func (d *dividerElement) applyForm(f *CustomForm) { f.elements = append(f.elements, d) }

// Divider adds a horizontal divider line element.
func Divider(opts ...DividerOption) FormOption {
	e := &dividerElement{config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyDivider(&e.config)
	}
	return e
}

type textElement struct {
	kind   ElementKind
	text   *Observable[string]
	config elementConfig
}

func (e *textElement) describe() ElementDescriptor {
	return ElementDescriptor{Kind: e.kind, StringValue: e.text.Get(), Visible: e.config.visible.Get()}
}
func (e *textElement) handleUpdate(_ string, _ UpdateValue, _ *sendOrigin) func() { return nil }
func (e *textElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	specific := "label_visible"
	if e.kind == ElementHeader {
		specific = "header_visible"
	}
	return combineUnbinds(
		bindString(e.text, path+".text", origin, fn),
		bindVisible(e.config.visible, path, specific, origin, fn),
	)
}
func (e *textElement) applyForm(f *CustomForm) { f.elements = append(f.elements, e) }

// Label adds a read-only text label.
func Label[T string | *Observable[string]](text T, opts ...TextOption) FormOption {
	return newTextElement(ElementLabel, text, opts...)
}

// Header adds prominent section text.
func Header[T string | *Observable[string]](text T, opts ...TextOption) FormOption {
	return newTextElement(ElementHeader, text, opts...)
}

func newTextElement[T string | *Observable[string]](kind ElementKind, text T, opts ...TextOption) *textElement {
	e := &textElement{kind: kind, text: toStringObs(text), config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyText(&e.config)
	}
	return e
}

type textFieldElement struct {
	label  *Observable[string]
	value  *Observable[string]
	config elementConfig
}

func (t *textFieldElement) describe() ElementDescriptor {
	return valueElementDescriptor(ElementTextField, t.label.Get(), t.config, t.value.Get(), 0, false)
}
func (t *textFieldElement) handleUpdate(property string, value UpdateValue, origin *sendOrigin) func() {
	if property == "text" && value.Kind == UpdateKindString && t.config.visible.Get() && !t.config.disabled.Get() && t.value.clientWritable {
		t.value.setFromClient(value.String, origin)
	}
	return nil
}
func (t *textFieldElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindValueFrame(t.label, t.config, path, "textfield_visible", origin, fn),
		bindString(t.value, path+".text", origin, fn),
	)
}
func (t *textFieldElement) applyForm(f *CustomForm) { f.elements = append(f.elements, t) }

// TextField adds a text input element bound to value.
func TextField[T string | *Observable[string]](label T, value *Observable[string], opts ...TextFieldOption) FormOption {
	e := &textFieldElement{label: toStringObs(label), value: value, config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyTextField(&e.config)
	}
	return e
}

type dropdownElement struct {
	label   *Observable[string]
	value   *Observable[int]
	options []DropdownOption
	config  elementConfig
}

func (d *dropdownElement) describe() ElementDescriptor {
	desc := valueElementDescriptor(ElementDropdown, d.label.Get(), d.config, "", d.value.Get(), false)
	desc.Options = append([]DropdownOption(nil), d.options...)
	return desc
}
func (d *dropdownElement) handleUpdate(property string, update UpdateValue, origin *sendOrigin) func() {
	if property != "value" || update.Kind != UpdateKindFloat ||
		!d.config.visible.Get() || d.config.disabled.Get() || !d.value.clientWritable {
		return nil
	}
	for _, option := range d.options {
		if float64(option.Value) == update.Float {
			d.value.setFromClient(option.Value, origin)
			return nil
		}
	}
	return nil
}
func (d *dropdownElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindValueFrame(d.label, d.config, path, "dropdown_visible", origin, fn),
		d.value.bindSendFrom(origin, func(value int) {
			fn(UpdateNotification{Path: path + ".value", Value: UpdateValue{Kind: UpdateKindFloat, Float: float64(value)}})
		}),
	)
}
func (d *dropdownElement) applyForm(f *CustomForm) { f.elements = append(f.elements, d) }

// Dropdown adds a dropdown selection element bound to value.
func Dropdown[T string | *Observable[string]](label T, value *Observable[int], options []DropdownOption, opts ...DropdownOptionConfig) FormOption {
	e := &dropdownElement{label: toStringObs(label), value: value, options: options, config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyDropdown(&e.config)
	}
	return e
}

type toggleElement struct {
	label  *Observable[string]
	value  *Observable[bool]
	config elementConfig
}

func (t *toggleElement) describe() ElementDescriptor {
	return valueElementDescriptor(ElementToggle, t.label.Get(), t.config, "", 0, t.value.Get())
}
func (t *toggleElement) handleUpdate(property string, update UpdateValue, origin *sendOrigin) func() {
	if property == "toggled" && update.Kind == UpdateKindBool && t.config.visible.Get() && !t.config.disabled.Get() && t.value.clientWritable {
		t.value.setFromClient(update.Bool, origin)
	}
	return nil
}
func (t *toggleElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindValueFrame(t.label, t.config, path, "toggle_visible", origin, fn),
		t.value.bindSendFrom(origin, func(value bool) {
			fn(UpdateNotification{Path: path + ".toggled", Value: UpdateValue{Kind: UpdateKindBool, Bool: value}})
		}),
	)
}
func (t *toggleElement) applyForm(f *CustomForm) { f.elements = append(f.elements, t) }

// Toggle adds a boolean toggle element bound to value.
func Toggle[T string | *Observable[string]](label T, value *Observable[bool], opts ...ToggleOption) FormOption {
	e := &toggleElement{label: toStringObs(label), value: value, config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyToggle(&e.config)
	}
	return e
}

type sliderElement struct {
	label          *Observable[string]
	value          *Observable[float64]
	min, max, step *Observable[float64]
	config         elementConfig
}

func (s *sliderElement) describe() ElementDescriptor {
	desc := valueElementDescriptor(ElementSlider, s.label.Get(), s.config, "", 0, false)
	desc.FloatValue, desc.Min, desc.Max, desc.Step = s.value.Get(), s.min.Get(), s.max.Get(), s.step.Get()
	return desc
}
func (s *sliderElement) handleUpdate(property string, update UpdateValue, origin *sendOrigin) func() {
	if property == "value" && update.Kind == UpdateKindFloat && validSliderValue(update.Float, s.min.Get(), s.max.Get(), s.step.Get()) &&
		s.config.visible.Get() && !s.config.disabled.Get() && s.value.clientWritable {
		s.value.setFromClient(update.Float, origin)
	}
	return nil
}

func validSliderValue(value, min, max, step float64) bool {
	if math.IsNaN(value) || math.IsNaN(min) || math.IsNaN(max) || math.IsNaN(step) ||
		math.IsInf(value, 0) || math.IsInf(min, 0) || math.IsInf(max, 0) || math.IsInf(step, 0) ||
		value < min || value > max {
		return false
	}
	if step <= 0 {
		return true
	}
	position := (value - min) / step
	return math.Abs(position-math.Round(position)) <= 1e-9*math.Max(1, math.Abs(position))
}
func (s *sliderElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindValueFrame(s.label, s.config, path, "slider_visible", origin, fn),
		bindFloat(s.value, path+".value", origin, fn),
		bindFloat(s.min, path+".minValue", origin, fn),
		bindFloat(s.max, path+".maxValue", origin, fn),
		bindFloat(s.step, path+".step", origin, fn),
	)
}
func (s *sliderElement) applyForm(f *CustomForm) { f.elements = append(f.elements, s) }

// Slider adds a numeric slider element bound to value. min and max define the range.
func Slider[L string | *Observable[string], Min int | float64 | *Observable[float64], Max int | float64 | *Observable[float64]](
	label L, value *Observable[float64], min Min, max Max, opts ...SliderOption,
) FormOption {
	e := &sliderElement{
		label:  toStringObs(label),
		value:  value,
		min:    toFloatObs(min),
		max:    toFloatObs(max),
		step:   NewObservable(1.0, false),
		config: defaultElementConfig(),
	}
	for _, opt := range opts {
		opt.applySlider(e)
	}
	return e
}

type buttonElement struct {
	label   *Observable[string]
	onClick func()
	config  elementConfig
}

func (b *buttonElement) describe() ElementDescriptor {
	return ElementDescriptor{
		Kind:     ElementButton,
		Label:    b.label.Get(),
		Tooltip:  b.config.tooltip.Get(),
		Visible:  b.config.visible.Get(),
		Disabled: b.config.disabled.Get(),
	}
}
func (b *buttonElement) handleUpdate(property string, update UpdateValue, _ *sendOrigin) func() {
	if property == "onClick" && validEventFloat(update) && b.config.visible.Get() && !b.config.disabled.Get() && b.onClick != nil {
		return b.onClick
	}
	return nil
}

func validEventFloat(update UpdateValue) bool {
	return update.Kind == UpdateKindFloat && !math.IsNaN(update.Float) && !math.IsInf(update.Float, 0)
}
func (b *buttonElement) bindSend(path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindString(b.label, path+".label", origin, fn),
		bindTooltip(b.config.tooltip, path, origin, fn),
		bindVisible(b.config.visible, path, "button_visible", origin, fn),
		bindBool(b.config.disabled, path+".disabled", origin, fn),
	)
}

func bindTooltip(obs *Observable[string], path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return obs.bindSendFrom(origin, func(value string) {
		fn(UpdateNotification{Path: path + ".tooltip", Value: UpdateValue{Kind: UpdateKindString, String: value}})
		fn(UpdateNotification{Path: path + ".tooltip_visible", Value: UpdateValue{Kind: UpdateKindBool, Bool: value != ""}})
	})
}
func (b *buttonElement) applyForm(f *CustomForm) { f.elements = append(f.elements, b) }

// Button adds a clickable button element. onClick is called when the client clicks it.
func Button[T string | *Observable[string]](label T, onClick func(), opts ...ButtonOption) FormOption {
	e := &buttonElement{label: toStringObs(label), onClick: onClick, config: defaultElementConfig()}
	for _, opt := range opts {
		opt.applyButton(&e.config)
	}
	return e
}

func valueElementDescriptor(kind ElementKind, label string, config elementConfig, stringValue string, intValue int, boolValue bool) ElementDescriptor {
	return ElementDescriptor{
		Kind:        kind,
		Label:       label,
		Description: config.description.Get(),
		Tooltip:     config.tooltip.Get(),
		StringValue: stringValue,
		IntValue:    intValue,
		BoolValue:   boolValue,
		Visible:     config.visible.Get(),
		Disabled:    config.disabled.Get(),
	}
}

func bindValueFrame(label *Observable[string], config elementConfig, path, specificVisible string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return combineUnbinds(
		bindString(label, path+".label", origin, fn),
		bindString(config.description, path+".description", origin, fn),
		bindVisible(config.visible, path, specificVisible, origin, fn),
		bindBool(config.disabled, path+".disabled", origin, fn),
	)
}

func bindVisible(obs *Observable[bool], path, specific string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return obs.bindSendFrom(origin, func(value bool) {
		update := UpdateValue{Kind: UpdateKindBool, Bool: value}
		fn(UpdateNotification{Path: path + ".visible", Value: update})
		fn(UpdateNotification{Path: path + "." + specific, Value: update})
	})
}

func bindString(obs *Observable[string], path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return obs.bindSendFrom(origin, func(value string) {
		fn(UpdateNotification{Path: path, Value: UpdateValue{Kind: UpdateKindString, String: value}})
	})
}

func bindBool(obs *Observable[bool], path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return obs.bindSendFrom(origin, func(value bool) {
		fn(UpdateNotification{Path: path, Value: UpdateValue{Kind: UpdateKindBool, Bool: value}})
	})
}

func bindFloat(obs *Observable[float64], path string, origin *sendOrigin, fn func(UpdateNotification)) func() {
	return obs.bindSendFrom(origin, func(value float64) {
		fn(UpdateNotification{Path: path, Value: UpdateValue{Kind: UpdateKindFloat, Float: value}})
	})
}

func combineUnbinds(unbinds ...func()) func() {
	return func() {
		for _, unbind := range unbinds {
			unbind()
		}
	}
}

func toBoolObs[T bool | *Observable[bool]](value T) *Observable[bool] {
	if obs, ok := any(value).(*Observable[bool]); ok {
		return obs
	}
	return NewObservable(any(value).(bool), false)
}

func toFloatObs[T int | float64 | *Observable[float64]](value T) *Observable[float64] {
	switch value := any(value).(type) {
	case int:
		return NewObservable(float64(value), false)
	case float64:
		return NewObservable(value, false)
	case *Observable[float64]:
		return value
	}
	panic("unreachable")
}
