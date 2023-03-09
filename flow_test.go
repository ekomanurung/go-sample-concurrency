package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type result struct {
	c       int
	data    string
	numbers map[int]bool
	orders  []int
}

func TestSequentialFlow(t *testing.T) {
	t.Run("with multiple request success", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				fmt.Printf("Requested number is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}

		assert.NoError(t, ExecSeq(context.Background(), steps...))
	})

	t.Run("with multiple request failed", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				if n == 8 {
					return errors.New("failed execute request")
				}
				fmt.Printf("Requested number is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}

		assert.Error(t, ExecSeq(context.Background(), steps...))
	})

	t.Run("with multiple request delay response", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					time.Sleep(time.Duration(100+n) * time.Millisecond)
				}
				fmt.Printf("Requested number is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}
		assert.NoError(t, ExecSeq(context.Background(), steps...))
	})

	t.Run("with multiple request save response and delay", func(t *testing.T) {
		myResult := &result{numbers: map[int]bool{}}
		myfunc := func(n int, res *result) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					time.Sleep(time.Duration(100+n) * time.Millisecond)
				}
				res.numbers[n] = true
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i, myResult))
		}

		assert.NoError(t, ExecSeq(context.Background(), steps...))
		assert.Equal(t, 10, len(myResult.numbers))
	})
}

func TestConcurrentFlow(t *testing.T) {
	t.Run("run multiple request without response success", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				fmt.Printf("Requested number is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}
		assert.NoError(t, ExecCon(context.Background(), steps...))
	})

	t.Run("run multiple request without response success ordered by fast response", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					d := time.Duration(100+n) * time.Millisecond
					time.Sleep(d)
				}
				fmt.Printf("Requested number is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}
		assert.NoError(t, ExecCon(context.Background(), steps...))
	})

	t.Run("run multiple request without response flaky", func(t *testing.T) {
		myfunc := func(n int) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					return errors.New("failed executed")
				}
				fmt.Printf("Requested number success is %d\n", n)
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i))
		}
		assert.Error(t, ExecCon(context.Background(), steps...))
	})

	t.Run("run multiple request with response success", func(t *testing.T) {
		cRes := &result{numbers: map[int]bool{}}
		lock := make(chan struct{}, 1)
		myfunc := func(n int, res *result) IStep {
			return func(ctx context.Context) error {
				lock <- struct{}{}
				res.numbers[n] = true
				<-lock
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i, cRes))
		}
		assert.NoError(t, ExecCon(context.Background(), steps...))
		assert.Equal(t, 10, len(cRes.numbers))
	})

	t.Run("run multiple request with response success order by fastest", func(t *testing.T) {
		cRes := &result{numbers: map[int]bool{}, orders: []int{}}
		lock := make(chan struct{}, 1)
		myfunc := func(n int, res *result) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					time.Sleep(time.Duration(100+n) * time.Millisecond)
				}
				lock <- struct{}{}
				res.numbers[n] = true
				cRes.orders = append(cRes.orders, n)
				<-lock
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i, cRes))
		}
		assert.NoError(t, ExecCon(context.Background(), steps...))
		assert.Equal(t, 10, len(cRes.numbers))
		fmt.Printf("orders are: %v\n", cRes.orders)
	})

	t.Run("run multiple request with response flaky", func(t *testing.T) {
		cRes := &result{numbers: map[int]bool{}, orders: []int{}}
		lock := make(chan struct{}, 1)
		myfunc := func(n int, res *result) IStep {
			return func(ctx context.Context) error {
				if n%2 == 0 {
					return errors.New("flaky errors")
				}
				lock <- struct{}{}
				res.numbers[n] = true
				cRes.orders = append(cRes.orders, n)
				<-lock
				return nil
			}
		}
		steps := []Step{}
		for i := 1; i <= 10; i++ {
			steps = append(steps, myfunc(i, cRes))
		}
		assert.Error(t, ExecCon(context.Background(), steps...))
		fmt.Printf("numbers are: %v\n", cRes.numbers)
	})
}

func TestExecuteFlowWithStep(t *testing.T) {

	res := &result{}
	t.Run("success", func(t *testing.T) {

		var addFunc = func(a, b int, res *result) IStep {
			return func(ctx context.Context) error {
				res.c = a + b
				return nil
			}
		}
		assert.NoError(t, Exec(context.Background(), addFunc(5, 10, res)))
		assert.Equal(t, 15, res.c)
	})

	t.Run("failed", func(t *testing.T) {
		var addFunc = func(a, b int, res *result) IStep {
			return func(ctx context.Context) error {
				return errors.New("failed to add number")
			}
		}
		assert.Error(t, Exec(context.Background(), addFunc(5, 10, res)))
	})
}
