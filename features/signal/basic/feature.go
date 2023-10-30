package basic

import (
	"context"

	"github.com/temporalio/features/harness/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	MySignalName = "mySignal"
	SignalData   = "signal-data"
)

var Feature = harness.Feature{
	Workflows:       MyWorkflow,
	ExpectRunResult: SignalData,
	Execute: func(ctx context.Context, runner *harness.Runner) (client.WorkflowRun, error) {
		run, err := runner.ExecuteDefault(ctx)
		if err != nil {
			return nil, err
		}
		err = runner.Client.SignalWorkflow(
			context.Background(),
			run.GetID(),
			run.GetRunID(),
			MySignalName,
			SignalData,
		)
		if err != nil {
			return nil, err
		}
		return run, nil
	},
}

func MyWorkflow(ctx workflow.Context) (string, error) {
	wfResult := ""
	signalCh := workflow.GetSignalChannel(ctx, MySignalName)
	signalCh.Receive(ctx, &wfResult)
	return wfResult, nil
}
