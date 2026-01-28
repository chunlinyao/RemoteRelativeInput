//go:build darwin
// +build darwin

package client

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

extern CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon);

static CFMachPortRef startEventTap() {
	CGEventMask mask = CGEventMaskBit(kCGEventKeyDown)
		| CGEventMaskBit(kCGEventKeyUp)
		| CGEventMaskBit(kCGEventMouseMoved)
		| CGEventMaskBit(kCGEventLeftMouseDown)
		| CGEventMaskBit(kCGEventLeftMouseUp)
		| CGEventMaskBit(kCGEventRightMouseDown)
		| CGEventMaskBit(kCGEventRightMouseUp)
		| CGEventMaskBit(kCGEventOtherMouseDown)
		| CGEventMaskBit(kCGEventOtherMouseUp)
		| CGEventMaskBit(kCGEventScrollWheel)
		| CGEventMaskBit(kCGEventLeftMouseDragged)
		| CGEventMaskBit(kCGEventRightMouseDragged)
		| CGEventMaskBit(kCGEventOtherMouseDragged);

	return CGEventTapCreate(kCGSessionEventTap,
		kCGHeadInsertEventTap,
		kCGEventTapOptionListenOnly,
		mask,
		eventTapCallback,
		NULL);
}

static void runEventTap(CFMachPortRef tap) {
	CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
	CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
	CGEventTapEnable(tap, true);
	CFRunLoopRun();
}
*/
import "C"

import (
	"os"
	"unsafe"

	"github.com/TKMAX777/RemoteRelativeInput/debug"
	"github.com/TKMAX777/RemoteRelativeInput/keymap"
	"github.com/TKMAX777/RemoteRelativeInput/remote_send"
)

func startEventTap() {
	tap := C.startEventTap()
	if tap == nil {
		debug.Debugln("Failed to create event tap. Ensure Input Monitoring permission is granted.")
		os.Exit(1)
	}

	C.runEventTap(tap)
}

//export eventTapCallback
func eventTapCallback(proxy C.CGEventTapProxy, eventType C.CGEventType, event C.CGEventRef, refcon unsafe.Pointer) C.CGEventRef {
	state := activeClient
	if state == nil {
		return event
	}

	switch eventType {
	case C.kCGEventTapDisabledByTimeout, C.kCGEventTapDisabledByUserInput:
		return event
	case C.kCGEventKeyDown:
		handleKeyEvent(state, event, remote_send.KeyDown)
	case C.kCGEventKeyUp:
		handleKeyEvent(state, event, remote_send.KeyUp)
	case C.kCGEventMouseMoved, C.kCGEventLeftMouseDragged, C.kCGEventRightMouseDragged, C.kCGEventOtherMouseDragged:
		handleMouseMove(state, event)
	case C.kCGEventLeftMouseDown:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x01, remote_send.KeyDown)
	case C.kCGEventLeftMouseUp:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x01, remote_send.KeyUp)
	case C.kCGEventRightMouseDown:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x02, remote_send.KeyDown)
	case C.kCGEventRightMouseUp:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x02, remote_send.KeyUp)
	case C.kCGEventOtherMouseDown:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x04, remote_send.KeyDown)
	case C.kCGEventOtherMouseUp:
		state.sendInput(keymap.EV_TYPE_MOUSE, 0x04, remote_send.KeyUp)
	case C.kCGEventScrollWheel:
		handleScroll(state, event)
	}

	return event
}

func handleKeyEvent(state *clientState, event C.CGEventRef, keyState remote_send.InputType) {
	keycode := uint16(C.CGEventGetIntegerValueField(event, C.kCGKeyboardEventKeycode))
	windowsKey, ok := windowsKeyFromMacKeycode(keycode)
	if !ok {
		return
	}

	state.sendInput(keymap.EV_TYPE_KEY, windowsKey, keyState)
}

func handleMouseMove(state *clientState, event C.CGEventRef) {
	deltaX := int32(C.CGEventGetIntegerValueField(event, C.kCGMouseEventDeltaX))
	deltaY := int32(C.CGEventGetIntegerValueField(event, C.kCGMouseEventDeltaY))

	if state.isRelative && (deltaX != 0 || deltaY != 0) {
		state.remote.SendRelativeCursor(deltaX, deltaY)
	}
}

func handleScroll(state *clientState, event C.CGEventRef) {
	delta := C.CGEventGetIntegerValueField(event, C.kCGScrollWheelEventDeltaAxis1)
	if delta == 0 {
		return
	}

	if delta > 0 {
		state.sendInput(keymap.EV_TYPE_WHEEL, 0, remote_send.KeyUp)
	} else {
		state.sendInput(keymap.EV_TYPE_WHEEL, 0, remote_send.KeyDown)
	}
}
