// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { DAY_KEYS } from "@/components/site-page";
import type { SiteChrome } from "@/lib/content";

const SOCIAL_LINKS: { key: keyof SiteChrome["socialLinks"]; label: string }[] = [
  { key: "facebook", label: "Facebook" },
  { key: "instagram", label: "Instagram" },
  { key: "youtube", label: "YouTube" },
  { key: "twitter", label: "Twitter" },
  { key: "website", label: "Website" },
];

function formatScheduleTime(startTime?: string | null, endTime?: string | null): string | null {
  if (!startTime) return null;
  return endTime ? `${startTime}–${endTime}` : startTime;
}

// M14.11: the tenant site's footer — address and service schedule read live from religion_sites/
// religion_service_schedules (SiteChrome composes them server-side; this component only renders
// what it's handed, never fetches religion data itself), plus social links, both content_sites'
// own settings.
export async function SiteFooter({ chrome }: { chrome: SiteChrome }) {
  const t = await getTranslations("CongregationPage");
  const tm = await getTranslations("DiscoveryMap");

  const socialLinks = SOCIAL_LINKS.filter(({ key }) => chrome.socialLinks[key]);
  const hasSchedules = chrome.schedules.length > 0;

  if (!chrome.address && !hasSchedules && socialLinks.length === 0) {
    return null;
  }

  return (
    <footer className="border-t">
      <div className="mx-auto flex max-w-3xl flex-col gap-4 px-6 py-8 text-sm">
        {chrome.address ? <p className="text-muted-foreground">{chrome.address}</p> : null}

        {hasSchedules ? (
          <div className="flex flex-col gap-1">
            <h2 className="font-semibold">{t("serviceTimes")}</h2>
            <ul className="flex flex-col gap-0.5">
              {chrome.schedules.map((schedule, i) => {
                const time = formatScheduleTime(schedule.startTime, schedule.endTime);
                const day = schedule.dayOfWeek != null ? tm(DAY_KEYS[schedule.dayOfWeek]) : null;
                return (
                  <li key={i} className="text-muted-foreground">
                    {[day, time, schedule.language].filter(Boolean).join(" · ")}
                  </li>
                );
              })}
            </ul>
          </div>
        ) : null}

        {socialLinks.length > 0 ? (
          <ul className="flex flex-wrap gap-4">
            {socialLinks.map(({ key, label }) => (
              <li key={key}>
                <a
                  href={chrome.socialLinks[key] as string}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:underline"
                >
                  {label}
                </a>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </footer>
  );
}
