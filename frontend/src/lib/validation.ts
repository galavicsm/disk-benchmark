import type { app } from "../../wailsjs/go/models";
import { parseByteSize } from "./format";

export type FieldErrors = Partial<Record<keyof app.BenchmarkRequest, string>>;

const durationPattern = /^(?:0|(?:\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h))+)$/;
const requestFields = new Set<string>([
  "directory",
  "size",
  "blockSize",
  "iterations",
  "warmupIterations",
  "keepFile",
  "verifyData",
  "syncWrites",
  "workload",
  "suiteFile",
  "duration",
  "workers",
  "queueDepth",
  "randomReadPercent",
  "randomSeed",
  "ioMode",
  "cacheControl",
  "writeImpactConfirmed",
]);

export function isBenchmarkRequestField(
  value: string,
): value is keyof app.BenchmarkRequest {
  return requestFields.has(value);
}

export function validateRequestLocally(
  request: app.BenchmarkRequest,
  inspection: app.DirectoryInspection | null,
): FieldErrors {
  const errors: FieldErrors = {};
  const size = parseByteSize(request.size);
  const blockSize = parseByteSize(request.blockSize);

  if (request.directory.trim() === "") {
    errors.directory = "Choose a benchmark directory.";
  }
  if (size === null) {
    errors.size = "Use a positive binary size such as 1GiB or 64MiB.";
  }
  if (blockSize === null) {
    errors.blockSize = "Use a positive binary size such as 1MiB or 4KiB.";
  } else if (size !== null && blockSize > size) {
    errors.blockSize = "Block size must not exceed the working-set size.";
  }
  if (!Number.isInteger(request.iterations) || request.iterations < 1) {
    errors.iterations = "Use at least one measured iteration.";
  }
  if (!Number.isInteger(request.warmupIterations) || request.warmupIterations < 0) {
    errors.warmupIterations = "Warm-up iterations cannot be negative.";
  }
  if (!Number.isInteger(request.workers) || request.workers < 1) {
    errors.workers = "Use at least one worker.";
  }
  if (!Number.isInteger(request.queueDepth) || request.queueDepth < 1) {
    errors.queueDepth = "Use a queue depth of at least one.";
  }
  if (
    !Number.isInteger(request.randomReadPercent) ||
    request.randomReadPercent < 0 ||
    request.randomReadPercent > 100
  ) {
    errors.randomReadPercent = "Enter a whole-number percentage from 0 to 100.";
  }
  if (!/^-?\d+$/.test(request.randomSeed.trim())) {
    errors.randomSeed = "Enter a whole-number random seed.";
  }
  if (request.duration.trim() !== "" && !durationPattern.test(request.duration.trim())) {
    errors.duration = "Use a Go duration such as 30s, 500ms, or 2m.";
  }
  if (request.suiteFile !== "" && request.workload !== "") {
    errors.suiteFile = "Choose either a suite file or a built-in workload, not both.";
  } else if (request.suiteFile === "" && request.workload === "") {
    errors.suiteFile = "Choose a suite file.";
  }

  if (inspection !== null) {
    if (size !== null && size > BigInt(inspection.availableBytes)) {
      errors.size = "The working set exceeds currently available space.";
    }
    if (
      request.ioMode === "direct" &&
      inspection.directIo.capability === "available" &&
      inspection.directIo.alignmentBytes !== undefined
    ) {
      const alignment = BigInt(inspection.directIo.alignmentBytes);
      if (blockSize !== null && blockSize % alignment !== 0n) {
        errors.blockSize = `Direct I/O requires a multiple of ${alignment} bytes.`;
      } else if (size !== null && size % alignment !== 0n) {
        errors.size = `Direct I/O requires a multiple of ${alignment} bytes.`;
      }
    }
  }

  return errors;
}
