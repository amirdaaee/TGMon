// Package stream exposes a simple container to access a worker pool from
// components that only need worker-level operations.
package stream

// IWorkerContainer provides accessors to the underlying worker pool and
// a convenience method to fetch the next worker directly.
//
//go:generate mockgen -source=container.go -destination=../../mocks/stream/container.go -package=mocks
type IWorkerContainer interface {
	GetWorkerPool() IWorkerPool
	GetNextWorker() IWorker
}
type workerContainer struct {
	wp IWorkerPool
}

var _ IWorkerContainer = (*workerContainer)(nil)

// GetWorkerPool returns the underlying worker pool.
func (c *workerContainer) GetWorkerPool() IWorkerPool {
	return c.wp
}

// GetNextWorker returns the next worker from the pool.
func (c *workerContainer) GetNextWorker() IWorker {
	return c.wp.GetNextWorker()
}

// NewWorkerContainer wraps a worker pool, exposing a minimal interface.
func NewWorkerContainer(wp IWorkerPool) IWorkerContainer {
	return &workerContainer{wp: wp}
}
