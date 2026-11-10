package outbox

import "time"

// Option configures a Drainer.
type Option func(*Drainer)

// WithInterval sets the pause between two drain passes. Default 2 seconds.
func WithInterval(d time.Duration) Option {
	return func(dr *Drainer) { dr.interval = d }
}

// WithBatchSize caps the rows one pass locks and sends. Default 100.
func WithBatchSize(n int) Option {
	return func(dr *Drainer) { dr.batchSize = n }
}

// WithRetention sets how long a sent row is kept before it is pruned. Default 7 days.
func WithRetention(d time.Duration) Option {
	return func(dr *Drainer) { dr.retention = d }
}

// WithMaxBackoff caps the pause after consecutive failed passes. Default 1 minute.
func WithMaxBackoff(d time.Duration) Option {
	return func(dr *Drainer) { dr.maxBackoff = d }
}

// WithOnError receives every failed pass: an emit failure or a database error.
func WithOnError(fn func(error)) Option {
	return func(dr *Drainer) { dr.onError = fn }
}
