package ui

import (
	"fmt"
	"time"
)

// Spinner animates a loading indicator in the terminal.
// For full TUI, use charmbracelet/bubbles spinner model.
// This is a lightweight stdout spinner for non-TUI contexts.
type Spinner struct {
	frames  []string
	msg     string
	stop    chan struct{}
	done    chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(msg string) *Spinner {
	return &Spinner{
		frames: spinnerFrames,
		msg:    msg,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start begins the spinner animation in a goroutine.
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Printf("\r%-60s\r", "") // clear line
				return
			default:
				frame := StyleChain.Render(s.frames[i%len(s.frames)])
				fmt.Printf("\r%s  %s", frame, s.msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
}

// Stop halts the spinner and waits for it to finish.
func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}

// StopWithMsg halts the spinner and prints a final message.
func (s *Spinner) StopWithMsg(msg string) {
	s.Stop()
	fmt.Println(msg)
}
