import type { app } from "../../wailsjs/go/models";

export function benchmarkRequest(
  overrides: Partial<app.BenchmarkRequest> = {},
): app.BenchmarkRequest {
  return {
    directory: "C:\\benchmark",
    size: "1GiB",
    blockSize: "1MiB",
    iterations: 1,
    warmupIterations: 0,
    keepFile: false,
    verifyData: false,
    syncWrites: true,
    workload: "sequential",
    suiteFile: "",
    duration: "0s",
    workers: 1,
    queueDepth: 1,
    randomReadPercent: 50,
    randomSeed: "1",
    ioMode: "buffered",
    cacheControl: "off",
    writeImpactConfirmed: false,
    ...overrides,
  };
}

export function directoryInspection(): app.DirectoryInspection {
  return {
    resolvedPath: "C:\\benchmark",
    filesystem: "NTFS",
    volume: "C:",
    availableBytes: "1099511627776",
    directIo: {
      capability: "available",
      alignmentBytes: "4096",
    },
  };
}

export function benchmarkReport(): app.Report {
  return {
    startedAt: "2026-08-31T12:00:00Z",
    config: {
      directory: "C:\\benchmark",
      sizeBytes: "1073741824",
      blockSizeBytes: "1048576",
      iterations: 1,
      keepFile: true,
      syncWrites: true,
      workers: 1,
      queueDepth: 1,
      randomReadPercent: 50,
      randomSeed: "1",
      ioMode: "buffered",
      ioAlignmentBytes: "4096",
      cacheControl: "off",
      warmupIterations: 0,
      verifyData: true,
      durationNs: "0",
    },
    environment: {
      hostname: "test-host",
      operatingSystem: "windows",
      architecture: "amd64",
      cpus: 8,
      goVersion: "go1.25.0",
      filesystem: "NTFS",
      volume: "C:",
      availableBytesBefore: "1099511627776",
      availableBytesAfter: "1098437885952",
    },
    iterations: [
      {
        number: 1,
        filePath: "C:\\benchmark\\diskbenchmark-test.tmp",
        measurements: [
          {
            name: "Sequential write",
            bytes: "1073741824",
            operations: "1024",
            reads: "0",
            writes: "1024",
            durationNs: "2000000000",
            targetDurationNs: "0",
            latency: {
              averageNs: "1953125",
              p50Ns: "1800000",
              p95Ns: "2400000",
              p99Ns: "2700000",
              maximumNs: "3000000",
              sampleCount: "1024",
              bucketCount: 20,
            },
            cacheControl: {
              mode: "off",
              status: "disabled",
            },
            mibPerSecond: 512,
            iops: 512,
          },
        ],
        storage: {
          logicalBytes: "1073741824",
          allocatedBytes: "1073741824",
          sparse: false,
        },
        verification: {
          enabled: true,
          passed: true,
          bytes: "1073741824",
          durationNs: "1000000000",
        },
      },
    ],
    summaries: [
      {
        name: "Sequential write",
        minimumMibPerSecond: 500,
        maximumMibPerSecond: 524,
        averageMibPerSecond: 512,
        minimumIops: 500,
        maximumIops: 524,
        averageIops: 512,
      },
    ],
    warmupsCompleted: 0,
  };
}
