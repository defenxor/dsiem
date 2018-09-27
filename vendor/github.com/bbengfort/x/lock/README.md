# Locks

This package is a diagnostic package to track where lock contention occurs. When used instead of a `sync.RWMutex`, you can register the calling method's name, then print a report that shows which locks are currently being held.

Basic usage:

```go
type MyStruct struct {
    lock.RWLock
}

def (m *MyStruct) MyFunc() {
    m.Lock("myfunc")
    defer m.Unlock("myfunc")
}
```

This package adds a bit of overhead to the locking process, so it is really only used for diagnostics.

## Future Work

Future versions of this package will use [`runtime.Caller`](https://stackoverflow.com/questions/35212985/is-it-possible-get-information-about-caller-function-in-golang) to identify which function is calling lock, making `lock.Lock` hot swappable with `sync.Mutex`!
