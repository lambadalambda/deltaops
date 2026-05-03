//go:build windows

package runtime

import (
	"os"
	"os/signal"
	"sync"
)

type OSSignalSource struct {
	done     chan struct{}
	stop     chan struct{}
	signals  chan os.Signal
	once     sync.Once
	stopOnce sync.Once
}

func NewOSSignalSource() *OSSignalSource {
	source := &OSSignalSource{done: make(chan struct{}), stop: make(chan struct{}), signals: make(chan os.Signal, 1)}
	signal.Notify(source.signals, os.Interrupt)
	go func() {
		select {
		case <-source.signals:
			source.close()
		case <-source.stop:
		}
	}()
	return source
}

func (s *OSSignalSource) Done() <-chan struct{} {
	return s.done
}

func (s *OSSignalSource) Stop() {
	signal.Stop(s.signals)
	s.stopOnce.Do(func() { close(s.stop) })
	s.close()
}

func (s *OSSignalSource) close() {
	s.once.Do(func() { close(s.done) })
}
