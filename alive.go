// alive helps servers to coordinate graceful or fast stopping
package alive

import (
	"fmt"
	"sync"
	"sync/atomic"
)

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
	chStop   chan struct{}
	chFinish chan struct{}
}

func NewAlive() *Alive {
	self := &Alive{
		state:    stateRunning,
		chStop:   make(chan struct{}, 1),
		chFinish: make(chan struct{}, 1),
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

func push(ch chan struct{}) {
	for {
		select {
		case ch <- struct{}{}:
			continue
		default:
			return
		}
	}
}

func pull(ch chan struct{}) {
	<-ch
	push(ch)
}

func (self *Alive) Stop() {
	self.lk.Lock()
	defer self.lk.Unlock()
	state := atomic.LoadUint32(&self.state)
	switch state {
	case stateRunning:
		atomic.StoreUint32(&self.state, stateStopping)
		push(self.chStop)
		go self.finish()
		return
	case stateStopping, stateFinished:
		return
	}
	panic(formatBugState(state, "Stop"))
}

func (self *Alive) finish() {
	self.WaitTasks()
	self.lk.Lock()
	defer self.lk.Unlock()
	state := atomic.LoadUint32(&self.state)
	switch state {
	case stateFinished:
		return
	case stateStopping:
		atomic.StoreUint32(&self.state, stateFinished)
		push(self.chFinish)
		return
	}
	panic(formatBugState(state, "finish"))
}

func (self *Alive) StopChan() <-chan struct{} {
	ch := make(chan struct{})
	go func(ch1, ch2 chan struct{}) {
		pull(ch1)
		push(ch2)
	}(self.chStop, ch)
	return ch
}

func (self *Alive) WaitChan() <-chan struct{} {
	ch := make(chan struct{})
	go func(w func(), ch1, ch2 chan struct{}) {
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
