import { describe, expect, it } from "vitest";
import { benchmarkRequest } from "../test/fixtures";
import { serializeBenchmarkRequest } from "./request";

describe("serializeBenchmarkRequest", () => {
  it("serializes every binding field without retaining the form object", () => {
    const request = benchmarkRequest({
      directory: "D:\\bench",
      workload: "",
      suiteFile: "D:\\suite.json",
      writeImpactConfirmed: true,
    });

    const serialized = serializeBenchmarkRequest(request);

    expect(serialized).toEqual(request);
    expect(serialized).not.toBe(request);
    expect(Object.keys(serialized)).toHaveLength(18);
  });
});
