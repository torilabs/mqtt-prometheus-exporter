package mqtt

import (
	"sync"
	"testing"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestDelegatingMessageHandler_SingleHandler(t *testing.T) {
	callCount := 0
	handler := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		callCount++
	}

	delegatingHandler := NewDelegatingMessageHandler(handler)
	msg := &fakeMessage{topic: "test/topic", payload: []byte("test")}

	delegatingHandler(nil, msg)

	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}
}

func TestDelegatingMessageHandler_MultipleHandlers(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	handler1 := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	}

	handler2 := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	}

	handler3 := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	}

	delegatingHandler := NewDelegatingMessageHandler(handler1, handler2, handler3)
	msg := &fakeMessage{topic: "test/topic", payload: []byte("test")}

	delegatingHandler(nil, msg)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 3 {
		t.Errorf("Expected all 3 handlers to be called, got %d calls", callCount)
	}
}

func TestDelegatingMessageHandler_HandlerPanic(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	handler1 := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		panic("test panic")
	}

	handler2 := func(_ pahomqtt.Client, _ pahomqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	}

	delegatingHandler := NewDelegatingMessageHandler(handler1, handler2)
	msg := &fakeMessage{topic: "test/topic", payload: []byte("test")}

	// Should not panic, both handlers should be called
	delegatingHandler(nil, msg)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 2 {
		t.Errorf("Expected both handlers to be called even with panic, got %d calls", callCount)
	}
}

func TestDelegatingMessageHandler_NoHandlers(t *testing.T) {
	delegatingHandler := NewDelegatingMessageHandler()
	msg := &fakeMessage{topic: "test/topic", payload: []byte("test")}

	delegatingHandler(nil, msg)
}
