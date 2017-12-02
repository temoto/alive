What
====

alive waits for subtasks, coordinate graceful or fast shutdown. sync.WaitGroup on steroids.

Usage
=====

Key takeaways:

* Zero value of `alive.Alive{}` is *not* ready, you *must* use `NewAlive()` constructor.
```
    srv := MyServer{ alive: alive.NewAlive() }
```
* Call `.Add(n)` and `.Done()` just as with `WaitGroup`, monitor `.IsRunning()`.
```
    for srv.alive.IsRunning() {
        task := <-queue
        srv.alive.Add(1)
        go func() {
            // be useful
            srv.alive.Done()
        }()
    }
```
* Call `.Stop()` to switch `IsRunning` and stop creating new tasks if programmed so.
```
    sigShutdownChan := make(chan os.Signal, 1)
    signal.Notify(sigShutdownChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
    go func(ch <-chan os.Signal) {
        <-ch
        log.Printf("graceful stop")
        sdnotify("READY=0\nSTATUS=stopping\n")
        srv.alive.Stop()
    }(sigShutdownChan)
```
* Call `.Wait()` to synchronize on all subtasks `.Done()`, just as with `WaitGroup`.
```
func main() {
    // ...
    srv.alive.Wait()
}
```
* `.StopChan()` lets your observe `.Stop()` call from another place. A better option to `IsRunning()` poll.
```
    stopch := srv.alive.StopChan()
    for {
        select {
        case job := <-queue:
            // be useful
        case <-stopch:
            // break for loop
        }
    }
```
* `.WaitChan()` is `select`-friendly version of `.Wait()`.

Install
=======

`go get -u github.com/temoto/alive`

