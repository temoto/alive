// alive helps servers to coordinate graceful or fast stopping
package alive

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type nothing struct{}

const (
	stateRunning = iota
	stateStopping
	stateFinished
)

func stateString(s uint32) string {
	switch s {
	case stateRunning:
		return "running"
	case stateStopping:
		return "stopping"
	case stateFinished:
		return "finished"
	}
	return "unknown!"
}

func formatBugState(state uint32, source string) string {
	return fmt.Sprintf(`Bug in package 'alive': unexpected state value %d (%s) in %s.
Please post minimal reproduction code on Github issues, see package import path.`,
		state, stateString(state), source)
}

type Alive struct {
	wg       sync.WaitGroup
	state    uint32
	lk       sync.Mutex
	chStop   chan nothing
	chFinish chan nothing
}

func NewAlive() *Alive {
	self := &Alive{
		state:    stateRunning,
		chStop:   make(chan nothing, 1),
		chFinish: make(chan nothing, 1),
	}
	return self
}

func (self *Alive) Add(delta int) {
	state := atomic.LoadUint32(&self.state)
	switch state {
	case stateRunning:
		self.wg.Add(delta)
		return
	case stateStopping, stateFinished:
		panic("Alive.Add(): need state Running. Attempted to run new task after Stop().")
	}
	panic(formatBugState(state, "Add"))
}

func (self *Alive) Done() {
	state := atomic.LoadUint32(&self.state)
	switch state {
	case stateRunning, stateStopping:
		self.wg.Done()
		return
	}
	panic(formatBugState(state, "Done"))
}

func (self *Alive) IsRunning() bool  { return atomic.LoadUint32(&self.state) == stateRunning }
func (self *Alive) IsStopping() bool { return atomic.LoadUint32(&self.state) == stateStopping }
func (self *Alive) IsFinished() bool { return atomic.LoadUint32(&self.state) == stateFinished }

func push(ch chan nothing) {
	for {
		select {
		case ch <- nothing{}:
			continue
		default:
			return
		}
	}
}

func pull(ch chan nothing) {
	<-ch
	push(ch)
}

func (self *Alive) Stop() {
	self.lk.Lock()
	defer self.lk.Unlock()
	switch self.state {
	case stateRunning:
		self.state = stateStopping
		push(self.chStop)
		go self.finish()
		return
	case stateStopping, stateFinished:
		return
	}
	panic(formatBugState(self.state, "Stop"))
}

func (self *Alive) finish() {
	self.lk.Lock()
	defer self.lk.Unlock()
	switch self.state {
	case stateStopping:
		self.WaitTasks()
		self.state = stateFinished
		push(self.chFinish)
		return
	}
	panic(formatBugState(self.state, "finish"))
}

func (self *Alive) StopChan() <-chan nothing {
	ch := make(chan nothing)
	go func(ch1, ch2 chan nothing) {
		pull(ch1)
		push(ch2)
	}(self.chStop, ch)
	return ch
}

func (self *Alive) WaitChan() <-chan nothing {
	ch := make(chan nothing)
	go func(w func(), ch1, ch2 chan nothing) {
		w()
		pull(ch1)
		push(ch2)
	}(self.WaitTasks, self.chFinish, ch)
	return ch
}

func (self *Alive) WaitTasks() {
	self.wg.Wait()
}

func (self *Alive) Wait() {
	self.WaitTasks()
	pull(self.chFinish)
}

func (self *Alive) String() string {
	return fmt.Sprintf("state=%s",
		stateString(atomic.LoadUint32(&self.state)),
	)
}
