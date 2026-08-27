// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// D-RichTextNodes' renderer: maps a richText node array (internal/content/application/
// blockvalidation.go's own shared shape, seeded onto paragraph/heading/quote/staff_card.bio/list by
// migrations/0022_content_richtext.sql) straight to React elements. There is no HTML string
// anywhere in this pipeline, so there is no HTML parser and no sanitizer to get wrong.
//
// A malformed or legacy (pre-M14.2, still-a-plain-string) value renders as nothing rather than
// crashing the page — the same defensive posture block-security.ts already takes for URLs.
import { safeUrl } from "@/lib/block-security";

interface TextMark {
  type: "bold" | "italic" | "link";
  href?: unknown;
}

interface TextNode {
  type: "text";
  text?: unknown;
  marks?: unknown;
}

interface ListItemNode {
  type: "listItem";
  content?: unknown;
}

interface ListNode {
  type: "list";
  style?: unknown;
  items?: unknown;
}

type RichTextNode = TextNode | ListNode | { type: unknown };

function renderTextNode(node: TextNode, key: number): React.ReactNode {
  const text = String(node.text ?? "");
  const marks = Array.isArray(node.marks) ? (node.marks as TextMark[]) : [];

  let content: React.ReactNode = text;
  if (marks.some((m) => m?.type === "bold")) content = <strong>{content}</strong>;
  if (marks.some((m) => m?.type === "italic")) content = <em>{content}</em>;

  const linkMark = marks.find((m) => m?.type === "link");
  if (linkMark) {
    const href = safeUrl(linkMark.href);
    if (href) {
      content = (
        <a href={href} className="underline">
          {content}
        </a>
      );
    }
  }

  return <span key={key}>{content}</span>;
}

function renderListNode(node: ListNode, key: number): React.ReactNode {
  const Tag = node.style === "ordered" ? "ol" : "ul";
  const items = Array.isArray(node.items) ? (node.items as ListItemNode[]) : [];
  return (
    <Tag key={key} className={Tag === "ol" ? "list-decimal pl-5" : "list-disc pl-5"}>
      {items.map((item, i) => (
        <li key={i}>
          <RichText nodes={item?.content} />
        </li>
      ))}
    </Tag>
  );
}

/** Renders a richText node array (D-RichTextNodes). Anything else — undefined, a legacy plain
 * string, a malformed value — renders as nothing. */
export function RichText({ nodes }: { nodes: unknown }) {
  if (!Array.isArray(nodes)) return null;
  return (
    <>
      {(nodes as RichTextNode[]).map((node, i) => {
        if (!node || typeof node !== "object") return null;
        switch (node.type) {
          case "text":
            return renderTextNode(node as TextNode, i);
          case "list":
            return renderListNode(node as ListNode, i);
          default:
            return null;
        }
      })}
    </>
  );
}
