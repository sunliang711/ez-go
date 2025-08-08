package ezcamunda

import (
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

type CamundaWorker struct {
	JobType       string
	WorkerName    string
	JobHandler    worker.JobHandler
	Concurrency   int
	MaxJobActives int
	Timeout       time.Duration
	PollInterval  time.Duration
}

func NewCamundaWorker(jobType, workerName string, jobHandler worker.JobHandler, concurrency, maxJobActives int, timeout, pollInterval time.Duration) *CamundaWorker {
	return &CamundaWorker{
		JobType:       jobType,
		WorkerName:    workerName,
		JobHandler:    jobHandler,
		Concurrency:   concurrency,
		MaxJobActives: maxJobActives,
		Timeout:       timeout,
		PollInterval:  pollInterval,
	}
}

type JobWorker struct {
	Name   string
	Worker worker.JobWorker
}
