const byteUnits = [
  { suffix: "TiB", size: 1024n ** 4n },
  { suffix: "GiB", size: 1024n ** 3n },
  { suffix: "MiB", size: 1024n ** 2n },
  { suffix: "KiB", size: 1024n },
] as const;

export function formatBytes(value: string): string {
  let bytes: bigint;
  try {
    bytes = BigInt(value);
  } catch {
    return value;
  }

  for (const unit of byteUnits) {
    if (bytes >= unit.size) {
      const hundredths = (bytes * 100n) / unit.size;
      return `${hundredths / 100n}.${(hundredths % 100n).toString().padStart(2, "0")} ${unit.suffix}`;
    }
  }
  return `${bytes} B`;
}

export function formatDuration(value: string): string {
  let nanoseconds: bigint;
  try {
    nanoseconds = BigInt(value);
  } catch {
    return value;
  }

  const units = [
    { suffix: "h", size: 3_600_000_000_000n },
    { suffix: "m", size: 60_000_000_000n },
    { suffix: "s", size: 1_000_000_000n },
    { suffix: "ms", size: 1_000_000n },
    { suffix: "us", size: 1_000n },
  ] as const;
  const unit = units.find((candidate) => nanoseconds >= candidate.size);
  if (unit === undefined) {
    return `${nanoseconds} ns`;
  }

  const hundredths = (nanoseconds * 100n) / unit.size;
  const fraction = hundredths % 100n;
  return fraction === 0n
    ? `${hundredths / 100n} ${unit.suffix}`
    : `${hundredths / 100n}.${fraction.toString().padStart(2, "0")} ${unit.suffix}`;
}

export function formatElapsed(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${seconds
      .toString()
      .padStart(2, "0")}`;
  }
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export function formatMetric(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

export function formatCount(value: string): string {
  try {
    return new Intl.NumberFormat().format(BigInt(value));
  } catch {
    return value;
  }
}

export function parseByteSize(value: string): bigint | null {
  const match = value.trim().match(/^(\d+)(?:\.(\d+))?\s*(B|KiB|MiB|GiB|TiB)?$/i);
  if (match === null) {
    return null;
  }
  const whole = BigInt(match[1]);
  const fraction = match[2] ?? "";
  const suffix = (match[3] ?? "B").toUpperCase();
  const multiplier = new Map<string, bigint>([
    ["B", 1n],
    ["KIB", 1024n],
    ["MIB", 1024n ** 2n],
    ["GIB", 1024n ** 3n],
    ["TIB", 1024n ** 4n],
  ]).get(suffix);
  if (multiplier === undefined) {
    return null;
  }

  const denominator = 10n ** BigInt(fraction.length);
  const numerator = whole * denominator + BigInt(fraction || "0");
  const scaled = numerator * multiplier;
  if (scaled % denominator !== 0n) {
    return null;
  }
  const bytes = scaled / denominator;
  return bytes > 0n ? bytes : null;
}
