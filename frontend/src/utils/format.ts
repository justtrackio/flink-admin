/**
 * Shared formatting utilities for the flink admin frontend.
 */

/**
 * Formats a number with locale-specific thousand separators.
 * @param num - The number to format
 * @returns Formatted number string (e.g., "1,234,567")
 */
export function formatNumber(num: number): string {
  return num.toLocaleString('en-US');
}

/**
 * Formats a timestamp into a relative age string (e.g., "5m ago", "3h ago", "2d ago").
 * @param timestamp - ISO 8601 timestamp string
 * @returns Relative age string
 */
export function formatAge(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffDay > 0) return `${diffDay}d ago`;
  if (diffHour > 0) return `${diffHour}h ago`;
  if (diffMin > 0) return `${diffMin}m ago`;
  return `${diffSec}s ago`;
}

/**
 * Extracts the image tag from a full image URL.
 * @param image - Full image URL (e.g., "531429218937.dkr.ecr.eu-central-1.amazonaws.com/flink-lakeingester:stable-1.20.3-java11")
 * @returns Image tag (e.g., "stable-1.20.3-java11")
 */
export function formatImageTag(image: string): string {
  const parts = image.split(':');
  return parts.length > 1 ? parts[parts.length - 1] : image;
}

/**
 * Formats a timestamp to a UTC date/time string.
 * @param timestamp - ISO 8601 string or epoch timestamp in seconds/milliseconds
 * @returns Formatted date/time string (e.g., "2026-06-04 02:48:01 UTC")
 */
export function formatTimestamp(timestamp: string | number): string {
  const date = parseTimestamp(timestamp);

  if (!date) {
    return 'Invalid timestamp';
  }

  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  const hours = String(date.getUTCHours()).padStart(2, '0');
  const minutes = String(date.getUTCMinutes()).padStart(2, '0');
  const seconds = String(date.getUTCSeconds()).padStart(2, '0');

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} UTC`;
}

export const formatTimestampUtc = formatTimestamp;

function parseTimestamp(timestamp: string | number): Date | null {
  if (typeof timestamp === 'number') {
    return createDateFromEpoch(timestamp);
  }

  const trimmedTimestamp = timestamp.trim();

  if (!trimmedTimestamp) {
    return null;
  }

  const numericTimestamp = Number(trimmedTimestamp);

  if (!Number.isNaN(numericTimestamp)) {
    return createDateFromEpoch(numericTimestamp);
  }

  const date = new Date(trimmedTimestamp);
  return Number.isNaN(date.getTime()) ? null : date;
}

function createDateFromEpoch(epoch: number): Date | null {
  const epochMs = Math.abs(epoch) < 1_000_000_000_000 ? epoch * 1000 : epoch;
  const date = new Date(epochMs);

  return Number.isNaN(date.getTime()) ? null : date;
}

/**
 * Formats a byte size into a human-readable string (e.g., "1.5 MB", "3.2 GB").
 * @param bytes - Size in bytes
 * @returns Formatted byte size string
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

/**
 * Formats a duration in milliseconds into a human-readable string (e.g., "2.5s", "1m 30s").
 * @param durationMs - Duration in milliseconds
 * @returns Formatted duration string
 */
export function formatDuration(durationMs: number): string {
  if (durationMs < 1000) {
    return `${durationMs}ms`;
  }
  
  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  
  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
  }
  
  if (minutes > 0) {
    const remainingSeconds = seconds % 60;
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
  }
  
  return `${seconds}s`;
}
