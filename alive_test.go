package alive

import (
	"testing"
	"time"
)

func Test01(t *testing.T) {
	{
		a := NewAlive()
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
	a.Stop()
	ch := make(chan int, 2)
	go func() {
		time.Sleep(1 * time.Second)
		ch <- 1
		a.Done()
	}()
	go func() { a.Wait(); ch <- 2 }()
	x1, x2 := <-ch, <-ch
	if x1 != 1 || x2 != 2 {
		t.Fatal("Alive.Wait must wait WaitGroup tasks")
	}
}

// TODO: test Stop(); go Stop(); Wait()
