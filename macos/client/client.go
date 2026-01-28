//go:build darwin
// +build darwin

package client

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	relative_input "github.com/TKMAX777/RemoteRelativeInput"
	"github.com/TKMAX777/RemoteRelativeInput/debug"
	"github.com/TKMAX777/RemoteRelativeInput/keymap"
	"github.com/TKMAX777/RemoteRelativeInput/remote_send"
)

const (
	defaultToggleKey = "F8"
	defaultExitKey   = "F12"
)

type ToggleType int

const (
	ToggleTypeOnce ToggleType = iota + 1
	ToggleTypeAlive
)

type clientState struct {
	remote     *remote_send.Handler
	isRelative bool
	toggleType ToggleType
	toggleKey  uint32
	exitKey    uint32
	exitTimer  *time.Timer
}

var activeClient *clientState

func StartClient() {
	defer os.Stdout.Write([]byte("CLOSE\n"))

	debug.Debugln("==== START CLIENT APPLICATION (macOS) ====")
	debug.Debugln("ServerProtocolVersion:", relative_input.PROTOCOL_VERSION)

	remote := remote_send.New(os.Stdout)
	state := &clientState{
		remote:     remote,
		isRelative: true,
		toggleType: ToggleTypeAlive,
	}

	toggleKeyName := os.Getenv("RELATIVE_INPUT_TOGGLE_KEY")
	if toggleKeyName == "" {
		toggleKeyName = defaultToggleKey
	}
	state.toggleKey = parseWindowsKeyValue(toggleKeyName, defaultToggleKey)

	exitKeyName := defaultExitKey
	state.exitKey = parseWindowsKeyValue(exitKeyName, defaultExitKey)

	switch os.Getenv("RELATIVE_INPUT_TOGGLE_TYPE") {
	case "ONCE":
		state.toggleType = ToggleTypeOnce
	default:
		state.toggleType = ToggleTypeAlive
	}

	activeClient = state

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-signalChan
		os.Exit(0)
	}()

	startEventTap()
}

func parseWindowsKeyValue(name string, fallback string) uint32 {
	key, err := keymap.GetWindowsKeyDetailFromEventInput(name)
	if err == nil {
		return key.Value
	}

	key, err = keymap.GetWindowsKeyDetailFromEventInput(fallback)
	if err == nil {
		return key.Value
	}

	return 0
}

func (s *clientState) handleExitKey(state remote_send.InputType) {
	if s.exitKey == 0 {
		return
	}

	switch state {
	case remote_send.KeyDown:
		if s.exitTimer != nil {
			s.exitTimer.Stop()
		}
		s.exitTimer = time.AfterFunc(500*time.Millisecond, func() {
			s.remote.SendExit()
			os.Exit(0)
		})
	case remote_send.KeyUp:
		if s.exitTimer != nil {
			s.exitTimer.Stop()
		}
	}
}

func (s *clientState) toggleRelative(state remote_send.InputType) {
	if s.toggleKey == 0 {
		return
	}

	switch state {
	case remote_send.KeyDown:
		s.isRelative = !s.isRelative || s.toggleType == ToggleTypeOnce
	case remote_send.KeyUp:
		if s.toggleType == ToggleTypeOnce {
			s.isRelative = false
		}
	}
}

func (s *clientState) sendInput(evType keymap.EV_TYPE, key uint32, state remote_send.InputType) {
	if key == s.exitKey {
		s.handleExitKey(state)
	}

	if key == s.toggleKey {
		s.toggleRelative(state)
		return
	}

	if s.isRelative {
		s.remote.SendInput(evType, key, state)
	}
}
