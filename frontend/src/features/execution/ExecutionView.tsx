import { useEffect, useMemo, useState } from "react";
import { formatElapsed } from "../../lib/format";
import type { ProgressEvent, RunState } from "./state";

interface ExecutionViewProps {
  state: RunState;
  progress: ProgressEvent | null;
  startedAt: number;
  canCancel: boolean;
  errorMessage: string;
  onCancel: () => void;
}

const phaseLabels: Record<string, string> = {
  preflight: "Preflight checks",
  warmup: "Warm-up",
  iteration: "Measured iteration",
  "workload-preparation": "Preparing workload",
  workload: "Running workload",
  verification: "Verifying data",
  cleanup: "Cleaning up",
};

export function ExecutionView({
  state,
  progress,
  startedAt,
  canCancel,
  errorMessage,
  onCancel,
}: ExecutionViewProps) {
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(interval);
  }, []);

  const progressValue = useMemo(() => calculateProgress(progress), [progress]);
  const indeterminate =
    progress === null || progress.indeterminate || progressValue === null;
  const phase =
    state === "validating"
      ? "Validating configuration"
      : state === "cancelling"
        ? "Cancelling benchmark"
        : phaseLabels[progress?.phase ?? ""] ?? "Starting benchmark";

  return (
    <section className="panel execution-view" aria-labelledby="execution-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Execution</p>
          <h2 id="execution-heading">{phase}</h2>
        </div>
        <span className={`state-badge ${state}`}>{state}</span>
      </div>

      <div className="execution-facts">
        <ExecutionFact
          label="Warm-up"
          value={counter(progress?.warmup, progress?.warmupsTotal)}
        />
        <ExecutionFact
          label="Measured iteration"
          value={counter(progress?.iteration, progress?.iterationsTotal)}
        />
        <ExecutionFact
          label="Workload"
          value={
            progress?.workload ||
            counter(progress?.workloadNumber, progress?.workloadsTotal)
          }
        />
        <ExecutionFact label="Elapsed time" value={formatElapsed(now - startedAt)} />
      </div>

      <div className="progress-block" aria-live="polite">
        <div>
          <strong>{progress?.workload ?? phase}</strong>
          <span>
            {indeterminate
              ? "Completion time depends on the active operation."
              : `${Math.round(progressValue * 100)}% complete`}
          </span>
        </div>
        <progress
          aria-label="Benchmark progress"
          max={1}
          value={indeterminate ? undefined : progressValue}
        />
      </div>

      {errorMessage !== "" && (
        <div className="error-banner" role="alert">
          <strong>Unable to update benchmark state</strong>
          <span>{errorMessage}</span>
        </div>
      )}

      <aside className="cancellation-notice">
        <div>
          <strong>Cancellation is cooperative</strong>
          <p>
            Stopping can take until the active operating-system I/O call returns.
            DiskBenchmark will then close and clean up its temporary file.
          </p>
        </div>
        <button
          type="button"
          className="button danger"
          disabled={!canCancel || state === "cancelling"}
          onClick={onCancel}
        >
          {state === "cancelling" ? "Cancelling..." : "Cancel benchmark"}
        </button>
      </aside>
    </section>
  );
}

function ExecutionFact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function counter(current?: number, total?: number): string {
  if (total === undefined || total === 0) {
    return "Not applicable";
  }
  return `${current ?? 0} of ${total}`;
}

function calculateProgress(progress: ProgressEvent | null): number | null {
  if (progress === null || progress.indeterminate) {
    return null;
  }
  const totalPasses = progress.warmupsTotal + progress.iterationsTotal;
  if (totalPasses === 0) {
    return null;
  }
  if (progress.phase === "preflight") {
    return progress.status === "completed" ? 0.02 : 0;
  }

  const passNumber =
    progress.warmup > 0
      ? progress.warmup
      : progress.warmupsTotal + Math.max(progress.iteration, 1);
  let fraction = 0;
  if (progress.phase === "workload-preparation") {
    fraction = Math.max(progress.workloadNumber - 1, 0) /
      Math.max(progress.workloadsTotal, 1);
  } else if (progress.phase === "workload") {
    fraction =
      (Math.max(progress.workloadNumber - 1, 0) +
        (progress.status === "completed" ? 1 : 0.5)) /
      Math.max(progress.workloadsTotal, 1);
  } else if (progress.phase === "verification") {
    fraction = progress.status === "completed" ? 0.95 : 0.9;
  } else if (progress.phase === "cleanup") {
    fraction = progress.status === "completed" ? 1 : 0.97;
  } else if (progress.status === "completed") {
    fraction = 1;
  }
  return Math.min(1, (passNumber - 1 + fraction) / totalPasses);
}
