package ddui

import "sync"

// Observable holds a value that can be observed for changes.
type Observable[T string | int | float64 | bool] struct {
	listeners      []listener[T]
	nextListenerID uint64
	sendFns        map[uint64]sendBinding[T]
	nextSendID     uint64
	clientWritable bool
	value          T
	mut            sync.RWMutex
}

type listener[T string | int | float64 | bool] struct {
	id uint64
	fn func(value T)
}

type sendBinding[T string | int | float64 | bool] struct {
	origin *sendOrigin
	fn     func(value T)
}

type sendOrigin struct{ token byte }

// NewObservable creates a new Observable with the given initial value.
// clientWritable controls whether client-originated packets may write back into
// this observable. Set it to false for server-authoritative values.
func NewObservable[T string | int | float64 | bool](initialValue T, clientWritable bool) *Observable[T] {
	return &Observable[T]{
		listeners:      make([]listener[T], 0),
		sendFns:        make(map[uint64]sendBinding[T]),
		clientWritable: clientWritable,
		value:          initialValue,
	}
}

// Set updates the current value and notifies all listeners and bound send functions.
func (o *Observable[T]) Set(value T) {
	o.set(value, nil, false)
}

func (o *Observable[T]) setFromClient(value T, origin *sendOrigin) {
	o.set(value, origin, true)
}

func (o *Observable[T]) set(value T, origin *sendOrigin, skipOrigin bool) {
	o.mut.Lock()
	if o.value == value {
		o.mut.Unlock()
		return
	}
	o.value = value
	listeners := make([]func(value T), 0, len(o.listeners))
	for _, listener := range o.listeners {
		listeners = append(listeners, listener.fn)
	}
	sendFns := make([]func(value T), 0, len(o.sendFns))
	for _, binding := range o.sendFns {
		if !skipOrigin || binding.origin != origin {
			sendFns = append(sendFns, binding.fn)
		}
	}
	o.mut.Unlock()

	for _, fn := range sendFns {
		fn(value)
	}
	for _, fn := range listeners {
		fn(value)
	}
}

// Get returns the current value.
func (o *Observable[T]) Get() T {
	o.mut.RLock()
	defer o.mut.RUnlock()
	return o.value
}

// Listen adds a listener that is called when the value changes via Set or from the client.
// The returned function removes the listener.
func (o *Observable[T]) Listen(fn func(value T)) func() {
	o.mut.Lock()
	o.nextListenerID++
	id := o.nextListenerID
	o.listeners = append(o.listeners, listener[T]{id: id, fn: fn})
	o.mut.Unlock()

	return func() {
		o.mut.Lock()
		for i, listener := range o.listeners {
			if listener.id == id {
				copy(o.listeners[i:], o.listeners[i+1:])
				o.listeners = o.listeners[:len(o.listeners)-1]
				break
			}
		}
		o.mut.Unlock()
	}
}

func (o *Observable[T]) bindSend(fn func(T)) func() {
	return o.bindSendFrom(nil, fn)
}

func (o *Observable[T]) bindSendFrom(origin *sendOrigin, fn func(T)) func() {
	o.mut.Lock()
	if o.sendFns == nil {
		o.sendFns = make(map[uint64]sendBinding[T])
	}
	o.nextSendID++
	id := o.nextSendID
	o.sendFns[id] = sendBinding[T]{origin: origin, fn: fn}
	o.mut.Unlock()

	return func() {
		o.mut.Lock()
		delete(o.sendFns, id)
		o.mut.Unlock()
	}
}
