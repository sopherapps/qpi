import React, { useMemo, useState } from "react";
import type { Driver, EventRow } from "@/types";
import { ReadingsChart } from "./elements/ReadingsChart";

interface MonitoringTabProps {
  events: EventRow[];
  drivers: Driver[];
}

interface ChannelSeries {
  channel: string;
  unit?: string;
  points: { ts: number; value: number }[];
}

/** Live view of monitoring drivers (RFC 0001 §10): reads the `events` trace
 * log — already loaded and kept fresh by realtime in App.tsx — filtered to
 * CryostatReading, and charts each channel's readings over time. */
export const MonitoringTab: React.FC<MonitoringTabProps> = ({
  events,
  drivers,
}) => {
  const monitors = useMemo(
    () => drivers.filter((d) => d.kind === "bluefors_gen1"),
    [drivers],
  );

  const [selectedDriver, setSelectedDriver] = useState<string>("all");

  const readings = useMemo(() => {
    const filtered = events.filter(
      (e) =>
        e.type === "CryostatReading" &&
        (selectedDriver === "all" || e.driver === selectedDriver),
    );

    const series = new Map<string, ChannelSeries>();
    // events are loaded newest-first; walk in chronological order so each
    // channel's points end up sorted left-to-right for the chart.
    for (const event of [...filtered].reverse()) {
      const ts = new Date(event.ts).getTime();
      if (Number.isNaN(ts)) continue;

      for (const [channel, reading] of Object.entries(
        event.payload?.readings ?? {},
      )) {
        if (reading.value === null || reading.value === undefined) continue;
        if (!series.has(channel)) {
          series.set(channel, { channel, unit: reading.unit, points: [] });
        }
        series.get(channel)!.points.push({ ts, value: reading.value });
      }
    }
    return Array.from(series.values()).sort((a, b) =>
      a.channel.localeCompare(b.channel),
    );
  }, [events, selectedDriver]);

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-geist text-gray-900 dark:text-white">
            Monitoring
          </h1>
          <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
            Live channel readings from cryostat monitor drivers.
          </p>
        </div>
        {monitors.length > 0 && (
          <select
            data-testid="monitoring-driver-select"
            value={selectedDriver}
            onChange={(e) => setSelectedDriver(e.target.value)}
            className="bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 text-gray-900 dark:text-white rounded px-3 py-2 text-sm focus:outline-none focus:border-zinc-500 transition-colors"
          >
            <option value="all">All monitors</option>
            {monitors.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name} ({d.status})
              </option>
            ))}
          </select>
        )}
      </div>

      {monitors.length === 0 ? (
        <div className="text-sm text-gray-500 dark:text-zinc-400 border border-dashed border-gray-200 dark:border-zinc-800 rounded-lg p-8 text-center">
          No cryostat monitor drivers registered yet. Register one from the
          Drivers page with kind <code>bluefors_gen1</code>.
        </div>
      ) : readings.length === 0 ? (
        <div className="text-sm text-gray-500 dark:text-zinc-400 border border-dashed border-gray-200 dark:border-zinc-800 rounded-lg p-8 text-center">
          No readings yet. Once a monitor driver connects and reports, its
          channels will chart here in real time.
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {readings.map((s) => (
            <ReadingsChart
              key={s.channel}
              channel={s.channel}
              unit={s.unit}
              points={s.points}
            />
          ))}
        </div>
      )}
    </div>
  );
};
