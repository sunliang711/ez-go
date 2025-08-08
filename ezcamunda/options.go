package ezcamunda

import "time"

type Option func(*startOptions)

type startOptions struct {
	concurrency   int
	maxJobActives int
	timeout       time.Duration
	pollInternal  time.Duration
}

func defaultStartOptions() startOptions {
	return startOptions{
		concurrency:   2,
		maxJobActives: 2,
		timeout:       time.Second * 10,
		pollInternal:  time.Second * 10,
	}
}

func WithConcurrency(concurrency int) Option {
	return func(so *startOptions) {
		so.concurrency = concurrency
	}
}

func WithMaxJobsActive(m int) Option {
	return func(so *startOptions) {
		so.maxJobActives = m
	}
}

func WithTimeout(t time.Duration) Option {
	return func(so *startOptions) {
		so.timeout = t
	}
}

func WithPoolInterval(t time.Duration) Option {
	return func(so *startOptions) {
		so.pollInternal = t
	}
}
