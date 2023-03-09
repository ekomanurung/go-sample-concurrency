package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// Step is unit of work. it has a capabilities to be executed
type Step interface {
	Exec(ctx context.Context) error
}

// IStep is a definition of Step in function. we can pass a step as a func argument
type IStep func(ctx context.Context) error

// Exec is implementation from interface Step
func (f IStep) Exec(ctx context.Context) error {
	return f(ctx)
}

// Exec execute a step
func Exec(ctx context.Context, step Step) error {
	return step.Exec(ctx)
}

// Sequential is collection of steps
// Done is a boolean function to break the function from the execution
type Sequential struct {
	Steps []Step
	Done  func() bool
}

// Concurrent is collection steps that will be executed in parallel
type Concurrent struct {
	Steps []Step
}

// ExecCon execute multiple steps in concurrent way
// Break execution if there is an error in particular function, return error
// Don't support atomic operation, need to handle separately on each step/method
func ExecCon(ctx context.Context, step ...Step) error {
	return Exec(ctx, &Concurrent{
		Steps: step,
	})
}

// ExecSeq execute multiple steps in sequential way
func ExecSeq(ctx context.Context, step ...Step) error {
	return Exec(ctx, &Sequential{
		Steps: step,
	})
}

func (c *Concurrent) Exec(ctx context.Context) error {
	if len(c.Steps) == 0 {
		return nil
	}

	done := make(chan struct{})
	defer close(done)

	errChan := make(chan error)

	var wg sync.WaitGroup
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg.Add(len(c.Steps))

	for _, step := range c.Steps {
		//need to clone, so it is copied new object to pass into anonymous function
		s := step
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					c.publishError(done, errChan, fmt.Errorf("flow: panic with %v: stackTrace: %s", r, string(debug.Stack())))
				}
			}()

			if err := s.Exec(childCtx); err != nil {
				c.publishError(done, errChan, err)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-childCtx.Done():
		return childCtx.Err()
	}
}

func (c *Concurrent) publishError(done <-chan struct{}, errChan chan<- error, error error) {
	select {
	case <-done:
	case errChan <- error:
	}
}

func (s *Sequential) Exec(ctx context.Context) error {
	for _, step := range s.Steps {
		if s.Done != nil && s.Done() {
			return nil
		}

		if err := step.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
