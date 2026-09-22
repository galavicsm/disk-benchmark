import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { app } from "../../../wailsjs/go/models";
import {
  CancelBenchmark,
  InspectDirectory,
  SelectDirectory,
  SelectSuiteFile,
  StartBenchmark,
  ValidateBenchmark,
} from "../../../wailsjs/go/app/Service";
import { formatBytes } from "../../lib/format";
import { serializeBenchmarkRequest } from "../../lib/request";
import {
  type FieldErrors,
  isBenchmarkRequestField,
  validateRequestLocally,
} from "../../lib/validation";
import { ExecutionView } from "../execution/ExecutionView";
import { useExecution } from "../execution/ExecutionContext";
import {
  type RunState,
  serviceError,
  statusForState,
} from "../execution/state";
import { ResultsView } from "../results/ResultsView";
import { ConfirmationDialog } from "./ConfirmationDialog";

interface BenchmarkFormProps {
  initialRequest: app.BenchmarkRequest;
}

interface SelectedSuite {
  path: string;
  summary: app.SuiteSummary;
}

const activeStates = new Set<RunState>(["validating", "running", "cancelling"]);
const randomWorkloads = new Set(["random", "mixed", "all"]);

function FieldError({ message }: { message?: string }) {
  return message ? <span className="field-error">{message}</span> : null;
}

function focusFirstError(errors: FieldErrors) {
  const field = Object.entries(errors).find(([, message]) => message !== undefined)?.[0];
  if (field === undefined) {
    return;
  }

  window.requestAnimationFrame(() => {
    document.getElementById(field)?.focus();
  });
}

export function BenchmarkForm({ initialRequest }: BenchmarkFormProps) {
  const { state: execution, dispatch } = useExecution();
  const [request, setRequest] = useState(initialRequest);
  const [sourceMode, setSourceMode] = useState<"built-in" | "suite">("built-in");
  const [lastBuiltIn, setLastBuiltIn] = useState(initialRequest.workload);
  const [selectedSuite, setSelectedSuite] = useState<SelectedSuite | null>(null);
  const [inspection, setInspection] = useState<app.DirectoryInspection | null>(null);
  const [inspectionSource, setInspectionSource] = useState("");
  const [inspectionBusy, setInspectionBusy] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [banner, setBanner] = useState("");
  const inspectionSequence = useRef(0);
  const startButton = useRef<HTMLButtonElement>(null);

  const locked =
    activeStates.has(execution.status) ||
    execution.status === "awaiting-confirmation";
  const effectiveInspection =
    inspectionSource === request.directory ? inspection : null;
  const showRandomOptions =
    sourceMode === "built-in" && randomWorkloads.has(request.workload);

  const inspect = useCallback(async (directory: string) => {
    if (directory.trim() === "") {
      setInspection(null);
      setInspectionSource("");
      return;
    }
    const sequence = ++inspectionSequence.current;
    setInspectionBusy(true);
    const result = await InspectDirectory(directory);
    if (sequence !== inspectionSequence.current) {
      return;
    }
    setInspectionBusy(false);
    setInspectionSource(directory);
    if (result.error !== undefined) {
      setInspection(null);
      setFieldErrors((current) => ({ ...current, directory: result.error?.message }));
      return;
    }
    setInspection(result.inspection ?? null);
    setFieldErrors((current) => ({ ...current, directory: undefined }));
  }, []);

  useEffect(() => {
    void inspect(initialRequest.directory);
  }, [initialRequest.directory, inspect]);

  const updateRequest = <Key extends keyof app.BenchmarkRequest>(
    key: Key,
    value: app.BenchmarkRequest[Key],
  ) => {
    setRequest((current) => ({
      ...current,
      [key]: value,
      writeImpactConfirmed: false,
    }));
    setFieldErrors((current) => ({ ...current, [key]: undefined }));
  };

  const chooseDirectory = async () => {
    setBanner("");
    const result = await SelectDirectory();
    if (result.error !== undefined) {
      setBanner(result.error.message);
      return;
    }
    if (!result.cancelled && result.path !== undefined) {
      updateRequest("directory", result.path);
      await inspect(result.path);
    }
  };

  const chooseSuite = async () => {
    setBanner("");
    const result = await SelectSuiteFile();
    if (result.error !== undefined) {
      setFieldErrors((current) => ({
        ...current,
        suiteFile: result.error?.message,
      }));
      return;
    }
    if (!result.cancelled && result.path !== undefined && result.suite !== undefined) {
      const suite = { path: result.path, summary: result.suite };
      setSelectedSuite(suite);
      setRequest((current) => ({
        ...current,
        workload: "",
        suiteFile: suite.path,
        writeImpactConfirmed: false,
      }));
      setFieldErrors((current) => ({ ...current, suiteFile: undefined }));
    }
  };

  const changeSourceMode = (mode: "built-in" | "suite") => {
    setSourceMode(mode);
    setRequest((current) => {
      if (mode === "suite") {
        if (current.workload !== "") {
          setLastBuiltIn(current.workload);
        }
        return {
          ...current,
          workload: "",
          suiteFile: selectedSuite?.path ?? "",
          writeImpactConfirmed: false,
        };
      }
      return {
        ...current,
        workload: lastBuiltIn || "sequential",
        suiteFile: "",
        writeImpactConfirmed: false,
      };
    });
    setFieldErrors({});
  };

  const execute = async (submitted: app.BenchmarkRequest) => {
    setBanner("");
    const serialized = serializeBenchmarkRequest(submitted);
    dispatch({
      type: "run-requested",
      request: serialized,
      startedAt: Date.now(),
    });
    try {
      const result = await StartBenchmark(serialized);
      dispatch({ type: "rpc-resolved", result });
    } catch (error: unknown) {
      dispatch({
        type: "rpc-rejected",
        error: serviceError(
          "execution",
          error instanceof Error ? error.message : String(error),
        ),
      });
    }
  };

  const submit = async () => {
    setBanner("");
    const localErrors = validateRequestLocally(request, effectiveInspection);
    setFieldErrors(localErrors);
    if (Object.keys(localErrors).length > 0) {
      focusFirstError(localErrors);
      return;
    }

    const submitted = serializeBenchmarkRequest({
      ...request,
      writeImpactConfirmed: false,
    });
    dispatch({
      type: "validation-started",
      request: submitted,
      startedAt: Date.now(),
    });
    let validation: app.ValidationResult;
    try {
      validation = await ValidateBenchmark(submitted);
    } catch (error: unknown) {
      dispatch({
        type: "validation-failed",
        error: serviceError(
          "validation",
          error instanceof Error ? error.message : String(error),
        ),
      });
      return;
    }
    if (!validation.valid) {
      const errors: FieldErrors = {};
      let generalError: app.ServiceError | undefined;
      for (const error of validation.errors) {
        if (error.field !== undefined && isBenchmarkRequestField(error.field)) {
          errors[error.field] = error.message;
        } else {
          generalError = error;
        }
      }
      setFieldErrors(errors);
      focusFirstError(errors);
      dispatch({
        type: "validation-failed",
        error:
          generalError ??
          validation.errors[0] ??
          serviceError("validation", "Go validation rejected this configuration."),
      });
      return;
    }
    if (validation.writeImpact.requiresConfirmation) {
      dispatch({
        type: "confirmation-required",
        request: serializeBenchmarkRequest({
          ...submitted,
          writeImpactConfirmed: true,
        }),
        assessment: validation.writeImpact,
      });
      return;
    }
    await execute(submitted);
  };

  const cancel = async () => {
    if (execution.runId === null) {
      setBanner("The benchmark run ID is not available yet.");
      return;
    }
    const runId = execution.runId;
    try {
      const result = await CancelBenchmark(runId);
      dispatch({ type: "cancel-resolved", runId, result });
    } catch (error: unknown) {
      dispatch({
        type: "cancel-rejected",
        runId,
        error: serviceError(
          "execution",
          error instanceof Error ? error.message : String(error),
        ),
      });
    }
  };

  const workloadWarning = useMemo(() => {
    if (request.duration.trim() !== "" && request.duration.trim() !== "0s") {
      return "Duration-based workloads may process and write the working set repeatedly.";
    }
    if (request.workload === "fsync" || request.workload === "all") {
      return "Fsync-per-block workloads issue a durable write for every block.";
    }
    return "";
  }, [request.duration, request.workload]);

  const confirmRun = () => {
    if (execution.submittedRequest === null) {
      dispatch({
        type: "validation-failed",
        error: serviceError(
          "validation",
          "The submitted benchmark configuration is no longer available.",
        ),
      });
      return;
    }
    void execute(serializeBenchmarkRequest(execution.submittedRequest));
  };

  if (activeStates.has(execution.status)) {
    return (
      <ExecutionView
        state={execution.status}
        progress={execution.progress}
        startedAt={execution.startedAt}
        canCancel={execution.runId !== null}
        errorMessage={execution.error?.message ?? banner}
        onCancel={() => void cancel()}
      />
    );
  }

  if (
    execution.status === "completed" &&
    execution.report !== null &&
    execution.runId !== null
  ) {
    return (
      <ResultsView
        runId={execution.runId}
        report={execution.report}
        onNewBenchmark={() => {
          dispatch({ type: "reset" });
          setBanner("");
        }}
      />
    );
  }

  return (
    <>
      <form
        className="configuration-form"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        {(execution.status === "failed" || execution.status === "cancelled") && (
          <section
            className={`panel run-outcome ${execution.status}`}
            aria-labelledby="run-outcome-heading"
          >
            <div>
              <p className="eyebrow">
                {execution.status === "cancelled" ? "Cancelled" : "Execution error"}
              </p>
              <h2 id="run-outcome-heading">
                {execution.status === "cancelled"
                  ? "Benchmark was cancelled"
                  : "Benchmark did not complete"}
              </h2>
              <p>
                Your submitted configuration is retained below. Review it and start
                again when ready.
              </p>
            </div>
            <button
              type="button"
              className="button secondary"
              onClick={() => {
                dispatch({ type: "reset" });
                setBanner("");
                startButton.current?.focus();
              }}
            >
              Review configuration
            </button>
          </section>
        )}

        <fieldset disabled={locked} className="form-fieldset">
          <legend className="sr-only">Benchmark configuration</legend>

          <section className="panel primary-panel" aria-labelledby="basic-heading">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Configuration</p>
                <h2 id="basic-heading">Benchmark setup</h2>
              </div>
              <span className="step-badge">Required</span>
            </div>

            <div className="field full-width">
              <label htmlFor="directory">Target directory</label>
              <div className="input-action">
                <input
                  id="directory"
                  value={request.directory}
                  aria-invalid={fieldErrors.directory !== undefined}
                  aria-describedby="directory-help directory-error"
                  onChange={(event) => updateRequest("directory", event.target.value)}
                  onBlur={() => void inspect(request.directory)}
                />
                <button type="button" className="button secondary" onClick={() => void chooseDirectory()}>
                  Browse
                </button>
              </div>
              <span id="directory-help" className="field-help">
                Temporary benchmark files are created on this volume.
              </span>
              <span id="directory-error"><FieldError message={fieldErrors.directory} /></span>
            </div>

            <div className="form-grid">
              <div className="field">
                <label htmlFor="size">Working-set size</label>
                <input
                  id="size"
                  value={request.size}
                  aria-invalid={fieldErrors.size !== undefined}
                  aria-describedby="size-help size-error"
                  onChange={(event) => updateRequest("size", event.target.value)}
                />
                <span id="size-help" className="field-help">
                  {inspectionBusy && "Inspecting available space..."}
                  {!inspectionBusy &&
                    effectiveInspection !== null &&
                    `${formatBytes(effectiveInspection.availableBytes)} available on ${effectiveInspection.volume || "this volume"}.`}
                  {!inspectionBusy && effectiveInspection === null && "For example: 1GiB or 512MiB."}
                </span>
                <span id="size-error"><FieldError message={fieldErrors.size} /></span>
              </div>
              <div className="field">
                <label htmlFor="block-size">Block size</label>
                <input
                  id="block-size"
                  value={request.blockSize}
                  aria-invalid={fieldErrors.blockSize !== undefined}
                  aria-describedby="block-help block-error"
                  onChange={(event) => updateRequest("blockSize", event.target.value)}
                />
                <span id="block-help" className="field-help">For example: 1MiB or 4KiB.</span>
                <span id="block-error"><FieldError message={fieldErrors.blockSize} /></span>
              </div>
            </div>

            <fieldset className="source-selector">
              <legend>Workload source</legend>
              <label className="choice-card">
                <input
                  type="radio"
                  name="source-mode"
                  checked={sourceMode === "built-in"}
                  onChange={() => changeSourceMode("built-in")}
                />
                <span>
                  <strong>Built-in preset</strong>
                  <small>Use a standard workload included with DiskBenchmark.</small>
                </span>
              </label>
              <label className="choice-card">
                <input
                  type="radio"
                  name="source-mode"
                  checked={sourceMode === "suite"}
                  onChange={() => changeSourceMode("suite")}
                />
                <span>
                  <strong>Suite file</strong>
                  <small>Run an ordered, versioned JSON workload suite.</small>
                </span>
              </label>
            </fieldset>

            {sourceMode === "built-in" ? (
              <div className="field full-width">
                <label htmlFor="workload">Workload preset</label>
                <select
                  id="workload"
                  value={request.workload}
                  onChange={(event) => {
                    setLastBuiltIn(event.target.value);
                    updateRequest("workload", event.target.value);
                  }}
                >
                  <option value="sequential">Sequential write and read</option>
                  <option value="random">Random mixed I/O</option>
                  <option value="mixed">Random mixed I/O (alias)</option>
                  <option value="overwrite">Sequential overwrite</option>
                  <option value="fsync">Fsync-per-block writes</option>
                  <option value="all">All built-in workloads</option>
                </select>
              </div>
            ) : (
              <div className="suite-picker">
                <div className="input-action">
                  <div className="selected-path" title={request.suiteFile}>
                    {request.suiteFile || "No suite selected"}
                  </div>
                  <button type="button" className="button secondary" onClick={() => void chooseSuite()}>
                    Choose suite
                  </button>
                </div>
                <FieldError message={fieldErrors.suiteFile} />
                {selectedSuite !== null && (
                  <details className="suite-summary">
                    <summary className="suite-summary-heading">
                      <strong>Suite version {selectedSuite.summary.version}</strong>
                      <span>{selectedSuite.summary.workloads.length} entries</span>
                    </summary>
                    <ol>
                      {selectedSuite.summary.workloads.map((workload, index) => (
                        <li key={`${workload.name}-${index}`}>
                          <div>
                            <strong>{workload.name}</strong>
                            <span>{workload.type}</span>
                          </div>
                          <div className="override-list">
                            {workload.duration !== undefined && (
                              <span>duration {workload.duration}</span>
                            )}
                            {workload.randomReadPercent !== undefined && (
                              <span>{workload.randomReadPercent}% reads</span>
                            )}
                            {workload.randomSeed !== undefined && (
                              <span>seed {workload.randomSeed}</span>
                            )}
                          </div>
                        </li>
                      ))}
                    </ol>
                  </details>
                )}
              </div>
            )}
          </section>

          <details className="panel advanced-panel">
            <summary>
              <span>
                <strong>Advanced options</strong>
                <small>Iterations, concurrency, I/O behavior, and verification</small>
              </span>
              <span className="summary-action" aria-hidden="true" />
            </summary>
            <div className="advanced-content">
              <div className="form-grid three-columns">
                <NumberField
                  id="iterations"
                  label="Measured iterations"
                  value={request.iterations}
                  min={1}
                  error={fieldErrors.iterations}
                  onChange={(value) => updateRequest("iterations", value)}
                />
                <NumberField
                  id="warmups"
                  label="Warm-up iterations"
                  value={request.warmupIterations}
                  min={0}
                  error={fieldErrors.warmupIterations}
                  onChange={(value) => updateRequest("warmupIterations", value)}
                />
                <div className="field">
                  <label htmlFor="duration">Duration per workload</label>
                  <input
                    id="duration"
                    value={request.duration}
                    placeholder="0s"
                    aria-invalid={fieldErrors.duration !== undefined}
                    onChange={(event) => updateRequest("duration", event.target.value)}
                  />
                  <span className="field-help">Zero processes the working set once.</span>
                  <FieldError message={fieldErrors.duration} />
                </div>
                <NumberField
                  id="workers"
                  label="Workers"
                  value={request.workers}
                  min={1}
                  error={fieldErrors.workers}
                  onChange={(value) => updateRequest("workers", value)}
                />
                <NumberField
                  id="queue-depth"
                  label="Queue depth per worker"
                  value={request.queueDepth}
                  min={1}
                  error={fieldErrors.queueDepth}
                  onChange={(value) => updateRequest("queueDepth", value)}
                />
              </div>

              {showRandomOptions && (
                <fieldset className="option-group">
                  <legend>Random workload</legend>
                  <div className="form-grid">
                    <NumberField
                      id="read-percent"
                      label="Read percentage"
                      value={request.randomReadPercent}
                      min={0}
                      max={100}
                      error={fieldErrors.randomReadPercent}
                      onChange={(value) => updateRequest("randomReadPercent", value)}
                    />
                    <div className="field">
                      <label htmlFor="random-seed">Random seed</label>
                      <input
                        id="random-seed"
                        value={request.randomSeed}
                        aria-invalid={fieldErrors.randomSeed !== undefined}
                        onChange={(event) => updateRequest("randomSeed", event.target.value)}
                      />
                      <FieldError message={fieldErrors.randomSeed} />
                    </div>
                  </div>
                </fieldset>
              )}

              <fieldset className="option-group">
                <legend>I/O behavior</legend>
                <div className="form-grid">
                  <div className="field">
                    <label htmlFor="io-mode">I/O mode</label>
                    <select
                      id="io-mode"
                      value={request.ioMode}
                      onChange={(event) => updateRequest("ioMode", event.target.value)}
                    >
                      <option value="buffered">Buffered</option>
                      <option value="direct">Direct / cache bypass</option>
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="cache-control">Read cache control</label>
                    <select
                      id="cache-control"
                      value={request.cacheControl}
                      onChange={(event) => updateRequest("cacheControl", event.target.value)}
                    >
                      <option value="off">Off</option>
                      <option value="attempt">Attempt</option>
                      <option value="require">Require</option>
                    </select>
                  </div>
                </div>
                {request.ioMode === "direct" && effectiveInspection !== null && (
                  <p className="inline-notice">
                    {effectiveInspection.directIo.capability === "available"
                      ? `Direct I/O alignment: ${effectiveInspection.directIo.alignmentBytes ?? "unknown"} bytes.`
                      : `Direct I/O could not be confirmed: ${effectiveInspection.directIo.detail ?? "unsupported"}.`}
                  </p>
                )}
              </fieldset>

              <fieldset className="option-group checkbox-grid">
                <legend>Data handling</legend>
                <CheckboxField
                  checked={request.syncWrites}
                  label="Sync writes"
                  description="Flush written data before a write measurement completes."
                  onChange={(checked) => updateRequest("syncWrites", checked)}
                />
                <CheckboxField
                  checked={request.verifyData}
                  label="Verify data"
                  description="Read the generated file and verify deterministic contents."
                  onChange={(checked) => updateRequest("verifyData", checked)}
                />
                <CheckboxField
                  checked={request.keepFile}
                  label="Keep generated file"
                  description="Retain files only after successful measured iterations."
                  onChange={(checked) => updateRequest("keepFile", checked)}
                />
              </fieldset>

              <details className="platform-notes">
                <summary>Platform-specific I/O behavior</summary>
                <ul>
                  <li>Windows does not provide safe buffered-cache eviction for this application.</li>
                  <li>Linux direct I/O requires filesystem support and aligned sizes.</li>
                  <li>macOS F_NOCACHE bypasses cache but is not equivalent to Windows or Linux direct I/O.</li>
                </ul>
              </details>
            </div>
          </details>
        </fieldset>

        {(banner !== "" || execution.error !== null) && (
          <div className="error-banner" role="alert">
            <strong>Unable to continue</strong>
            <span>{banner || execution.error?.message}</span>
          </div>
        )}

        {workloadWarning !== "" && <p className="write-note">{workloadWarning}</p>}

        <aside className="endurance-warning" aria-labelledby="endurance-title">
          <div className="warning-mark" aria-hidden="true">!</div>
          <div>
            <strong id="endurance-title">This benchmark writes to the selected drive</strong>
            <p>
              Duration-based, repeated, and fsync workloads may write substantially more
              than the configured working set and consume SSD endurance.
            </p>
          </div>
        </aside>

        <div className="action-bar">
          <div className="run-status" role="status" aria-live="polite">
            <span className={`status-dot ${activeStates.has(execution.status) ? "active" : ""}`} />
            <span>{statusForState(execution.status)}</span>
          </div>
          <div className="action-buttons">
            {activeStates.has(execution.status) && (
              <button
                type="button"
                className="button secondary"
                disabled={execution.status === "cancelling" || execution.runId === null}
                onClick={() => void cancel()}
              >
                {execution.status === "cancelling" ? "Cancelling..." : "Cancel"}
              </button>
            )}
            <button
              ref={startButton}
              type="submit"
              className="button primary"
              disabled={locked}
            >
              {execution.status === "validating" ? "Validating..." : "Start benchmark"}
            </button>
          </div>
        </div>
      </form>

      {execution.status === "awaiting-confirmation" &&
        execution.confirmation !== null &&
        execution.submittedRequest !== null && (
        <ConfirmationDialog
          assessment={execution.confirmation}
          returnFocus={startButton.current}
          onCancel={() => {
            dispatch({ type: "confirmation-dismissed" });
          }}
          onConfirm={confirmRun}
        />
      )}
    </>
  );
}

interface NumberFieldProps {
  id: string;
  label: string;
  value: number;
  min: number;
  max?: number;
  error?: string;
  onChange: (value: number) => void;
}

function NumberField({
  id,
  label,
  value,
  min,
  max,
  error,
  onChange,
}: NumberFieldProps) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        type="number"
        value={value}
        min={min}
        max={max}
        step={1}
        aria-invalid={error !== undefined}
        onChange={(event) => onChange(event.target.valueAsNumber)}
      />
      <FieldError message={error} />
    </div>
  );
}

interface CheckboxFieldProps {
  checked: boolean;
  label: string;
  description: string;
  onChange: (checked: boolean) => void;
}

function CheckboxField({
  checked,
  label,
  description,
  onChange,
}: CheckboxFieldProps) {
  return (
    <label className="checkbox-field">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span>
        <strong>{label}</strong>
        <small>{description}</small>
      </span>
    </label>
  );
}
