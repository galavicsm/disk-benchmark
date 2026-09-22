export namespace app {
	
	export interface BenchmarkRequest {
	    directory: string;
	    size: string;
	    blockSize: string;
	    iterations: number;
	    warmupIterations: number;
	    keepFile: boolean;
	    verifyData: boolean;
	    syncWrites: boolean;
	    workload: string;
	    suiteFile: string;
	    duration: string;
	    workers: number;
	    queueDepth: number;
	    randomReadPercent: number;
	    randomSeed: string;
	    ioMode: string;
	    cacheControl: string;
	    writeImpactConfirmed: boolean;
	}
	export interface ServiceError {
	    code: string;
	    field?: string;
	    message: string;
	}
	export interface Summary {
	    name: string;
	    minimumMibPerSecond: number;
	    maximumMibPerSecond: number;
	    averageMibPerSecond: number;
	    minimumIops: number;
	    maximumIops: number;
	    averageIops: number;
	}
	export interface VerificationResult {
	    enabled: boolean;
	    passed: boolean;
	    bytes: string;
	    durationNs: string;
	}
	export interface FileStorage {
	    logicalBytes: string;
	    allocatedBytes: string;
	    sparse: boolean;
	}
	export interface CacheControlResult {
	    mode: string;
	    api?: string;
	    status: string;
	    detail?: string;
	}
	export interface LatencyStats {
	    averageNs: string;
	    p50Ns: string;
	    p95Ns: string;
	    p99Ns: string;
	    maximumNs: string;
	    sampleCount: string;
	    bucketCount: number;
	}
	export interface Measurement {
	    name: string;
	    bytes: string;
	    operations: string;
	    reads: string;
	    writes: string;
	    durationNs: string;
	    targetDurationNs: string;
	    latency: LatencyStats;
	    cacheControl: CacheControlResult;
	    mibPerSecond: number;
	    iops: number;
	}
	export interface IterationResult {
	    number: number;
	    filePath?: string;
	    measurements: Measurement[];
	    storage: FileStorage;
	    verification: VerificationResult;
	}
	export interface Environment {
	    hostname: string;
	    operatingSystem: string;
	    architecture: string;
	    cpus: number;
	    goVersion: string;
	    filesystem: string;
	    volume: string;
	    availableBytesBefore: string;
	    availableBytesAfter: string;
	}
	export interface ReportConfig {
	    directory: string;
	    sizeBytes: string;
	    blockSizeBytes: string;
	    iterations: number;
	    keepFile: boolean;
	    syncWrites: boolean;
	    workers: number;
	    queueDepth: number;
	    randomReadPercent: number;
	    randomSeed: string;
	    ioMode: string;
	    ioAlignmentBytes: string;
	    cacheControl: string;
	    warmupIterations: number;
	    verifyData: boolean;
	    durationNs: string;
	    suiteFile?: string;
	    suiteVersion?: number;
	}
	export interface Report {
	    startedAt: string;
	    config: ReportConfig;
	    environment: Environment;
	    iterations: IterationResult[];
	    summaries: Summary[];
	    warmupsCompleted: number;
	}
	export interface BenchmarkResult {
	    runId?: string;
	    state: string;
	    report?: Report;
	    error?: ServiceError;
	}
	
	export interface CancelResult {
	    runId?: string;
	    state: string;
	    error?: ServiceError;
	}
	export interface DirectIOInspection {
	    capability: string;
	    alignmentBytes?: string;
	    detail?: string;
	}
	export interface DirectoryInspection {
	    resolvedPath: string;
	    filesystem: string;
	    volume: string;
	    availableBytes: string;
	    directIo: DirectIOInspection;
	}
	export interface DirectoryInspectionResult {
	    inspection?: DirectoryInspection;
	    error?: ServiceError;
	}
	
	export interface ExportRequest {
	    runId: string;
	    format: string;
	}
	export interface ExportResult {
	    path?: string;
	    cancelled: boolean;
	    error?: ServiceError;
	}
	
	
	
	
	export interface SuiteWorkloadSummary {
	    name: string;
	    type: string;
	    duration?: string;
	    randomReadPercent?: number;
	    randomSeed?: string;
	}
	export interface SuiteSummary {
	    version: number;
	    workloads: SuiteWorkloadSummary[];
	}
	export interface PathSelectionResult {
	    path?: string;
	    cancelled: boolean;
	    suite?: SuiteSummary;
	    error?: ServiceError;
	}
	
	
	
	
	
	
	export interface WriteImpactAssessment {
	    requiresConfirmation: boolean;
	    reasons: string[];
	    largeSizeThresholdBytes: string;
	}
	export interface ValidationResult {
	    valid: boolean;
	    errors: ServiceError[];
	    writeImpact: WriteImpactAssessment;
	}
	

}

