package alive

import (
	"testing"
	"time"
)

func Test01(t *testing.T) {
	{
		a := NewAlive()
		expectStateRunning(t, a)
		a.Stop()
		a.Wait()
		a.Stop()
	}
	{
		a := NewAlive()
		a.Stop()
		a.Stop()
		a.Wait()
	}
}

func TestTasks(t *testing.T) {
	a := NewAlive()
	a.Add(1)
	expectStateRunning(t, a)
	a.Stop()
	expectStateStopping(t, a)
	ch := make(chan int, 2)
	go func() {
		time.Sleep(1 * time.Second)
		ch <- 1
		a.Done()
	}()
	a.WaitTasks()
	go func() { a.Wait(); ch <- 2 }()
	x1, x2 := <-ch, <-ch
	expectStateFinished(t, a)
	if x1 != 1 || x2 != 2 {
		t.Fatal("Alive.Wait must wait WaitGroup tasks")
	}
}

func TestConcurrentStop(t *testing.T) {
	a := NewAlive()
	expectStateRunning(t, a)
	go a.Stop()
	go a.Stop()
	go a.IsRunning()
	a.Wait()
	expectStateFinished(t, a)
}

func TestMultiStopChan(t *testing.T) {
	a := NewAlive()
	stopch := a.StopChan()
	a.Add(1)
	go a.Stop()
	<-stopch
	expectStateStopping(t, a)
	<-stopch
	a.Done()
}

func TestMultiWaitChan(t *testing.T) {
	a := NewAlive()
	expectStateRunning(t, a)
	a.Add(1)
	waitch := a.WaitChan()
	go a.Stop()
	go a.Done()
	<-waitch
	expectStateFinished(t, a)
	<-waitch
}

func TestAddAfterStop(t *testing.T) {
	func() { // don't spill panic/recover to testing framework
		a := NewAlive()
		a.Stop()
		defer func() {
			x := recover()
			if s, ok := x.(string); !ok || s != NotRunning {
				t.Errorf("expected panic(NotRunning) recover=%v", x)
			}
		}()
		a.Add(1)
	}()
}

func BenchmarkConcurrentStopChan(b *testing.B) {
	a := NewAlive()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 1; i <= b.N; i++ {
		a.Add(1)
		go func() {
			<-a.StopChan()
			a.Done()
		}()
	}
	a.Stop()
	a.WaitTasks()
}

func expectStateRunning(t testing.TB, a *Alive) {
	t.Helper()
	if !a.IsRunning() {
		t.Errorf("IsRunning() expected=true")
	}
	expectStateString(t, a, "state=running")
}
func expectStateStopping(t testing.TB, a *Alive) {
	t.Helper()
	if !a.IsStopping() {
		t.Errorf("IsStopping() expected=true")
	}
	expectStateString(t, a, "state=stopping")
}
func expectStateFinished(t testing.TB, a *Alive) {
	t.Helper()
	if !a.IsFinished() {
		t.Errorf("IsFinished() expected=true")
	}
	expectStateString(t, a, "state=finished")
}

func expectStateString(t testing.TB, a *Alive, expect string) {
	t.Helper()
	stateString := a.String()
	if stateString != expect {
		t.Errorf("String() expected=%s actual=%s", expect, stateString)
	}
}
