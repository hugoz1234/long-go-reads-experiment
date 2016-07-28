//slowreads
package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
	"sync"
	"os"
	"strconv"
	)

/*
	Idea: when a read op acquires an RLock, other read operations can concurrently
	read shared data until a write op calls for a Lock. Write locks can progress once 
	the final RLock is released. Reads cannot progress during a write op.
	
	Goal: Test latency caused by long reads
*/



func check(e error) {
    if e != nil {
        panic(e)
    }
}

func main(){

	var lock = &sync.RWMutex{}

    writeticker := time.NewTicker(time.Second) //write every second
	quit := make(chan struct{})
	durationOfExperiment := 30
	readers := 5

	durations := make(chan time.Duration, durationOfExperiment)

	/*************************** n readers with random sleep time*********************/
	for i := 0; i<readers; i++{
		go func(i int) {
		    for {
	        	lock.RLock()
		        multiplier := time.Duration(math.Abs(rand.NormFloat64()))
		        time.Sleep(time.Second * multiplier)
		        lock.RUnlock()

		        runtime.Gosched()
		    }
		 }(i)
	}

	/*********************** 1 writer **********************/
	go func(){
		for {
			select{
				case <- writeticker.C:
					now := time.Now()
					lock.Lock()
					lock.Unlock()
					duration := time.Since(now)
					
					durations <- duration

				case <- quit:
					fmt.Println("I AM HERE")
					writeticker.Stop()
					durations <- time.Duration(0)
					return
			}
		}
	}()

	/****************recording happens hear***************/
	f, err := os.OpenFile("data.txt", os.O_RDWR|os.O_APPEND, 0666)
	dataHeader := "\n\nNumber of readers: " + strconv.Itoa(readers) + "\n"
	f.WriteString(dataHeader)
    check(err)
    defer f.Close()

    writes := 0
	durationAverage := time.Duration(time.Second*0)

	go func(){
		for {
			select {
			case <- quit:
				return
			default:
				duration := <-durations
				fmt.Println("Duration of write: ", duration)
	    		writes+=1
	    		durationAverage +=duration
	    		_, err := f.WriteString(duration.String()+"\n")
	    		check(err)
			}

    	}
	}()
	
	/*****************Upon finishing the experiment***********/
    time.Sleep(time.Second*time.Duration(durationOfExperiment)) 
    close(quit)

    durationAverage= time.Duration(time.Duration(durationAverage.Seconds()/float64(writes)) * time.Second)
    _,err = f.WriteString("Number of writes: " + strconv.Itoa(writes) + "\n"+ "Duration Average: " + durationAverage.String())
    check(err)

    fmt.Println("DONE")
}