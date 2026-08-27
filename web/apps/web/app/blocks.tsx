// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Renders the MVP block-type catalog (migrations/0002_content.sql's 13 seeded types, plus `list`
// added by migrations/0022_content_richtext.sql/M14.2) for the public per-congregation page. A
// plain server component — no interactivity needed to read a published page. Unknown block types
// (a future catalog addition this renderer hasn't caught up with yet) fall through to a harmless
// no-op rather than crashing the page.
import { getTranslations } from "next-intl/server";

import { isValidYoutubeVideoId, safeEmbedSrc, safeSocialEmbedUrl, safeUrl } from "@/lib/block-security";
import type { Block } from "@/lib/content";
import { RichText } from "@/lib/rich-text";

type BlocksT = Awaited<ReturnType<typeof getTranslations>>;

export async function Blocks({ blocks }: { blocks: Block[] }) {
  const t = await getTranslations("Blocks");
  return (
    <div className="flex flex-col gap-4">
      {[...blocks]
        .sort((a, b) => a.position - b.position)
        .map((b) => <BlockView key={b.id} blockTypeCode={b.blockTypeCode} data={b.data} t={t} />)}
    </div>
  );
}

/** A nested block (inside a "columns" block) has only blockTypeCode/data, no id/position of its
 * own — its position is implicit in array order (json_schema has no position field for these). */
interface NestedBlock {
  blockTypeCode: string;
  data: unknown;
}

function BlockView({
  blockTypeCode,
  data: rawData,
  t,
}: {
  blockTypeCode: string;
  data: unknown;
  t: BlocksT;
}) {
  const data = (rawData ?? {}) as Record<string, unknown>;
  switch (blockTypeCode) {
    case "heading": {
      const level = Math.min(6, Math.max(1, Number(data.level) || 2));
      const Tag = `h${level}` as keyof React.JSX.IntrinsicElements;
      return (
        <Tag className="font-semibold">
          <RichText nodes={data.text} />
        </Tag>
      );
    }
    case "paragraph":
      return (
        <p>
          <RichText nodes={data.text} />
        </p>
      );
    case "list":
      return <RichText nodes={data.content} />;
    case "image": {
      const src = safeUrl(data.url);
      if (!src) return null;
      return (
        <figure>
          {/* eslint-disable-next-line @next/next/no-img-element -- external, unknown-dimension admin-authored URLs */}
          <img
            src={src}
            alt={String(data.alt ?? "")}
            className="max-w-full rounded"
            loading="lazy"
            referrerPolicy="no-referrer"
          />
          {data.caption ? <figcaption className="text-sm text-gray-500">{String(data.caption)}</figcaption> : null}
        </figure>
      );
    }
    case "gallery": {
      const images = Array.isArray(data.images) ? (data.images as { url: string; alt?: string }[]) : [];
      const safeImages = images
        .map((img) => ({ ...img, url: safeUrl(img.url) }))
        .filter((img): img is { url: string; alt?: string } => Boolean(img.url));
      return (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          {safeImages.map((img, i) => (
            // eslint-disable-next-line @next/next/no-img-element -- see "image" above
            <img
              key={i}
              src={img.url}
              alt={img.alt ?? ""}
              className="w-full rounded"
              loading="lazy"
              referrerPolicy="no-referrer"
            />
          ))}
        </div>
      );
    }
    case "youtube_embed": {
      const videoId = data.videoId;
      if (!isValidYoutubeVideoId(videoId)) return null;
      const src = safeEmbedSrc("youtube_embed", `https://www.youtube.com/embed/${videoId}`);
      if (!src) return null;
      return (
        <iframe
          className="aspect-video w-full rounded"
          src={src}
          title={String(data.title ?? t("youtubeVideoDefaultTitle"))}
          allowFullScreen
          sandbox="allow-scripts allow-popups allow-presentation"
          referrerPolicy="no-referrer"
        />
      );
    }
    case "social_embed": {
      const href = safeSocialEmbedUrl(data.url, data.platform);
      if (!href) return null;
      return (
        <a href={href} target="_blank" rel="noreferrer" className="underline">
          {String(data.platform ?? t("socialFallback"))} {t("socialPost")}
        </a>
      );
    }
    case "button": {
      const href = safeUrl(data.href);
      if (!href) return null;
      return (
        <a
          href={href}
          className={
            data.style === "secondary"
              ? "inline-block rounded border px-4 py-2"
              : "inline-block rounded bg-blue-600 px-4 py-2 text-white"
          }
        >
          {String(data.label ?? "")}
        </a>
      );
    }
    case "contact_info":
      return (
        <dl className="flex flex-col gap-1 text-sm">
          {data.address ? <div><dt className="inline font-medium">{t("address")} </dt><dd className="inline">{String(data.address)}</dd></div> : null}
          {data.phone ? <div><dt className="inline font-medium">{t("phone")} </dt><dd className="inline">{String(data.phone)}</dd></div> : null}
          {data.email ? <div><dt className="inline font-medium">{t("email")} </dt><dd className="inline">{String(data.email)}</dd></div> : null}
          {data.hours ? <div><dt className="inline font-medium">{t("hours")} </dt><dd className="inline">{String(data.hours)}</dd></div> : null}
        </dl>
      );
    case "map_embed":
      return (
        <a
          href={`https://www.openstreetmap.org/?mlat=${data.latitude}&mlon=${data.longitude}#map=${Number(data.zoom) || 15}/${data.latitude}/${data.longitude}`}
          target="_blank"
          rel="noreferrer"
          className="underline"
        >
          {t("viewOnMap")}
        </a>
      );
    case "divider":
      return data.style === "space" ? <div className="h-8" /> : <hr />;
    case "staff_card": {
      const photoUrl = safeUrl(data.photoUrl);
      return (
        <div className="flex items-center gap-3">
          {photoUrl ? (
            // eslint-disable-next-line @next/next/no-img-element -- see "image" above
            <img
              src={photoUrl}
              alt=""
              className="h-16 w-16 rounded-full object-cover"
              loading="lazy"
              referrerPolicy="no-referrer"
            />
          ) : null}
          <div>
            <p className="font-medium">{String(data.name ?? "")}</p>
            {data.title ? <p className="text-sm text-gray-500">{String(data.title)}</p> : null}
            {Array.isArray(data.bio) && data.bio.length > 0 ? (
              <p className="text-sm">
                <RichText nodes={data.bio} />
              </p>
            ) : null}
          </div>
        </div>
      );
    }
    case "quote":
      return (
        <blockquote className="border-l-4 pl-4 italic">
          <p>
            <RichText nodes={data.text} />
          </p>
          {data.attribution ? <cite className="block text-sm not-italic text-gray-500">— {String(data.attribution)}</cite> : null}
        </blockquote>
      );
    case "columns": {
      const columns = Array.isArray(data.columns) ? (data.columns as { blocks: NestedBlock[] }[]) : [];
      return (
        <div className="flex flex-col gap-4 sm:flex-row">
          {columns.map((col, i) => (
            <div key={i} className="flex-1">
              {(col.blocks ?? []).map((nb, j) => (
                <BlockView key={j} blockTypeCode={nb.blockTypeCode} data={nb.data} t={t} />
              ))}
            </div>
          ))}
        </div>
      );
    }
    default:
      return null;
  }
}
