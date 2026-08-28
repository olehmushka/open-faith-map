// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.8: session-local undo/redo over a form's serialized snapshot. Shaped like
// use-debounced-autosave.ts (same "poll the form's current state" technique, for the same reason —
// block-data-form.tsx's fields are each their own uncontrolled component, so reading the form
// directly is the one mechanism that covers every widget uniformly) but otherwise independent: its
// own poll/settle constants, no coupling to the autosave hook. When the polled snapshot changes and
// then holds steady for `settleMs`, the *previous* settled snapshot is pushed onto `past` and any
// `future` is cleared — one mechanism that covers add/delete/reorder (the snapshot changes instantly
// and is already stable by the next tick) and edit (the snapshot changes per keystroke/blur, but
// only settles into a stack entry once the user pauses).
//
// `limit` is a plain in-memory bound for a session-only array of small JSON snapshots — unrelated to
// M14.6's server-side content_document_revisions cap (also 50 by coincidence): that one is durable,
// cross-session, and reached only through Publish; this one lives only as long as the tab does and
// is never persisted.
"use client";

import { useCallback, useEffect, useRef, useState } from "react";

const DEFAULT_POLL_MS = 500;
const DEFAULT_SETTLE_MS = 600;
const DEFAULT_LIMIT = 50;

interface Snapshot<T> {
  raw: T;
  text: string;
}

export function useBlockHistory<T>(
  getSnapshot: () => T | null,
  options: { pollMs?: number; settleMs?: number; limit?: number } = {},
) {
  const pollMs = options.pollMs ?? DEFAULT_POLL_MS;
  const settleMs = options.settleMs ?? DEFAULT_SETTLE_MS;
  const limit = options.limit ?? DEFAULT_LIMIT;

  const [past, setPast] = useState<T[]>([]);
  const [future, setFuture] = useState<T[]>([]);
  const baselineRef = useRef<Snapshot<T> | null>(null);
  const pendingRef = useRef<{ text: string; since: number } | null>(null);
  const getSnapshotRef = useRef(getSnapshot);
  getSnapshotRef.current = getSnapshot;

  const serialize = useCallback((): Snapshot<T> | null => {
    const raw = getSnapshotRef.current();
    if (raw === null) return null;
    return { raw, text: JSON.stringify(raw) };
  }, []);

  useEffect(() => {
    const interval = setInterval(() => {
      const snapshot = serialize();
      if (snapshot === null) return;
      const baseline = baselineRef.current;
      // The baseline is latched from the first poll tick rather than a mount effect: some form
      // controls (Radix Select's hidden native <select> mirror, in particular) aren't necessarily
      // present in the DOM in the very first effect pass after mount, so a snapshot taken that early
      // can be missing fields — latching too eagerly would make page load itself look like an edit.
      if (!baseline) {
        baselineRef.current = snapshot;
        return;
      }
      if (snapshot.text === baseline.text) {
        pendingRef.current = null;
        return;
      }
      if (!pendingRef.current || pendingRef.current.text !== snapshot.text) {
        pendingRef.current = { text: snapshot.text, since: Date.now() };
        return;
      }
      if (Date.now() - pendingRef.current.since >= settleMs) {
        setPast((p) => {
          const next = [...p, baseline.raw];
          return next.length > limit ? next.slice(next.length - limit) : next;
        });
        setFuture([]);
        baselineRef.current = snapshot;
        pendingRef.current = null;
      }
    }, pollMs);
    return () => clearInterval(interval);
  }, [serialize, settleMs, pollMs, limit]);

  // Undo/redo must move the poll's own baseline synchronously, in the same call — otherwise the next
  // tick would see the restore itself as a fresh user edit and corrupt the stack (a redo becoming
  // unreachable, or a duplicate past entry).
  const undo = useCallback((): T | null => {
    if (past.length === 0) return null;
    const current = baselineRef.current;
    const previous = past[past.length - 1];
    setPast((p) => p.slice(0, -1));
    if (current) setFuture((f) => [...f, current.raw]);
    baselineRef.current = { raw: previous, text: JSON.stringify(previous) };
    pendingRef.current = null;
    return previous;
  }, [past]);

  const redo = useCallback((): T | null => {
    if (future.length === 0) return null;
    const current = baselineRef.current;
    const next = future[future.length - 1];
    setFuture((f) => f.slice(0, -1));
    if (current) setPast((p) => [...p, current.raw]);
    baselineRef.current = { raw: next, text: JSON.stringify(next) };
    pendingRef.current = null;
    return next;
  }, [future]);

  return { canUndo: past.length > 0, canRedo: future.length > 0, undo, redo };
}
