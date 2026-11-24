// What the fortress page shows of the server's view, in GlimStone's shapes: the other apps with a row per place
// they run, the permissions that involve CastleParcel in its own words, and another member's status window.
import type { FortressApp, FortressInstance, FortressLogLine, FortressPermission, FortressStatusReport } from "@glimstone/core";

export const SELF = { id: "castleparcel", name: "CastleParcel" };

/** Another member of the fortress (server-go/internal/fortress/routes.go). App and name are empty for one that never
 * said which app it runs. */
export type FortressMember = {
  id: string;
  app: string;
  name: string;
  version: string;
  icon?: string;
  device: string;
  deployment: string;
  host?: string;
  online: boolean;
  /** Reached through the relay alone. */
  relay: boolean;
  lastSeen?: number;
  /** Whether its pages open from here. */
  pages: boolean;
};

export type MemberPermission = { from: string; to: string; action: string; on: boolean };

/** A member taken out since the phrase last changed. */
export type RemovedMember = { app: string; device: string };

/** Where this CastleParcel stands in the fortress. It never carries the phrase. */
export type FortressInfo = {
  joined: boolean;
  removedHere: boolean;
  phraseChanged: boolean;
  device: string;
  relay: { mode: "project" | "off" | "own"; url: string; connected: boolean };
  lan: boolean;
  members: FortressMember[];
  permissions: MemberPermission[];
  removed: RemovedMember[];
};

function deviceKind(m: FortressMember): NonNullable<FortressInstance["deviceKind"]> {
  if (m.deployment === "phone" || m.deployment === "android") return "phone";
  if (m.deployment === "desktop") return "pc";
  return "server";
}

/** Whether another member claims m's app on m's server or device, so that Open asks first. */
const doubted = (m: FortressMember, all: FortressMember[]) =>
  all.some((o) => o.id !== m.id && !!m.app && o.app === m.app && ((!!m.host && o.host === m.host) || o.device === m.device));

/** The card a member goes on: its app's, or one of its own for a member that never said its app. */
const cardOf = (m: FortressMember) => m.app || m.id;

/** The other apps of the fortress, each with its instances, sorted by name. */
export function fortressApps(members: FortressMember[], ago: (unix: number) => string): FortressApp[] {
  const apps = new Map<string, FortressApp>();
  for (const m of members) {
    if (m.app === SELF.id) continue;
    const key = cardOf(m);
    const app = apps.get(key) ?? { id: key, name: m.app ? m.name || m.app : m.device || m.id.slice(-8), logo: m.icon, instances: [] };
    app.instances.push({
      id: m.id,
      device: m.device,
      deviceKind: deviceKind(m),
      form: m.deployment === "phone" ? "android" : m.deployment || undefined,
      version: m.version || undefined,
      presence: m.online ? "connected" : "away",
      relay: m.relay,
      lastSeen: !m.online && m.lastSeen ? ago(m.lastSeen) : undefined,
      open: m.app ? {} : undefined,
      doubt: doubted(m, members),
      print: m.id.slice(-8),
    });
    apps.set(key, app);
  }
  return [...apps.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export function fortressPermissions(
  permissions: MemberPermission[],
  members: FortressMember[],
  t: (key: string, params?: Record<string, string>) => string,
): FortressPermission[] {
  const name = (id: string) => (id === SELF.id ? SELF.name : (members.find((m) => m.app === id)?.name ?? id));
  return permissions.map((p) => ({
    id: `${p.from} ${p.to} ${p.action}`,
    from: p.from,
    to: [p.to],
    on: p.on,
    title: p.action === "ui.open" ? t("fortress.uiOpen", { app: name(p.to) }) : p.action,
  }));
}

/** The permission an id of fortressPermissions names. */
export function permissionOf(id: string, on: boolean): MemberPermission {
  const [from = "", to = "", action = ""] = id.split(" ");
  return { from, to, action, on };
}

/** A member's status report and the end of its log, newest first, as the server hands them on. */
export type MemberStatus = {
  report: FortressStatusReport;
  log: FortressLogLine[];
};

/** The report and the log as GlimStone's status window shows them, the times in the reader's words. */
export function statusLines(status: MemberStatus, when: (at: string) => string): MemberStatus {
  const r = status.report;
  return {
    report: {
      state: r.state,
      needsYou: (r.needsYou ?? []).map((n) => ({ severity: n.severity, text: n.text })),
      work: (r.work ?? []).map((w) => (w.progress === undefined ? { title: w.title } : { title: w.title, progress: w.progress })),
      next: (r.next ?? []).map((n) => ({ title: n.title, at: when(n.at) })),
    },
    log: (status.log ?? []).map((l) => ({ at: when(l.at), level: l.level, text: l.text })),
  };
}
