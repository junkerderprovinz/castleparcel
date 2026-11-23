// What the Info page's Check for updates makes of the newest release tag the
// server reports.
import type { UpdateState } from "@glimstone/core";

function parts(version: string): number[] {
  return version.replace(/^v/, "").split(/[+-]/)[0]!.split(".").map((n) => Number.parseInt(n, 10) || 0);
}

/** Whether release tag `latest`, such as "v1.4.0", is newer than `running`. */
export function isNewer(latest: string, running: string): boolean {
  const a = parts(latest);
  const b = parts(running);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const d = (a[i] ?? 0) - (b[i] ?? 0);
    if (d) return d > 0;
  }
  return false;
}

/** The Version card's state for the server's answer to GET /api/latest-release. */
export function releaseState(answer: { latest?: string; checkedAt?: string }, running: string): UpdateState {
  // The server's reason is English and for its log, so the card says only that the check failed.
  if (!answer.latest) return { kind: "failed" };
  if (isNewer(answer.latest, running)) return { kind: "available", version: answer.latest.replace(/^v/, "") };
  return { kind: "current", checkedAt: answer.checkedAt ? new Date(answer.checkedAt) : new Date() };
}
