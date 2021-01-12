package main

import (
        "syscall"
        "fmt"
        "time"
        "os"
        "strconv"
)

func GetCPU() int64 {
    usage := new(syscall.Rusage)
    syscall.Getrusage(syscall.RUSAGE_SELF, usage)
    return usage.Utime.Nano() + usage.Stime.Nano()
}

func measured(f func()) (float64, float64) {
        startCPU := GetCPU()
        startTime := time.Now()

        f()

        stopCPU := GetCPU()
        return toMilliSeconds((stopCPU-startCPU)/1000), toMilliSeconds(time.Since(startTime).Microseconds())
}

type RingMessage int
type ControlMessage int

func ringElemLoop(inbox <-chan RingMessage, next chan<- RingMessage) {
        for r := range inbox {
                next <- r
        }
        close(next)
}

func createNextRingElem(N int, ringHeadInbox chan RingMessage, done chan<- struct{}) chan RingMessage {
        if N <= 0 {
                //for ring size 1 this is executed in the ringhead goroutine
                //and so it would block if we send synchronously.
                //so send asynchronously.
                go func() {
                        done <- struct{}{}
                }()
                return ringHeadInbox
        }
        nextInbox := make(chan RingMessage)
        go func() {
                next := createNextRingElem(N-1, ringHeadInbox, done)
                ringElemLoop(nextInbox, next)
        }()
        return nextInbox
}

type Ring struct {
        done <-chan struct{}
        nextMessage chan<- ControlMessage
        waitOnDestroy <-chan struct{}
}

func createRing(N int) (error, *Ring) {
        if (N <= 0) {
                return fmt.Errorf("Creating a ring with %v elems? Feeling funny today?", N), nil
        }
        nextMessage := make(chan ControlMessage)
        roundtripDone := make(chan struct{})

        ringHeadInbox := make(chan RingMessage)
        done := make(chan struct{})
        ringDestroyed := make(chan struct{})

        next := createNextRingElem(N-1, ringHeadInbox, done)
        go ringHeadLoop(ringHeadInbox, next, nextMessage, roundtripDone, ringDestroyed)

        <-done // wait for ring construction

        return nil, &Ring{nextMessage:nextMessage, done:roundtripDone, waitOnDestroy:ringDestroyed}
}

func ringHeadLoop(inbox <-chan RingMessage, next chan<- RingMessage, nextMessage <-chan ControlMessage, roundtripDone chan<- struct{}, ringDestroyed chan<- struct{}) {
        for {
                select {
                case x, ok := <-inbox:
                        if !ok {
                                // ring was destructed
                                ringDestroyed <- struct{}{}
                                return
                        }
                        if x > 0 {
                            //it could always be the case, we send this message
                            //to ourself (ringsize 1), so we have to do it
                            //asynchronously
                            go func() {
                                    next <- x-1
                            }()
                        } else {
                            roundtripDone <- struct{}{}
                        }
                case x, ok := <-nextMessage:
                        if !ok {
                                close(next)
                                nextMessage = nil // so this clause blocks now forever
                        } else {
                                go func() {
                                        next <- RingMessage(int(x)-1)
                                }()
                        }
                }
        }
}

func sendAndWait(N int, ring Ring) {
        ring.nextMessage <- ControlMessage(N)
        <-ring.done
}

func destroyAndWait(ring Ring) {
        close(ring.nextMessage)
        <-ring.waitOnDestroy
}

func toMilliSeconds(micros int64) float64 {
        return float64(micros)/1000
}

func main() {
        if (len(os.Args) < 3) {
                fmt.Printf("Usage: %s ringSize roundtrips\n", os.Args[0])
                return
        }
        ringSize, err := strconv.Atoi(os.Args[1])
        if err != nil {
                fmt.Printf("couldn't convert ringSize %s into a number: %v\n", os.Args[1], err)
                return
        }
        roundtrips, err := strconv.Atoi(os.Args[2])
        if err != nil {
                fmt.Printf("couldn't convert roundtrips %s into a number: %v\n", os.Args[2], err)
                return
        }

        var ring *Ring
        cpu, wallclock := measured(func () {
           err, ring = createRing(ringSize)
        })
        if err != nil {
                fmt.Print(err)
                return
        }
        fmt.Printf("ring creation took %.3f (%.3f) milliseconds\n", cpu, wallclock)

        cpu,wallclock = measured(func() {
                sendAndWait(roundtrips, *ring)
        })

        fmt.Printf("%v roundtrips took %.3f (%.3f) milliseconds\n", roundtrips, cpu, wallclock)

        cpu,wallclock = measured(func() {
                destroyAndWait(*ring)
        })
        fmt.Printf("destruction of the ring took %.3f (%.3f) milliseconds\n", cpu, wallclock)
}
