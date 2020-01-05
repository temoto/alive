package alive

import (
	"strconv"
	"sync"
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
	requireBool(t, true, a.Add(1))
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
	requireBool(t, true, a.Add(1))
	go a.Stop()
	<-stopch
	expectStateStopping(t, a)
	<-stopch
	a.Done()
}

func TestMultiWaitChan(t *testing.T) {
	a := NewAlive()
	expectStateRunning(t, a)
	requireBool(t, true, a.Add(1))
	waitch := a.WaitChan()
	go a.Stop()
	go a.Done()
	<-waitch
	expectStateFinished(t, a)
	<-waitch
}

func TestAdd(t *testing.T) {
	a := NewAlive()
	requireBool(t, true, a.Add(1))
}

func TestAddAfterStop(t *testing.T) {
	a := NewAlive()
	a.Stop()
	requireBool(t, false, a.Add(1))
}

func BenchmarkAdd(b *testing.B) {
	for _, c := range []int{1, 4, 16, 64, 256} {
		b.Run(strconv.Itoa(c), func(b *testing.B) {
			a := NewAlive()
			b.ReportAllocs()
			b.ResetTimer()
			wg := sync.WaitGroup{}
			wg.Add(c)
			for p := 1; p <= c; p++ {
				go func() {
					for i := 1; i <= b.N; i++ {
						requireBool(b, true, a.Add(1))
						a.Done()
					}
					wg.Done()
				}()
			}
			wg.Wait()
			a.Stop()
			a.WaitTasks()
		})
	}
}

func BenchmarkConcurrentStopChan(b *testing.B) {
	a := NewAlive()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 1; i <= b.N; i++ {
		requireBool(b, true, a.Add(1))
		go func() {
			<-a.StopChan()
			a.Done()
		}()
	}
	a.Stop()
	a.WaitTasks()
}

func requireBool(t testing.TB, expect, actual bool) bool {
	t.Helper()
	if actual != expect {
		t.Fatalf("expected %t", expect)
		return false
	}
	return true
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
