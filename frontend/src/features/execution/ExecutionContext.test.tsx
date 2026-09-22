import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ExecutionProvider } from "./ExecutionContext";

const runtime = vi.hoisted(() => ({
  EventsOn: vi.fn<
    (name: string, callback: (...data: unknown[]) => void) => () => void
  >(() => vi.fn()),
}));

vi.mock("../../../wailsjs/runtime/runtime", () => ({
  EventsOn: runtime.EventsOn,
}));

describe("ExecutionProvider", () => {
  it("subscribes once to every run event and removes every listener", () => {
    const removers = Array.from({ length: 5 }, () => vi.fn());
    runtime.EventsOn.mockReset();
    removers.forEach((remove) => runtime.EventsOn.mockReturnValueOnce(remove));

    const view = render(
      <ExecutionProvider>
        <div>child</div>
      </ExecutionProvider>,
    );

    expect(runtime.EventsOn.mock.calls.map((call) => call[0])).toEqual([
      "benchmark:state",
      "benchmark:progress",
      "benchmark:completed",
      "benchmark:failed",
      "benchmark:cancelled",
    ]);

    view.unmount();
    removers.forEach((remove) => expect(remove).toHaveBeenCalledOnce());
  });
});
