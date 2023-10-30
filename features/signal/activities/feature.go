package activites

import (
	"context"
	"time"

	"github.com/temporalio/features/harness/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	MySignalName   = "mySignal"
	ActivityCount  = 5
	ActivityResult = 6
)

func MyActivity(ctx context.Context) (int, error) {
	return ActivityResult, nil
}

var Feature = harness.Feature{
	Workflows:       MyWorkflow,
	Activities:      MyActivity,
	ExpectRunResult: ActivityResult * ActivityCount,
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
			nil,
		)
		if err != nil {
			return nil, err
		}
		return run, nil
	},
}

func MyWorkflow(ctx workflow.Context) (int, error) {
	total := 0
	signalCh := workflow.GetSignalChannel(ctx, MySignalName)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	signalHandler := func() {
		futures := make([]workflow.Future, ActivityCount)
		for i := 0; i < ActivityCount; i++ {
			futures[i] = workflow.ExecuteActivity(ctx, MyActivity)
		}
		for _, future := range futures {
			var singleResult int
			future.Get(ctx, &singleResult)
			total += singleResult
		}
	}

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
		signalHandler()
	})

	selector.Select(ctx)

	return total, nil
}
