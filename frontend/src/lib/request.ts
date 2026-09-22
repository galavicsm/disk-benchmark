import type { app } from "../../wailsjs/go/models";

export function serializeBenchmarkRequest(
  request: app.BenchmarkRequest,
): app.BenchmarkRequest {
  return {
    directory: request.directory,
    size: request.size,
    blockSize: request.blockSize,
    iterations: request.iterations,
    warmupIterations: request.warmupIterations,
    keepFile: request.keepFile,
    verifyData: request.verifyData,
    syncWrites: request.syncWrites,
    workload: request.workload,
    suiteFile: request.suiteFile,
    duration: request.duration,
    workers: request.workers,
    queueDepth: request.queueDepth,
    randomReadPercent: request.randomReadPercent,
    randomSeed: request.randomSeed,
    ioMode: request.ioMode,
    cacheControl: request.cacheControl,
    writeImpactConfirmed: request.writeImpactConfirmed,
  };
}
