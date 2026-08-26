// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useCallback, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useTranslations } from "next-intl";

import { Link, useRouter } from "@/i18n/navigation";
import { UnitStatusBadge } from "@/components/status-badge";
import { cn } from "@/lib/utils";
import type { Unit } from "@/lib/core";

const CHILDREN_LIMIT = 50;

type VisibleNode = { unit: Unit; depth: number; parentId: string | null };

// M12.7's admin hierarchy tree: a real WAI-ARIA treeview (role="tree"/"treeitem", roving tabindex,
// arrow-key navigation) over a flattened, aria-level-indented list rather than nested DOM groups —
// an accepted simplification of the authoring-practice pattern, and the only shape that keeps a
// single component in charge of roving focus across every expanded level at once. Every level below
// the root loads lazily, one hop at a time, via fetchChildren (a server action wrapping M12.7's
// unitChildren) — the root's own first level arrives pre-loaded from the server component that
// renders this, so opening the tree needs no round trip.
export function UnitTree({
  root,
  rootChildren,
  fetchChildren,
}: {
  root: Unit;
  rootChildren: Unit[];
  fetchChildren: (unitId: string) => Promise<Unit[]>;
}) {
  const t = useTranslations("SuperAdminUnitTreePage");
  const router = useRouter();
  const containerRef = useRef<HTMLDivElement>(null);

  const [childrenOf, setChildrenOf] = useState<Record<string, Unit[] | undefined>>({
    [root.id]: rootChildren,
  });
  const [expanded, setExpanded] = useState<Set<string>>(new Set([root.id]));
  const [loading, setLoading] = useState<Set<string>>(new Set());
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [focusedId, setFocusedId] = useState(root.id);

  const unitsById = useMemo(() => {
    const map: Record<string, Unit> = { [root.id]: root };
    for (const kids of Object.values(childrenOf)) {
      for (const u of kids ?? []) map[u.id] = u;
    }
    return map;
  }, [root, childrenOf]);

  const visible = useMemo(() => {
    const out: VisibleNode[] = [];
    function walk(id: string, depth: number, parentId: string | null) {
      const unit = unitsById[id];
      if (!unit) return;
      out.push({ unit, depth, parentId });
      if (expanded.has(id)) {
        for (const child of childrenOf[id] ?? []) {
          walk(child.id, depth + 1, id);
        }
      }
    }
    walk(root.id, 0, null);
    return out;
  }, [root.id, unitsById, childrenOf, expanded]);

  const loadChildren = useCallback(
    (unitId: string) => {
      if (childrenOf[unitId] !== undefined) return;
      setLoading((prev) => new Set(prev).add(unitId));
      fetchChildren(unitId)
        .then((kids) => {
          setChildrenOf((prev) => ({ ...prev, [unitId]: kids }));
        })
        .catch(() => {
          setErrors((prev) => ({ ...prev, [unitId]: t("loadError") }));
        })
        .finally(() => {
          setLoading((prev) => {
            const next = new Set(prev);
            next.delete(unitId);
            return next;
          });
        });
    },
    [childrenOf, fetchChildren, t],
  );

  const focusNode = useCallback((unitId: string) => {
    setFocusedId(unitId);
    requestAnimationFrame(() => {
      containerRef.current?.querySelector<HTMLElement>(`[data-unit-id="${unitId}"]`)?.focus();
    });
  }, []);

  function toggle(unitId: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(unitId)) {
        next.delete(unitId);
      } else {
        next.add(unitId);
        loadChildren(unitId);
      }
      return next;
    });
  }

  function handleKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const idx = visible.findIndex((n) => n.unit.id === focusedId);
    if (idx === -1) return;
    const current = visible[idx];
    const kids = childrenOf[current.unit.id];
    const isLeaf = kids !== undefined && kids.length === 0;

    switch (e.key) {
      case "ArrowDown": {
        e.preventDefault();
        const next = visible[idx + 1];
        if (next) focusNode(next.unit.id);
        break;
      }
      case "ArrowUp": {
        e.preventDefault();
        const prev = visible[idx - 1];
        if (prev) focusNode(prev.unit.id);
        break;
      }
      case "ArrowRight": {
        e.preventDefault();
        if (isLeaf) break;
        if (!expanded.has(current.unit.id)) {
          toggle(current.unit.id);
        } else {
          const first = visible[idx + 1];
          if (first && first.parentId === current.unit.id) focusNode(first.unit.id);
        }
        break;
      }
      case "ArrowLeft": {
        e.preventDefault();
        if (expanded.has(current.unit.id) && !isLeaf) {
          toggle(current.unit.id);
        } else if (current.parentId) {
          focusNode(current.parentId);
        }
        break;
      }
      case "Home": {
        e.preventDefault();
        if (visible[0]) focusNode(visible[0].unit.id);
        break;
      }
      case "End": {
        e.preventDefault();
        const last = visible[visible.length - 1];
        if (last) focusNode(last.unit.id);
        break;
      }
      case "Enter": {
        e.preventDefault();
        router.push(`/admin/units/${current.unit.id}`);
        break;
      }
      default:
        break;
    }
  }

  return (
    <div
      ref={containerRef}
      role="tree"
      aria-label={t("heading")}
      className="flex flex-col rounded-md border p-2 text-sm"
      onKeyDown={handleKeyDown}
    >
      {visible.map(({ unit, depth }) => {
        const kids = childrenOf[unit.id];
        const isExpanded = expanded.has(unit.id);
        const isLoading = loading.has(unit.id);
        const isLeaf = kids !== undefined && kids.length === 0;
        return (
          <div
            key={unit.id}
            role="treeitem"
            data-unit-id={unit.id}
            aria-expanded={isLeaf ? undefined : isExpanded}
            aria-level={depth + 1}
            tabIndex={focusedId === unit.id ? 0 : -1}
            onFocus={() => setFocusedId(unit.id)}
            className="flex items-center gap-2 rounded px-1 py-1 outline-none focus-visible:ring-2 focus-visible:ring-ring"
            style={{ paddingLeft: `${depth * 1.25}rem` }}
          >
            <button
              type="button"
              tabIndex={-1}
              onClick={() => {
                if (!isLeaf) toggle(unit.id);
              }}
              aria-label={isExpanded ? t("collapse", { name: unit.name }) : t("expand", { name: unit.name })}
              className={cn("flex size-4 shrink-0 items-center justify-center", isLeaf && "invisible")}
            >
              {isExpanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
            </button>
            <Link href={`/admin/units/${unit.id}`} tabIndex={-1} className="flex flex-1 items-center gap-2 hover:underline">
              <span>{unit.name}</span>
              {unit.code && <span className="text-xs text-muted-foreground">{unit.code}</span>}
              <UnitStatusBadge status={unit.state} />
            </Link>
            {isLoading && <span className="text-xs text-muted-foreground">…</span>}
            {errors[unit.id] && <span className="text-xs text-destructive">{errors[unit.id]}</span>}
            {kids && kids.length === CHILDREN_LIMIT && (
              <span className="text-xs text-muted-foreground">{t("truncatedHint")}</span>
            )}
          </div>
        );
      })}
    </div>
  );
}
