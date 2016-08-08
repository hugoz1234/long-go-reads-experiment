//slowreads
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

/*
	Idea: when a read op acquires an RLock, other read operations can concurrently
	read shared data until a write op calls for a Lock. Write locks can progress once
	the final RLock is released. Reads cannot progress during a write op.

	Goal: Test latency caused by long reads
*/

//log reads
//e^normal dist
// rel between sleep time and time to axquire

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type readData struct {
	timetoAcquire time.Duration
	sleepTime     time.Duration
}

func main() {

	var lock = &sync.RWMutex{}

	writeticker := time.NewTicker(time.Second) //write every second
	quit := make(chan struct{})
	durationOfExperiment := 60
	readers := 5

	writeTimetoAcquire := make(chan time.Duration, durationOfExperiment)
	rData := make(chan readData, 10000)

	/*************************** n readers with random sleep time*********************/
	for i := 0; i < readers; i++ {
		go func(i int) {
			for {
				sleepTime := time.Duration(math.Exp(rand.NormFloat64()))
				now := time.Now()
				lock.RLock()
				timetoAcquire := time.Since(now)
				time.Sleep(time.Second * sleepTime)
				lock.RUnlock()

				rData <- readData{timetoAcquire, sleepTime}
				runtime.Gosched()
			}
		}(i)
	}

	/*********************** 1 writer **********************/
	go func() {
		for {
			select {
			case <-writeticker.C:
				now := time.Now()
				lock.Lock()
				timetoAcquire := time.Since(now)
				lock.Unlock()

				writeTimetoAcquire <- timetoAcquire

			case <-quit:
				fmt.Println("I AM HERE")
				writeticker.Stop()
				writeTimetoAcquire <- time.Duration(0)
				return
			}
		}
	}()

	/****************recording happens here***************/
	f, err := os.OpenFile("writeData.txt", os.O_RDWR|os.O_APPEND, 0666)
	check(err)
	dataHeader := "\n\nNumber of readers: " + strconv.Itoa(readers) + "\n"
	f.WriteString(dataHeader)

	ff, err := os.OpenFile("readData.txt", os.O_RDWR|os.O_APPEND, 0666)
	check(err)
	ff.WriteString(dataHeader)

	sleepTimes := make([]time.Duration, 0)

	defer f.Close()
	defer ff.Close()

	writes := 0
	reads := 0

	//write recorder
	go func() {
		for {
			select {
			case <-quit:
				return
			default:
				tta := <-writeTimetoAcquire
				fmt.Println("Time to acquire write: ", tta)
				writes += 1
				_, err := f.WriteString(tta.String() + "\n")
				check(err)
			}

		}
	}()

	//read recorder

	go func() {
		for {
			select {
			case <-quit:
				return
			default:
				readData := <-rData
				tta := readData.timetoAcquire
				st := readData.sleepTime
				sleepTimes = append(sleepTimes, st)
				reads += 1
				_, err := ff.WriteString(tta.String() + "\n")
				check(err)
			}
		}
	}()

	time.Sleep(time.Second * time.Duration(durationOfExperiment))
	close(quit)

	/*****************Upon finishing the experiment***********/
	_, err = ff.WriteString("\n **SleepTimes \n")
	check(err)
	for i := 0; i < len(sleepTimes); i++ {
		_, err = ff.WriteString(sleepTimes[i].String() + "\n")
		check(err)
	}

	_, err = f.WriteString("Number of writes: " + strconv.Itoa(writes) + "\n")
	check(err)
	_, err = ff.WriteString("Number of reads: " + strconv.Itoa(reads) + "\n")
	check(err)

	fmt.Println("DONE")
}
