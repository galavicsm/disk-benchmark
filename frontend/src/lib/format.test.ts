import { describe, expect, it } from "vitest";
import {
  formatBytes,
  formatCount,
  formatDuration,
  formatElapsed,
  formatMetric,
} from "./format";

describe("metric formatting", () => {
  it("formats bytes and durations from exact decimal strings", () => {
    expect(formatBytes("1073741824")).toBe("1.00 GiB");
    expect(formatDuration("1500000000")).toBe("1.50 s");
    expect(formatDuration("2500")).toBe("2.50 us");
  });

  it("formats elapsed time and numeric metrics", () => {
    expect(formatElapsed(3_661_000)).toBe("1:01:01");
    expect(formatMetric(512)).toMatch(/512[.,]00/);
    expect(formatCount("1024")).toMatch(/1.?024/);
  });

  it("preserves invalid binding values rather than fabricating metrics", () => {
    expect(formatBytes("unknown")).toBe("unknown");
    expect(formatDuration("unknown")).toBe("unknown");
    expect(formatCount("unknown")).toBe("unknown");
  });
});
