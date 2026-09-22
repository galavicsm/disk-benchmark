import { describe, expect, it } from "vitest";
import { benchmarkRequest, directoryInspection } from "../test/fixtures";
import { validateRequestLocally } from "./validation";

describe("validateRequestLocally", () => {
  it("accepts a valid built-in workload", () => {
    expect(
      validateRequestLocally(benchmarkRequest(), directoryInspection()),
    ).toEqual({});
  });

  it("validates conditional suite and random fields", () => {
    const errors = validateRequestLocally(
      benchmarkRequest({
        workload: "",
        suiteFile: "",
        randomReadPercent: 101,
        randomSeed: "seed",
      }),
      null,
    );

    expect(errors.suiteFile).toMatch(/suite file/i);
    expect(errors.randomReadPercent).toMatch(/0 to 100/i);
    expect(errors.randomSeed).toMatch(/whole-number/i);
  });

  it("enforces direct-I/O alignment from inspection", () => {
    const errors = validateRequestLocally(
      benchmarkRequest({ ioMode: "direct", blockSize: "5KiB" }),
      directoryInspection(),
    );

    expect(errors.blockSize).toBe(
      "Direct I/O requires a multiple of 4096 bytes.",
    );
  });
});
