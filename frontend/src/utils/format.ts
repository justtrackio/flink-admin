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
 * Formats an epoch timestamp in milliseconds to a human-readable date/time string.
 * @param epochMs - Epoch timestamp in milliseconds
 * @returns Formatted date/time string (e.g., "Feb 7, 2026, 10:30:45 AM")
 */
export function formatTimestamp(epochMs: number): string {
  const date = new Date(epochMs);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
