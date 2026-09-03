// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Renders the MVP block-type catalog (migrations/0002_content.sql's 13 seeded types, plus `list`
// added by migrations/0022_content_richtext.sql/M14.2 and `contact_form` added by
// migrations/0031_content_form_submissions.sql/M14.16) for the public per-congregation page. A
// plain server component — no interactivity needed to read a published page, except
// "contact_form", which delegates to a client component (ContactFormBlock) rather than needing
// this whole tree to be interactive. Unknown block types (a future catalog addition this renderer
// hasn't caught up with yet) fall through to a harmless no-op rather than crashing the page.
import { getTranslations } from "next-intl/server";

import { ContactFormBlock, type ContactFormActionState } from "@/components/contact-form-block";
import { isValidYoutubeVideoId, safeEmbedSrc, safeSocialEmbedUrl, safeUrl } from "@/lib/block-security";
import { ContentApiError, submitContactForm, type Block } from "@/lib/content";
import { RichText } from "@/lib/rich-text";

type BlocksT = Awaited<ReturnType<typeof getTranslations>>;

// siteId is optional because most callers (Post/Event feeds, nested "columns" blocks) never need
// it — only the "contact_form" case below reads it, to know which site's inbox a submission
// belongs to. Every other case ignores the prop entirely.
export async function Blocks({ blocks, siteId }: { blocks: Block[]; siteId?: string }) {
  const t = await getTranslations("Blocks");
  return (
    <div className="flex flex-col gap-4">
      {[...blocks]
        .sort((a, b) => a.position - b.position)
        .map((b) => <BlockView key={b.id} blockTypeCode={b.blockTypeCode} data={b.data} t={t} siteId={siteId} />)}
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
  siteId,
}: {
  blockTypeCode: string;
  data: unknown;
  t: BlocksT;
  siteId?: string;
}) {
  const data = (rawData ?? {}) as Record<string, unknown>;
  switch (blockTypeCode) {
    case "heading": {
      const level = Math.min(6, Math.max(1, Number(data.level) || 2));
      const Tag = `h${level}` as keyof React.JSX.IntrinsicElements;
      return (
        <Tag className="font-heading font-semibold">
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
                <BlockView key={j} blockTypeCode={nb.blockTypeCode} data={nb.data} t={t} siteId={siteId} />
              ))}
            </div>
          ))}
        </div>
      );
    }
    case "contact_form": {
      if (!siteId) return null;
      // A const copy, not the parameter itself: TypeScript doesn't carry a parameter's narrowed
      // type into a nested function's closure (it could in principle be reassigned before that
      // function runs), so submitContactForm below would otherwise see `string | undefined`.
      const contactFormSiteId: string = siteId;
      // M14.16, D-InAppInbox. An inline Server Action, same shape as site-page.tsx's own `report`
      // (closes over siteId from this render's props — no bind/lib/actions file needed since this
      // whole module is a Server Component). Honeypot/too-fast-submission handling lives entirely
      // server-side in application.Service.SubmitContactForm; this action just forwards the two
      // anti-spam fields the client component captured, and never tells the caller which case (if
      // either) fired.
      async function submitAction(_prevState: ContactFormActionState, formData: FormData): Promise<ContactFormActionState> {
        "use server";
        const name = String(formData.get("name") ?? "").trim();
        const email = String(formData.get("email") ?? "").trim();
        const message = String(formData.get("message") ?? "").trim();
        const honeypot = String(formData.get("website") ?? "");
        const formRenderedAt = String(formData.get("formRenderedAt") ?? new Date().toISOString());
        try {
          await submitContactForm(contactFormSiteId, {
            name: name || undefined,
            email: email || undefined,
            message,
            honeypot: honeypot || undefined,
            formRenderedAt,
          });
        } catch (e) {
          if (e instanceof ContentApiError) return { ok: false, error: e.errorName };
          throw e;
        }
        return { ok: true };
      }
      return (
        <ContactFormBlock
          heading={typeof data.heading === "string" ? data.heading : undefined}
          description={typeof data.description === "string" ? data.description : undefined}
          action={submitAction}
        />
      );
    }
    default:
      return null;
  }
}
