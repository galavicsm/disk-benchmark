import type { app } from "../../../wailsjs/go/models";
import { serializeBenchmarkRequest } from "../../lib/request";

export type RunState =
  | "idle"
  | "validating"
  | "awaiting-confirmation"
  | "running"
  | "cancelling"
  | "completed"
  | "failed"
  | "cancelled";

export interface StateEvent {
  runId: string;
  state: RunState;
}

export interface ProgressEvent {
  runId: string;
  phase: string;
  status: string;
  warmup: number;
  warmupsTotal: number;
  iteration: number;
  iterationsTotal: number;
  workload?: string;
  workloadNumber: number;
  workloadsTotal: number;
  indeterminate: boolean;
}

export interface CompletedEvent {
  runId: string;
  report: app.Report;
}

export interface FailedEvent {
  runId: string;
  error: app.ServiceError;
}

export interface CancelledEvent {
  runId: string;
}

export interface ExecutionState {
  status: RunState;
  runId: string | null;
  acceptingEvents: boolean;
  retiredRunIds: readonly string[];
  submittedRequest: Readonly<app.BenchmarkRequest> | null;
  report: app.Report | null;
  progress: ProgressEvent | null;
  error: app.ServiceError | null;
  confirmation: app.WriteImpactAssessment | null;
  startedAt: number;
}

export type ExecutionAction =
  | {
      type: "validation-started";
      request: app.BenchmarkRequest;
      startedAt: number;
    }
  | { type: "validation-failed"; error: app.ServiceError }
  | {
      type: "confirmation-required";
      request: app.BenchmarkRequest;
      assessment: app.WriteImpactAssessment;
    }
  | { type: "confirmation-dismissed" }
  | {
      type: "run-requested";
      request: app.BenchmarkRequest;
      startedAt: number;
    }
  | { type: "state-event"; event: StateEvent }
  | { type: "progress-event"; event: ProgressEvent }
  | { type: "completed-event"; event: CompletedEvent }
  | { type: "failed-event"; event: FailedEvent }
  | { type: "cancelled-event"; event: CancelledEvent }
  | { type: "rpc-resolved"; result: app.BenchmarkResult }
  | { type: "rpc-rejected"; error: app.ServiceError }
  | { type: "cancel-resolved"; runId: string; result: app.CancelResult }
  | { type: "cancel-rejected"; runId: string; error: app.ServiceError }
  | { type: "reset" };

export const initialExecutionState: ExecutionState = {
  status: "idle",
  runId: null,
  acceptingEvents: false,
  retiredRunIds: [],
  submittedRequest: null,
  report: null,
  progress: null,
  error: null,
  confirmation: null,
  startedAt: 0,
};

export function executionReducer(
  state: ExecutionState,
  action: ExecutionAction,
): ExecutionState {
  switch (action.type) {
    case "validation-started":
      return {
        ...initialExecutionState,
        status: "validating",
        submittedRequest: snapshotRequest(action.request),
        retiredRunIds: state.retiredRunIds,
        startedAt: action.startedAt,
      };
    case "validation-failed":
      return {
        ...state,
        status: "failed",
        acceptingEvents: false,
        error: action.error,
      };
    case "confirmation-required":
      return {
        ...state,
        status: "awaiting-confirmation",
        submittedRequest: snapshotRequest(action.request),
        confirmation: action.assessment,
        error: null,
      };
    case "confirmation-dismissed":
      return {
        ...state,
        status: "idle",
        confirmation: null,
        error: null,
      };
    case "run-requested":
      return {
        ...state,
        status: "validating",
        runId: null,
        acceptingEvents: true,
        submittedRequest: snapshotRequest(action.request),
        report: null,
        progress: null,
        error: null,
        confirmation: null,
        startedAt: action.startedAt,
      };
    case "state-event":
      return applyStateEvent(state, action.event);
    case "progress-event":
      return acceptsRunEvent(state, action.event.runId)
        ? {
            ...state,
            runId: state.runId ?? action.event.runId,
            progress: action.event,
          }
        : state;
    case "completed-event":
      return acceptsRunEvent(state, action.event.runId)
        ? finishRun(state, action.event.runId, "completed", {
            report: action.event.report,
          })
        : state;
    case "failed-event":
      return acceptsRunEvent(state, action.event.runId)
        ? finishRun(state, action.event.runId, "failed", {
            error: action.event.error,
          })
        : state;
    case "cancelled-event":
      return acceptsRunEvent(state, action.event.runId)
        ? finishRun(state, action.event.runId, "cancelled")
        : state;
    case "rpc-resolved":
      return applyRPCResult(state, action.result);
    case "rpc-rejected":
      return state.acceptingEvents
        ? {
            ...state,
            status: "failed",
            acceptingEvents: false,
            error: action.error,
          }
        : state;
    case "cancel-resolved":
      if (
        state.runId !== action.runId ||
        state.retiredRunIds.includes(action.runId)
      ) {
        return state;
      }
      if (action.result.error !== undefined) {
        return { ...state, error: action.result.error };
      }
      if (
        action.result.runId !== undefined &&
        action.result.runId !== action.runId
      ) {
        return state;
      }
      return {
        ...state,
        status: asRunState(action.result.state),
        error: null,
      };
    case "cancel-rejected":
      return state.runId === action.runId &&
        !state.retiredRunIds.includes(action.runId)
        ? { ...state, error: action.error }
        : state;
    case "reset":
      return {
        ...initialExecutionState,
        retiredRunIds: state.retiredRunIds,
      };
  }
}

export function statusForState(state: RunState): string {
  switch (state) {
    case "validating":
      return "Validating configuration...";
    case "running":
      return "Benchmark is running. Configuration is locked.";
    case "cancelling":
      return "Cancellation requested. Waiting for the active I/O call.";
    case "cancelled":
      return "Benchmark cancelled.";
    case "completed":
      return "Benchmark completed.";
    case "failed":
      return "Benchmark failed.";
    case "awaiting-confirmation":
      return "Review the write-impact warning.";
    case "idle":
      return "Ready to configure a benchmark.";
  }
}

function applyStateEvent(
  state: ExecutionState,
  event: StateEvent,
): ExecutionState {
  if (!acceptsRunEvent(state, event.runId)) {
    return state;
  }
  if (event.state === "idle" && state.status === "cancelled") {
    return {
      ...state,
      acceptingEvents: false,
      retiredRunIds: retire(state.retiredRunIds, event.runId),
    };
  }
  return {
    ...state,
    status: event.state,
    runId: state.runId ?? event.runId,
  };
}

function applyRPCResult(
  state: ExecutionState,
  result: app.BenchmarkResult,
): ExecutionState {
  const runId = result.runId ?? state.runId;
  if (
    runId === null ||
    state.retiredRunIds.includes(runId) ||
    (state.runId !== null && state.runId !== runId)
  ) {
    return state;
  }
  const status = asRunState(result.state);
  if (status === "completed" && result.report !== undefined) {
    return finishRun(state, runId, status, { report: result.report });
  }
  if (status === "failed") {
    return finishRun(state, runId, status, {
      error:
        result.error ??
        serviceError("execution", "Benchmark failed without an error detail."),
    });
  }
  if (status === "cancelled") {
    return finishRun(state, runId, status);
  }
  return {
    ...state,
    status,
    runId,
    error: result.error ?? null,
  };
}

function acceptsRunEvent(state: ExecutionState, runId: string): boolean {
  return (
    state.acceptingEvents &&
    runId !== "" &&
    !state.retiredRunIds.includes(runId) &&
    (state.runId === null || state.runId === runId)
  );
}

function finishRun(
  state: ExecutionState,
  runId: string,
  status: "completed" | "failed" | "cancelled",
  values: Pick<ExecutionState, "report"> | Pick<ExecutionState, "error"> | object = {},
): ExecutionState {
  return {
    ...state,
    ...values,
    status,
    runId,
    acceptingEvents: false,
    confirmation: null,
    retiredRunIds: retire(state.retiredRunIds, runId),
  };
}

function retire(runIds: readonly string[], runId: string): readonly string[] {
  return runIds.includes(runId) ? runIds : [...runIds, runId];
}

function snapshotRequest(
  request: app.BenchmarkRequest,
): Readonly<app.BenchmarkRequest> {
  return Object.freeze(serializeBenchmarkRequest(request));
}

function asRunState(value: string): RunState {
  switch (value) {
    case "idle":
    case "validating":
    case "awaiting-confirmation":
    case "running":
    case "cancelling":
    case "completed":
    case "failed":
    case "cancelled":
      return value;
    default:
      return "failed";
  }
}

export function serviceError(code: string, message: string): app.ServiceError {
  return { code, message };
}
