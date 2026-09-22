import { describe, expect, it } from "vitest";
import { benchmarkReport, benchmarkRequest } from "../../test/fixtures";
import {
  executionReducer,
  initialExecutionState,
  serviceError,
} from "./state";

describe("executionReducer", () => {
  it("moves through validation, confirmation, running, and completion", () => {
    const request = benchmarkRequest();
    const validating = executionReducer(initialExecutionState, {
      type: "validation-started",
      request,
      startedAt: 100,
    });
    request.size = "2GiB";

    expect(validating.status).toBe("validating");
    expect(validating.submittedRequest?.size).toBe("1GiB");

    const awaiting = executionReducer(validating, {
      type: "confirmation-required",
      request: benchmarkRequest({ writeImpactConfirmed: true }),
      assessment: {
        requiresConfirmation: true,
        reasons: ["large write"],
        largeSizeThresholdBytes: "1",
      },
    });
    const requested = executionReducer(awaiting, {
      type: "run-requested",
      request: { ...awaiting.submittedRequest! },
      startedAt: 200,
    });
    const running = executionReducer(requested, {
      type: "state-event",
      event: { runId: "7", state: "running" },
    });
    const completed = executionReducer(running, {
      type: "completed-event",
      event: { runId: "7", report: benchmarkReport() },
    });

    expect(completed.status).toBe("completed");
    expect(completed.report?.summaries[0].averageMibPerSecond).toBe(512);
    expect(completed.retiredRunIds).toContain("7");
  });

  it("rejects events from another or retired run", () => {
    const requested = executionReducer(initialExecutionState, {
      type: "run-requested",
      request: benchmarkRequest(),
      startedAt: 100,
    });
    const running = executionReducer(requested, {
      type: "state-event",
      event: { runId: "current", state: "running" },
    });
    const staleProgress = executionReducer(running, {
      type: "progress-event",
      event: {
        runId: "stale",
        phase: "workload",
        status: "started",
        warmup: 0,
        warmupsTotal: 0,
        iteration: 1,
        iterationsTotal: 1,
        workload: "Wrong workload",
        workloadNumber: 1,
        workloadsTotal: 1,
        indeterminate: false,
      },
    });
    const cancelled = executionReducer(running, {
      type: "cancelled-event",
      event: { runId: "current" },
    });
    const lateCompletion = executionReducer(cancelled, {
      type: "completed-event",
      event: { runId: "current", report: benchmarkReport() },
    });

    expect(staleProgress).toBe(running);
    expect(lateCompletion).toBe(cancelled);
    expect(lateCompletion.status).toBe("cancelled");
  });

  it("retains the submitted request and structured failure", () => {
    const requested = executionReducer(initialExecutionState, {
      type: "run-requested",
      request: benchmarkRequest({ directory: "D:\\failed" }),
      startedAt: 100,
    });
    const failed = executionReducer(requested, {
      type: "failed-event",
      event: {
        runId: "9",
        error: serviceError("execution", "disk full"),
      },
    });

    expect(failed.status).toBe("failed");
    expect(failed.error?.message).toBe("disk full");
    expect(failed.submittedRequest?.directory).toBe("D:\\failed");
  });

  it("ignores delayed cancellation and RPC failures after a terminal event", () => {
    const requested = executionReducer(initialExecutionState, {
      type: "run-requested",
      request: benchmarkRequest(),
      startedAt: 100,
    });
    const running = executionReducer(requested, {
      type: "state-event",
      event: { runId: "10", state: "running" },
    });
    const cancelling = executionReducer(running, {
      type: "cancel-resolved",
      runId: "10",
      result: { runId: "10", state: "cancelling" },
    });
    const cancelledState = executionReducer(cancelling, {
      type: "state-event",
      event: { runId: "10", state: "cancelled" },
    });
    const cancelled = executionReducer(cancelledState, {
      type: "cancelled-event",
      event: { runId: "10" },
    });
    const delayedCancel = executionReducer(cancelled, {
      type: "cancel-resolved",
      runId: "10",
      result: { runId: "10", state: "cancelling" },
    });
    const delayedFailure = executionReducer(cancelled, {
      type: "rpc-rejected",
      error: serviceError("execution", "transport closed"),
    });

    expect(cancelling.status).toBe("cancelling");
    expect(cancelled.status).toBe("cancelled");
    expect(delayedCancel).toBe(cancelled);
    expect(delayedFailure).toBe(cancelled);
  });
});
