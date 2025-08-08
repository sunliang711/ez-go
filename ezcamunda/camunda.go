package ezcamunda

import (
	"context"
	"fmt"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/commands"
	"github.com/camunda/zeebe/clients/go/v8/pkg/pb"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// usage:
// cli, err := ezcamunda.New(gateway, workers...)
// if err != nil {
// 	log.Fatal().Err(err).Msg("failed to create camunda client")
// }
// cli.AddWorkers(workers...)
// cli.Start()
// defer cli.Close()
// cli.StartProcessInstance(ctx, "processId", 1, map[string]any{"name": "John"})

type CamundaClient struct {
	Client         zbc.Client
	camundaWorkers []CamundaWorker // 注册的job worker
	jobWorkers     []JobWorker     // 启动的job worker
}

func New(gateway string, workers ...CamundaWorker) (*CamundaClient, error) {
	config := zbc.ClientConfig{
		UsePlaintextConnection: true,
		GatewayAddress:         gateway,
		DialOpts:               []grpc.DialOption{grpc.WithBlock(), grpc.WithTimeout(5 * time.Second)},
	}
	client, err := zbc.NewClient(&config)
	if err != nil {
		return nil, fmt.Errorf("connect to camunda server error: %w", err)
	}

	return &CamundaClient{
		Client:         client,
		camundaWorkers: workers,
	}, nil
}

func (cli *CamundaClient) DeployProcess(ctx context.Context, name string, processDefinition []byte) (*pb.ProcessMetadata, error) {
	command := cli.Client.NewDeployResourceCommand().AddResource(processDefinition, name)

	resource, err := command.Send(ctx)
	if err != nil {
		return nil, err
	}

	if len(resource.GetDeployments()) == 0 {
		return nil, fmt.Errorf("failed to deploy process: %v", name)
	}

	demplyment := resource.GetDeployments()[0]
	process := demplyment.GetProcess()
	if process == nil {
		return nil, fmt.Errorf("failed to deploy process: %v, the deployment was successfule, but no process was returned", name)
	}

	return process, nil
}

func (cli *CamundaClient) StartProcessInstance(ctx context.Context, processId string, version int32, vars map[string]any) (*pb.CreateProcessInstanceResponse, error) {
	var step3 commands.CreateInstanceCommandStep3

	step2 := cli.Client.NewCreateInstanceCommand().BPMNProcessId(processId)

	if version < 1 {
		step3 = step2.LatestVersion()
	} else {
		step3 = step2.Version(version)
	}

	command, err := step3.VariablesFromMap(vars)
	if err != nil {
		return nil, err
	}

	process, err := command.Send(ctx)
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (cli *CamundaClient) AddWorkers(workers ...CamundaWorker) {
	cli.camundaWorkers = append(cli.camundaWorkers, workers...)
}

func (cli *CamundaClient) Start() error {
	var jobWorkers []JobWorker
	for _, w := range cli.camundaWorkers {
		jobWorker := cli.startWorker(w.JobType, w.WorkerName, w.JobHandler, WithConcurrency(w.Concurrency), WithMaxJobsActive(w.MaxJobActives), WithTimeout(w.Timeout), WithPoolInterval(w.PollInterval))

		jobWorkers = append(jobWorkers, JobWorker{
			Name:   w.WorkerName,
			Worker: jobWorker,
		})
	}

	cli.jobWorkers = jobWorkers

	return nil
}

func (cli *CamundaClient) startWorker(jobType, workerName string, jobHandler worker.JobHandler, opts ...Option) worker.JobWorker {
	so := defaultStartOptions()
	for _, o := range opts {
		o(&so)
	}

	log.Info().Msgf("start job worker: %s", workerName)

	worker := cli.Client.NewJobWorker().JobType(jobType).Handler(jobHandler).Concurrency(so.concurrency).MaxJobsActive(so.maxJobActives).RequestTimeout(so.timeout).PollInterval(so.pollInternal).Name(workerName).Open()
	return worker
}

func (cli *CamundaClient) Close() error {

	for _, w := range cli.jobWorkers {
		log.Info().Msgf("close job worker: %s", w.Name)
		w.Worker.Close()
	}

	return cli.Client.Close()
}
