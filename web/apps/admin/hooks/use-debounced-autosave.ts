// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.6: autosave a draft on a debounce, with a visible saved/unsaved indicator — no precedent
// exists anywhere in this codebase yet (no debounce hook, no "unsaved changes" indicator).
//
// Polls the form's current serialized state rather than wiring change events through every nested
// field, deliberately: the block editor's per-block data fields (block-data-form.tsx) are each
// their own uncontrolled component writing into a hidden `data` input, and several of its widgets
// (Select, the array/block-list widgets) never fire a bubbling native input/change event a
// form-level listener could catch — reading the form's actual current state directly is the one
// mechanism that covers every widget uniformly.
"use client";

import { useCallback, useEffect, useRef, useState, useTransition } from "react";

export type AutosaveStatus = "idle" | "unsaved" | "saving" | "saved" | "error";

export interface AutosaveResult {
  ok: boolean;
}

const DEFAULT_DEBOUNCE_MS = 10000;
const DEFAULT_POLL_MS = 2000;

export function useDebouncedAutosave<T>(
  getSnapshot: () => T | null,
  save: (payload: T) => Promise<AutosaveResult>,
  options: { debounceMs?: number; pollMs?: number } = {},
) {
  const debounceMs = options.debounceMs ?? DEFAULT_DEBOUNCE_MS;
  const pollMs = options.pollMs ?? DEFAULT_POLL_MS;

  const [status, setStatus] = useState<AutosaveStatus>("idle");
  const [, startTransition] = useTransition();
  const lastSavedRef = useRef<string | null>(null);
  const lastSeenRef = useRef<string | null>(null);
  const lastChangeAtRef = useRef<number>(0);
  const savingRef = useRef(false);
  const getSnapshotRef = useRef(getSnapshot);
  const saveRef = useRef(save);
  getSnapshotRef.current = getSnapshot;
  saveRef.current = save;

  const serialize = useCallback((): { raw: T; text: string } | null => {
    const raw = getSnapshotRef.current();
    if (raw === null) return null;
    return { raw, text: JSON.stringify(raw) };
  }, []);

  // Establishes the initial baseline once, so the first poll tick doesn't treat the freshly-loaded
  // document as an unsaved change.
  useEffect(() => {
    const initial = serialize();
    lastSavedRef.current = initial?.text ?? null;
    lastSeenRef.current = initial?.text ?? null;
  }, [serialize]);

  const flush = useCallback(() => {
    if (savingRef.current) return;
    const snapshot = serialize();
    if (snapshot === null || snapshot.text === lastSavedRef.current) {
      setStatus((s) => (s === "unsaved" ? "saved" : s));
      return;
    }
    savingRef.current = true;
    setStatus("saving");
    startTransition(async () => {
      const result = await saveRef.current(snapshot.raw);
      savingRef.current = false;
      if (result.ok) {
        lastSavedRef.current = snapshot.text;
        setStatus("saved");
      } else {
        setStatus("error");
      }
    });
  }, [serialize]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (savingRef.current) return;
      const snapshot = serialize();
      if (snapshot === null) return;
      if (snapshot.text !== lastSeenRef.current) {
        lastSeenRef.current = snapshot.text;
        lastChangeAtRef.current = Date.now();
        setStatus("unsaved");
        return;
      }
      if (snapshot.text !== lastSavedRef.current && Date.now() - lastChangeAtRef.current >= debounceMs) {
        flush();
      }
    }, pollMs);
    return () => clearInterval(interval);
  }, [serialize, flush, debounceMs, pollMs]);

  // Best-effort: fire a final save on unmount (e.g. client-side navigation away) so a fast
  // edit-then-leave isn't left stranded until the next poll tick would have caught it.
  useEffect(() => {
    return () => {
      const snapshot = serialize();
      if (snapshot !== null && snapshot.text !== lastSavedRef.current && !savingRef.current) {
        void saveRef.current(snapshot.raw);
      }
    };
    // Intentionally empty deps — this cleanup must only ever run on unmount, using the latest refs.
  }, []);

  return { status, flush };
}
