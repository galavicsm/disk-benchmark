import { useState } from "react";
import type { app } from "../../../wailsjs/go/models";
import { ExportReport } from "../../../wailsjs/go/app/Service";
import {
  formatBytes,
  formatCount,
  formatDuration,
  formatMetric,
} from "../../lib/format";

interface ResultsViewProps {
  runId: string;
  report: app.Report;
  onNewBenchmark: () => void;
}

type ExportState =
  | { status: "idle" }
  | { status: "exporting"; format: "json" | "csv" }
  | { status: "success"; message: string }
  | { status: "failed"; message: string };

type ResultsTab = "summary" | "measurements" | "details";

const summaryPageSize = 4;
const measurementPageSize = 4;

export function ResultsView({
  runId,
  report,
  onNewBenchmark,
}: ResultsViewProps) {
  const [exportState, setExportState] = useState<ExportState>({ status: "idle" });
  const [activeTab, setActiveTab] = useState<ResultsTab>("summary");
  const [summaryPage, setSummaryPage] = useState(0);
  const [iterationIndex, setIterationIndex] = useState(0);
  const [measurementPage, setMeasurementPage] = useState(0);

  const summaryPageCount = Math.max(
    1,
    Math.ceil(report.summaries.length / summaryPageSize),
  );
  const visibleSummaries = report.summaries.slice(
    summaryPage * summaryPageSize,
    (summaryPage + 1) * summaryPageSize,
  );
  const selectedIteration = report.iterations[iterationIndex];
  const measurementPageCount = Math.max(
    1,
    Math.ceil(
      (selectedIteration?.measurements.length ?? 0) / measurementPageSize,
    ),
  );
  const visibleMeasurements =
    selectedIteration?.measurements.slice(
      measurementPage * measurementPageSize,
      (measurementPage + 1) * measurementPageSize,
    ) ?? [];

  const exportReport = async (format: "json" | "csv") => {
    setExportState({ status: "exporting", format });
    try {
      const result = await ExportReport({ runId, format });
      if (result.error !== undefined) {
        setExportState({ status: "failed", message: result.error.message });
      } else if (result.cancelled) {
        setExportState({ status: "idle" });
      } else {
        setExportState({
          status: "success",
          message: `Report saved to ${result.path ?? "the selected destination"}.`,
        });
      }
    } catch (error: unknown) {
      setExportState({
        status: "failed",
        message: error instanceof Error ? error.message : String(error),
      });
    }
  };

  return (
    <div className="results-view">
      <section className="panel results-heading" aria-labelledby="results-heading">
        <div>
          <p className="eyebrow">Results</p>
          <h2 id="results-heading">Benchmark completed</h2>
          <p>
            Started {new Date(report.startedAt).toLocaleString()} ·{" "}
            {report.iterations.length} measured iteration
            {report.iterations.length === 1 ? "" : "s"}
          </p>
        </div>
        <div className="results-actions">
          <button type="button" className="button secondary" onClick={onNewBenchmark}>
            New benchmark
          </button>
          <button
            type="button"
            className="button secondary"
            disabled={exportState.status === "exporting"}
            onClick={() => void exportReport("json")}
          >
            Export JSON
          </button>
          <button
            type="button"
            className="button primary"
            disabled={exportState.status === "exporting"}
            onClick={() => void exportReport("csv")}
          >
            Export CSV
          </button>
        </div>
      </section>

      {exportState.status === "failed" && (
        <div className="error-banner" role="alert">
          <strong>Export failed</strong>
          <span>{exportState.message}</span>
        </div>
      )}
      {exportState.status === "success" && (
        <div className="success-banner" role="status">
          <strong>Export complete</strong>
          <span>{exportState.message}</span>
        </div>
      )}

      <div className="results-tabs" role="tablist" aria-label="Result sections">
        <ResultTab
          id="summary"
          label="Summary"
          activeTab={activeTab}
          onSelect={setActiveTab}
        />
        <ResultTab
          id="measurements"
          label="Measurements"
          activeTab={activeTab}
          onSelect={setActiveTab}
        />
        <ResultTab
          id="details"
          label="Run details"
          activeTab={activeTab}
          onSelect={setActiveTab}
        />
      </div>

      <section
        id="summary-panel"
        className="results-tab-panel"
        role="tabpanel"
        aria-labelledby="summary-tab"
        hidden={activeTab !== "summary"}
      >
        <div className="content-heading">
          <div>
            <p className="eyebrow">Summary</p>
            <h2>Workload performance</h2>
          </div>
          <PageControls
            label="Summary workloads"
            page={summaryPage}
            pageCount={summaryPageCount}
            onPageChange={setSummaryPage}
          />
        </div>
        <div className="summary-grid">
          {visibleSummaries.map((summary) => (
            <article className="panel summary-card" key={summary.name}>
              <h3>{summary.name}</h3>
              <MetricGroup
                label="MiB/s"
                average={summary.averageMibPerSecond}
                minimum={summary.minimumMibPerSecond}
                maximum={summary.maximumMibPerSecond}
              />
              <MetricGroup
                label="IOPS"
                average={summary.averageIops}
                minimum={summary.minimumIops}
                maximum={summary.maximumIops}
              />
            </article>
          ))}
        </div>
      </section>

      <section
        id="measurements-panel"
        className="panel details-panel results-tab-panel"
        role="tabpanel"
        aria-labelledby="measurements-tab"
        hidden={activeTab !== "measurements"}
      >
        <div className="content-heading measurement-heading">
          <div>
            <p className="eyebrow">Measurements</p>
            <h2>Per-iteration details</h2>
          </div>
          <label className="iteration-picker">
            <span>Iteration</span>
            <select
              value={iterationIndex}
              onChange={(event) => {
                setIterationIndex(Number(event.target.value));
                setMeasurementPage(0);
              }}
            >
              {report.iterations.map((iteration, index) => (
                <option value={index} key={iteration.number}>
                  {iteration.number}
                </option>
              ))}
            </select>
          </label>
          <PageControls
            label="Measurements"
            page={measurementPage}
            pageCount={measurementPageCount}
            onPageChange={setMeasurementPage}
          />
        </div>
        {selectedIteration !== undefined && (
          <div className="iteration-details">
            <div className="iteration-title">
              <strong>Iteration {selectedIteration.number}</strong>
              <span>{selectedIteration.measurements.length} workloads</span>
            </div>
            <div className="measurement-table-wrap">
              <table className="measurement-table">
                <thead>
                  <tr>
                    <th>Workload</th>
                    <th>Throughput</th>
                    <th>IOPS</th>
                    <th>Bytes</th>
                    <th>Operations</th>
                    <th>Reads / writes</th>
                    <th>Elapsed / target</th>
                    <th>Latency avg / p50 / p95 / p99 / max</th>
                    <th>Cache control</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleMeasurements.map((measurement) => (
                    <tr key={measurement.name}>
                      <th scope="row">{measurement.name}</th>
                      <td>{formatMetric(measurement.mibPerSecond)} MiB/s</td>
                      <td>{formatMetric(measurement.iops)}</td>
                      <td>{formatBytes(measurement.bytes)}</td>
                      <td>{formatCount(measurement.operations)}</td>
                      <td>
                        {formatCount(measurement.reads)} /{" "}
                        {formatCount(measurement.writes)}
                      </td>
                      <td>
                        {formatDuration(measurement.durationNs)} /{" "}
                        {measurement.targetDurationNs === "0"
                          ? "Not set"
                          : formatDuration(measurement.targetDurationNs)}
                      </td>
                      <td className="latency-cell">
                        {[
                          measurement.latency.averageNs,
                          measurement.latency.p50Ns,
                          measurement.latency.p95Ns,
                          measurement.latency.p99Ns,
                          measurement.latency.maximumNs,
                        ]
                          .map(formatDuration)
                          .join(" / ")}
                      </td>
                      <td>
                        <strong>
                          {measurement.cacheControl.status || "not reported"}
                        </strong>
                        <span className="table-detail">
                          {[
                            measurement.cacheControl.mode,
                            measurement.cacheControl.api,
                            measurement.cacheControl.detail,
                          ]
                            .filter(Boolean)
                            .join(" · ")}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="iteration-footer">
              <ResultFact
                label="Logical size"
                value={formatBytes(selectedIteration.storage.logicalBytes)}
              />
              <ResultFact
                label="Allocated size"
                value={formatBytes(selectedIteration.storage.allocatedBytes)}
              />
              <ResultFact
                label="Sparse/compressed allocation"
                value={selectedIteration.storage.sparse ? "Detected" : "Not detected"}
              />
              <ResultFact
                label="Verification"
                value={
                  selectedIteration.verification.enabled
                    ? selectedIteration.verification.passed
                      ? `Passed · ${formatBytes(selectedIteration.verification.bytes)} in ${formatDuration(selectedIteration.verification.durationNs)}`
                      : "Failed"
                    : "Not requested"
                }
              />
              {selectedIteration.filePath !== undefined && (
                <ResultFact
                  label="Retained file"
                  value={selectedIteration.filePath}
                  wide
                />
              )}
            </div>
          </div>
        )}
      </section>

      <section
        id="details-panel"
        className="metadata-grid results-tab-panel"
        role="tabpanel"
        aria-labelledby="details-tab"
        hidden={activeTab !== "details"}
      >
        <article className="panel metadata-card">
          <h2>Environment</h2>
          <dl>
            <Metadata label="Host" value={report.environment.hostname} />
            <Metadata
              label="Platform"
              value={`${report.environment.operatingSystem} / ${report.environment.architecture}`}
            />
            <Metadata label="CPU count" value={String(report.environment.cpus)} />
            <Metadata label="Go version" value={report.environment.goVersion} />
            <Metadata
              label="Filesystem"
              value={report.environment.filesystem || "Unknown"}
            />
            <Metadata label="Volume" value={report.environment.volume || "Unknown"} />
            <Metadata
              label="Available before"
              value={formatBytes(report.environment.availableBytesBefore)}
            />
            <Metadata
              label="Available after"
              value={formatBytes(report.environment.availableBytesAfter)}
            />
          </dl>
        </article>
        <article className="panel metadata-card">
          <h2>Effective configuration</h2>
          <dl>
            <Metadata label="Directory" value={report.config.directory} />
            <Metadata label="Working set" value={formatBytes(report.config.sizeBytes)} />
            <Metadata label="Block size" value={formatBytes(report.config.blockSizeBytes)} />
            <Metadata label="I/O mode" value={report.config.ioMode} />
            <Metadata
              label="I/O alignment"
              value={formatBytes(report.config.ioAlignmentBytes)}
            />
            <Metadata label="Cache control" value={report.config.cacheControl} />
            <Metadata label="Workers" value={String(report.config.workers)} />
            <Metadata label="Queue depth" value={String(report.config.queueDepth)} />
            <Metadata
              label="Configured duration"
              value={
                report.config.durationNs === "0"
                  ? "Working set once"
                  : formatDuration(report.config.durationNs)
              }
            />
          </dl>
        </article>
      </section>
    </div>
  );
}

function ResultTab({
  id,
  label,
  activeTab,
  onSelect,
}: {
  id: ResultsTab;
  label: string;
  activeTab: ResultsTab;
  onSelect: (tab: ResultsTab) => void;
}) {
  const selected = activeTab === id;
  return (
    <button
      id={`${id}-tab`}
      type="button"
      role="tab"
      aria-selected={selected}
      aria-controls={`${id}-panel`}
      tabIndex={selected ? 0 : -1}
      onClick={() => onSelect(id)}
    >
      {label}
    </button>
  );
}

function PageControls({
  label,
  page,
  pageCount,
  onPageChange,
}: {
  label: string;
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
}) {
  if (pageCount <= 1) {
    return null;
  }
  return (
    <div className="page-controls" aria-label={`${label} pages`}>
      <button
        type="button"
        className="button secondary"
        disabled={page === 0}
        onClick={() => onPageChange(page - 1)}
      >
        Previous
      </button>
      <span>
        {page + 1} / {pageCount}
      </span>
      <button
        type="button"
        className="button secondary"
        disabled={page + 1 === pageCount}
        onClick={() => onPageChange(page + 1)}
      >
        Next
      </button>
    </div>
  );
}

function MetricGroup({
  label,
  average,
  minimum,
  maximum,
}: {
  label: string;
  average: number;
  minimum: number;
  maximum: number;
}) {
  return (
    <div className="metric-group">
      <div className="metric-primary">
        <span>Average {label}</span>
        <strong>{formatMetric(average)}</strong>
      </div>
      <dl>
        <div>
          <dt>Minimum</dt>
          <dd>{formatMetric(minimum)}</dd>
        </div>
        <div>
          <dt>Maximum</dt>
          <dd>{formatMetric(maximum)}</dd>
        </div>
      </dl>
    </div>
  );
}

function ResultFact({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <div className={wide ? "wide" : undefined}>
      <span>{label}</span>
      <strong title={value}>{value}</strong>
    </div>
  );
}

function Metadata({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd title={value}>{value}</dd>
    </div>
  );
}
