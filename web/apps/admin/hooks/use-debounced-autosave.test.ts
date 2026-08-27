// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDebouncedAutosave } from "./use-debounced-autosave";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("useDebouncedAutosave", () => {
  it("does not save while nothing has changed", async () => {
    const save = vi.fn().mockResolvedValue({ ok: true });
    renderHook(() => useDebouncedAutosave(() => "same", save, { debounceMs: 10000, pollMs: 2000 }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(30000);
    });

    expect(save).not.toHaveBeenCalled();
  });

  it("saves once, debounceMs after the last detected change, even across several edits", async () => {
    const save = vi.fn().mockResolvedValue({ ok: true });
    let value = "initial";
    const { result } = renderHook(() =>
      useDebouncedAutosave(() => value, save, { debounceMs: 10000, pollMs: 2000 }),
    );

    // Establish baseline (the effect that runs on mount).
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    // Three edits, each well inside the debounce window of the last.
    value = "edit 1";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(result.current.status).toBe("unsaved");

    value = "edit 2";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(4000);
    });
    value = "edit 3 (final)";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(4000);
    });
    expect(save).not.toHaveBeenCalled();

    // Now let the debounce window elapse with no further edits.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(save).toHaveBeenCalledTimes(1);
    expect(save).toHaveBeenCalledWith("edit 3 (final)");
    expect(result.current.status).toBe("saved");
  });

  it("flush() saves immediately without waiting for the debounce window", async () => {
    const save = vi.fn().mockResolvedValue({ ok: true });
    let value = "initial";
    const { result } = renderHook(() =>
      useDebouncedAutosave(() => value, save, { debounceMs: 10000, pollMs: 2000 }),
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    value = "changed";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(result.current.status).toBe("unsaved");

    await act(async () => {
      result.current.flush();
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(save).toHaveBeenCalledTimes(1);
    expect(save).toHaveBeenCalledWith("changed");
  });

  it("surfaces a failed save as status \"error\"", async () => {
    const save = vi.fn().mockResolvedValue({ ok: false });
    let value = "initial";
    const { result } = renderHook(() =>
      useDebouncedAutosave(() => value, save, { debounceMs: 10000, pollMs: 2000 }),
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    value = "changed";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(12000);
    });

    expect(save).toHaveBeenCalledTimes(1);
    expect(result.current.status).toBe("error");
  });
});
