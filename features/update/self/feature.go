package self

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/features/features/update/updateutil"
	"go.temporal.io/features/harness/go/harness"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	updateName                = "update!"
	updateNotEnabledErrorType = "PermissionDenied"
)

type ConnMaterial struct {
	HostPort       string
	Namespace      string
	Identity       string
	ClientCertPath string
	ClientKeyPath  string
}

var Feature = harness.Feature{
	Workflows:  SelfUpdateWorkflow,
	Activities: SelfUpdateActivity,
	Execute: func(ctx context.Context, runner *harness.Runner) (client.WorkflowRun, error) {
		if reason := updateutil.CheckServerSupportsUpdate(ctx, runner.Client); reason != "" {
			return nil, runner.Skip(reason)
		}
		opts := runner.Feature.StartWorkflowOptions
		if opts.TaskQueue == "" {
			opts.TaskQueue = runner.TaskQueue
		}
		if opts.WorkflowExecutionTimeout == 0 {
			opts.WorkflowExecutionTimeout = 1 * time.Minute
		}
		return runner.Client.ExecuteWorkflow(ctx, opts, SelfUpdateWorkflow, ConnMaterial{
			HostPort:       runner.Feature.ClientOptions.HostPort,
			Namespace:      runner.Feature.ClientOptions.Namespace,
			Identity:       runner.Feature.ClientOptions.Identity,
			ClientCertPath: runner.ClientCertPath,
			ClientKeyPath:  runner.ClientKeyPath,
		})
	},
}

func SelfUpdateWorkflow(ctx workflow.Context, cm ConnMaterial) (string, error) {
	const expectedState = "called"
	state := "not " + expectedState
	workflow.SetUpdateHandler(ctx, updateName, func() error {
		state = expectedState
		return nil
	})
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				NonRetryableErrorTypes: []string{updateNotEnabledErrorType},
			},
		}),
		SelfUpdateActivity,
		cm,
	).Get(ctx, nil)
	if err != nil {
		return "", err
	}
	if state != expectedState {
		return "", fmt.Errorf("expected state == %q but found %q", expectedState, state)
	}
	return state, nil
}

func SelfUpdateActivity(ctx context.Context, cm ConnMaterial) error {
	tlsCfg, err := harness.LoadTLSConfig(cm.ClientCertPath, cm.ClientKeyPath)
	if err != nil {
		return err
	}
	client, err := client.Dial(
		client.Options{
			HostPort:          cm.HostPort,
			Namespace:         cm.Namespace,
			Logger:            activity.GetLogger(ctx),
			MetricsHandler:    activity.GetMetricsHandler(ctx),
			Identity:          cm.Identity,
			ConnectionOptions: client.ConnectionOptions{TLS: tlsCfg},
		},
	)
	if err != nil {
		return err
	}
	wfe := activity.GetInfo(ctx).WorkflowExecution
	updateHandle, err := client.UpdateWorkflow(ctx, wfe.ID, wfe.RunID, updateName)
	if err != nil {
		return err
	}
	return updateHandle.Get(ctx, nil)
}
