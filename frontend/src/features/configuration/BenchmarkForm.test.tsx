import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { app } from "../../../wailsjs/go/models";
import { benchmarkReport, benchmarkRequest, directoryInspection } from "../../test/fixtures";
import { ExecutionProvider } from "../execution/ExecutionContext";
import type { ProgressEvent, StateEvent } from "../execution/state";
import { BenchmarkForm } from "./BenchmarkForm";

const mocks = vi.hoisted(() => ({
  CancelBenchmark: vi.fn(),
  ExportReport: vi.fn(),
  InspectDirectory: vi.fn(),
  SelectDirectory: vi.fn(),
  SelectSuiteFile: vi.fn(),
  StartBenchmark: vi.fn(),
  ValidateBenchmark: vi.fn(),
  listeners: new Map<string, Set<(event: unknown) => void>>(),
}));

vi.mock("../../../wailsjs/go/app/Service", () => ({
  CancelBenchmark: mocks.CancelBenchmark,
  ExportReport: mocks.ExportReport,
  InspectDirectory: mocks.InspectDirectory,
  SelectDirectory: mocks.SelectDirectory,
  SelectSuiteFile: mocks.SelectSuiteFile,
  StartBenchmark: mocks.StartBenchmark,
  ValidateBenchmark: mocks.ValidateBenchmark,
}));

vi.mock("../../../wailsjs/runtime/runtime", () => ({
  EventsOn: vi.fn((name: string, callback: (event: unknown) => void) => {
    const listeners = mocks.listeners.get(name) ?? new Set();
    listeners.add(callback);
    mocks.listeners.set(name, listeners);
    return () => listeners.delete(callback);
  }),
}));

describe("BenchmarkForm", () => {
  beforeEach(() => {
    mocks.listeners.clear();
    for (const mock of [
      mocks.CancelBenchmark,
      mocks.ExportReport,
      mocks.InspectDirectory,
      mocks.SelectDirectory,
      mocks.SelectSuiteFile,
      mocks.StartBenchmark,
      mocks.ValidateBenchmark,
    ]) {
      mock.mockReset();
    }
    mocks.InspectDirectory.mockResolvedValue({
      inspection: directoryInspection(),
    });
    mocks.ValidateBenchmark.mockResolvedValue(validValidation());
    mocks.SelectDirectory.mockResolvedValue({ cancelled: true });
    mocks.SelectSuiteFile.mockResolvedValue({ cancelled: true });
  });

  it("renders basic and advanced configuration with conditional fields", async () => {
    const user = userEvent.setup();
    renderForm();

    expect(screen.getByLabelText("Target directory")).toHaveValue("C:\\benchmark");
    expect(screen.getByLabelText("Working-set size")).toHaveValue("1GiB");
    expect(screen.getByLabelText("Measured iterations")).toHaveValue(1);
    expect(screen.queryByLabelText("Read percentage")).not.toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Workload preset"), "random");
    expect(screen.getByLabelText("Read percentage")).toHaveValue(50);
    expect(screen.getByLabelText("Random seed")).toHaveValue("1");

    await user.click(screen.getByRole("radio", { name: /Suite file/ }));
    expect(screen.getByRole("button", { name: "Choose suite" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Workload preset")).not.toBeInTheDocument();
  });

  it("requires write-impact confirmation before starting", async () => {
    const user = userEvent.setup();
    mocks.ValidateBenchmark.mockResolvedValue({
      valid: true,
      errors: [],
      writeImpact: {
        requiresConfirmation: true,
        reasons: ["The configured duration may write repeatedly."],
        largeSizeThresholdBytes: "1073741824",
      },
    });
    mocks.StartBenchmark.mockReturnValue(new Promise(() => undefined));
    renderForm();

    await user.click(screen.getByRole("button", { name: "Start benchmark" }));

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("elevated write impact");
    expect(mocks.StartBenchmark).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Confirm and start" }));
    await waitFor(() => expect(mocks.StartBenchmark).toHaveBeenCalledOnce());
    expect(mocks.StartBenchmark.mock.calls[0][0].writeImpactConfirmed).toBe(true);
  });

  it("renders progress and sends cancellation for the active run", async () => {
    const user = userEvent.setup();
    mocks.StartBenchmark.mockReturnValue(new Promise(() => undefined));
    mocks.CancelBenchmark.mockResolvedValue({
      runId: "42",
      state: "cancelling",
    });
    renderForm();

    await user.click(screen.getByRole("button", { name: "Start benchmark" }));
    await waitFor(() => expect(mocks.StartBenchmark).toHaveBeenCalledOnce());

    emit<StateEvent>("benchmark:state", { runId: "42", state: "running" });
    emit<ProgressEvent>("benchmark:progress", {
      runId: "42",
      phase: "workload",
      status: "started",
      warmup: 0,
      warmupsTotal: 0,
      iteration: 1,
      iterationsTotal: 2,
      workload: "Sequential write",
      workloadNumber: 1,
      workloadsTotal: 2,
      indeterminate: false,
    });

    expect(await screen.findByRole("heading", { name: "Running workload" })).toBeVisible();
    expect(screen.getAllByText("Sequential write")).toHaveLength(2);
    expect(screen.getByText("1 of 2")).toBeVisible();
    expect(screen.getByText(/active operating-system I\/O call/i)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Cancel benchmark" }));
    expect(mocks.CancelBenchmark).toHaveBeenCalledWith("42");
    expect(await screen.findByText("Cancelling benchmark")).toBeVisible();
  });

  it("renders completed results and reports export failures", async () => {
    const user = userEvent.setup();
    mocks.StartBenchmark.mockResolvedValue({
      runId: "12",
      state: "completed",
      report: benchmarkReport(),
    });
    mocks.ExportReport.mockResolvedValue({
      cancelled: false,
      error: { code: "export", message: "permission denied" },
    });
    renderForm();

    await user.click(screen.getByRole("button", { name: "Start benchmark" }));

    expect(
      await screen.findByRole("heading", { name: "Benchmark completed" }),
    ).toBeVisible();
    expect(screen.getAllByText("Sequential write")).toHaveLength(2);
    expect(screen.getAllByText(/512[.,]00/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/verification/i).length).toBeGreaterThan(0);

    await user.click(screen.getByRole("tab", { name: "Measurements" }));
    expect(screen.getByRole("table")).toBeVisible();
    await user.click(screen.getByRole("tab", { name: "Run details" }));
    expect(screen.getByRole("heading", { name: "Environment" })).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Export JSON" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("permission denied");
  });

  it("keeps event-delivered results when the blocking RPC resolves afterward", async () => {
    const user = userEvent.setup();
    let resolveRun!: (result: app.BenchmarkResult) => void;
    mocks.StartBenchmark.mockReturnValue(
      new Promise<app.BenchmarkResult>((resolve) => {
        resolveRun = resolve;
      }),
    );
    renderForm();

    await user.click(screen.getByRole("button", { name: "Start benchmark" }));
    await waitFor(() => expect(mocks.StartBenchmark).toHaveBeenCalledOnce());

    emit<StateEvent>("benchmark:state", { runId: "21", state: "completed" });
    emit("benchmark:completed", {
      runId: "21",
      report: benchmarkReport(),
    });

    expect(
      await screen.findByRole("heading", { name: "Benchmark completed" }),
    ).toBeVisible();

    act(() => {
      resolveRun({
        runId: "21",
        state: "completed",
        report: benchmarkReport(),
      });
    });

    expect(
      screen.getByRole("heading", { name: "Benchmark completed" }),
    ).toBeVisible();
  });

  it("retains editable configuration after an execution failure", async () => {
    const user = userEvent.setup();
    mocks.StartBenchmark.mockResolvedValue({
      runId: "13",
      state: "failed",
      error: { code: "execution", message: "disk is full" },
    });
    renderForm();

    await user.clear(screen.getByLabelText("Target directory"));
    await user.type(screen.getByLabelText("Target directory"), "D:\\retry");
    await user.click(screen.getByRole("button", { name: "Start benchmark" }));

    expect(
      await screen.findByRole("heading", { name: "Benchmark did not complete" }),
    ).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("disk is full");
    expect(screen.getByLabelText("Target directory")).toHaveValue("D:\\retry");
  });
});

function renderForm() {
  return render(
    <ExecutionProvider>
      <BenchmarkForm initialRequest={benchmarkRequest()} />
    </ExecutionProvider>,
  );
}

function emit<Event>(name: string, event: Event) {
  act(() => {
    for (const listener of mocks.listeners.get(name) ?? []) {
      listener(event);
    }
  });
}

function validValidation(): app.ValidationResult {
  return {
    valid: true,
    errors: [],
    writeImpact: {
      requiresConfirmation: false,
      reasons: [],
      largeSizeThresholdBytes: "1073741824",
    },
  };
}
