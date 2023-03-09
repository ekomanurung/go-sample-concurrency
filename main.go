package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

func main() {
	//concurrencyDefault()
	//concurrencyUsingChannel()
	//concurrencyUsingWaitGroup()
	concurrencyUsingWaitGroupWithResponse()
}

func concurrencyDefault() {
	go printerA("eko")
	go printerA("januardy")
	//add timer to wait the goroutine finished
	time.Sleep(2 * time.Second)
}

// instead of using time sleep, we can use channel to wait the goroutine to finish
// but still sequential need to improve for multiple calls
func concurrencyUsingChannel() {
	//use channel
	ch := make(chan bool)
	go printerB("eko", ch)
	<-ch
	go printerB("januardy", ch)
	<-ch
}

// call concurrency using waitgroup, handling errors panics
// only support void function
func concurrencyUsingWaitGroup() {
	var wg sync.WaitGroup

	wg.Add(10)
	for i := 0; i < 10; i++ {
		//clone because the original is executed for all goroutine
		cloneData := i
		go func() {
			defer wg.Done()
			//catch any panic from method execution, if no panic this defer func is not executed
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Panics...")
				}
			}()

			printNumber(cloneData)
		}()
	}
	wg.Wait()
}

func concurrencyUsingWaitGroupWithResponse() {
	var wg sync.WaitGroup

	resChan := make(chan string)
	//create channel to handle error
	errChan := make(chan error)
	wg.Add(10)
	for i := 0; i < 10; i++ {
		//clone because the original is executed for all goroutine
		cloneData := i
		go func() {
			defer wg.Done()
			//catch any panic from method execution, if no panic this defer func is not executed
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("Panics Happened")
					return
				}
			}()

			res, err := getValue(cloneData)
			if err != nil {
				errChan <- err
				return
			}
			resChan <- res
		}()
	}

	//deadlock happened if we don't handle the wait inside goroutine
	go func() {
		wg.Wait()
		close(errChan)
		close(resChan)
	}()

	//change logic to handle both error & result
	for {
		select {
		case ex, ok := <-errChan:
			if !ok {
				return
			}
			fmt.Println(ex)
			return
		case res, ok := <-resChan:
			if !ok {
				return
			}
			fmt.Println(res)
		}
	}
}

func getValue(i int) (string, error) {
	if i%2 == 0 {
		return "", errors.New("exception returned")
	}
	return fmt.Sprintf("this your value %d", i), nil
}

func printNumber(i int) {
	if i%2 == 0 {
		panic("some panic example")
	}
	fmt.Printf("Your number is %d\n", i)
}

func printerA(word string) {
	fmt.Printf("Hello %s\n", word)
}

func printerB(word string, ch chan<- bool) {
	fmt.Printf("Hello %s\n", word)
	ch <- true
}
