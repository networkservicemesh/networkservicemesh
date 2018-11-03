## Stopwatch

A simple utility able to log measured time periods of various events. To create a new stopwatch, call:

`sw := NewStopwatch(name string, log logging.Logger)`

Stopwatch object can store a new entry with `sw.LogTimeEntry(n interface{}, d time.Duration)` where `n` is
a string representation of a measured entity (name of a function, struct or just simple string) and `d` is
time duration. If the name already exists, it will be indexed (for example _name#1_)

Use `sw.Print()` to print all measurements and clear stored entries.  