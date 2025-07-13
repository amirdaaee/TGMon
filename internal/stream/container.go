package stream

//go:generate mockgen -source=container.go -destination=../../mocks/stream/container.go -package=mocks
type IWorkerContainer interface {
	GetWorkerPool() IWorkerPool
	GetNextWorker() IWorker
}
type workerContainer struct {
	wp IWorkerPool
}

var _ IWorkerContainer = (*workerContainer)(nil)

func (c *workerContainer) GetWorkerPool() IWorkerPool {
	return c.wp
}
func (c *workerContainer) GetNextWorker() IWorker {
	return c.wp.GetNextWorker()
}
func NewWorkerContainer(wp IWorkerPool) IWorkerContainer {
	return &workerContainer{wp: wp}
}
