package retry

import (
	"context"
	"fmt"

	"uasl-reservation/internal/pkg/logger"
)

type Step struct {
	Name     string
	Rollback func(context.Context) error
	Metadata map[string]interface{}
}

type Orchestrator struct {
	successfulSteps []Step
	retryConfig     Config
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		successfulSteps: make([]Step, 0),
		retryConfig:     DefaultConfig(),
	}
}

func NewOrchestratorWithRetry(retryConfig Config) *Orchestrator {
	return &Orchestrator{
		successfulSteps: make([]Step, 0),
		retryConfig:     retryConfig,
	}
}

func (o *Orchestrator) RecordSuccess(step Step) {
	o.successfulSteps = append(o.successfulSteps, step)
	logger.LogInfo("saga: step recorded successfully",
		"step_name", step.Name,
		"total_steps", len(o.successfulSteps),
	)
}

func (o *Orchestrator) Rollback(ctx context.Context) []CompensationFailure {
	logger.LogInfo("saga: starting rollback",
		"total_steps", len(o.successfulSteps),
	)

	var failures []CompensationFailure
	total := len(o.successfulSteps)

	for i := total - 1; i >= 0; i-- {
		step := o.successfulSteps[i]

		logger.LogInfo("saga: executing compensation",
			"step_name", step.Name,
			"step_index", i,
			"total_steps", total,
		)

		err := WithBackoff(ctx, step.Rollback, o.retryConfig)
		if err != nil {
			failure := CompensationFailure{
				StepName:  step.Name,
				StepIndex: i,
				Metadata:  step.Metadata,
				Error:     err,
			}
			failures = append(failures, failure)

			logger.LogError("CRITICAL: saga compensation failed - manual intervention required",
				"step_name", step.Name,
				"step_index", i,
				"total_steps", total,
				"error", err,
				"metadata", step.Metadata,
				"severity", "CRITICAL",
				"requires_manual_intervention", true,
			)
		} else {
			logger.LogInfo("saga: compensation succeeded",
				"step_name", step.Name,
				"step_index", i,
			)
		}
	}

	if len(failures) > 0 {
		logger.LogError("saga: rollback completed with failures",
			"total_failures", len(failures),
			"total_steps", total,
		)
	} else {
		logger.LogInfo("saga: rollback completed successfully",
			"total_steps", total,
		)
	}

	return failures
}

func (o *Orchestrator) GetSuccessfulSteps() int {
	return len(o.successfulSteps)
}

type CompensationFailure struct {
	StepName  string
	StepIndex int
	Metadata  map[string]interface{}
	Error     error
}

func (cf CompensationFailure) String() string {
	return fmt.Sprintf("step=%s index=%d error=%v metadata=%v",
		cf.StepName, cf.StepIndex, cf.Error, cf.Metadata)
}
