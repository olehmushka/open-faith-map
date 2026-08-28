// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useBlockHistory } from "./use-block-history";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

const OPTS = { pollMs: 100, settleMs: 200, limit: 3 };

describe("useBlockHistory", () => {
  it("does not push a stack entry while a value is still changing", async () => {
    let value = "a";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    value = "b";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100);
    });
    value = "c";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100);
    });

    expect(result.current.canUndo).toBe(false);
  });

  it("pushes a stack entry once the value settles, and undo restores the prior value", async () => {
    let value = "a";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    value = "b";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    expect(result.current.canUndo).toBe(true);

    let restored: string | null = null;
    act(() => {
      restored = result.current.undo();
    });
    expect(restored).toBe("a");
    expect(result.current.canUndo).toBe(false);
    expect(result.current.canRedo).toBe(true);
  });

  it("redo re-applies an undone change", async () => {
    let value = "a";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    value = "b";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });

    act(() => {
      result.current.undo();
    });

    let redone: string | null = null;
    act(() => {
      redone = result.current.redo();
    });
    expect(redone).toBe("b");
    expect(result.current.canRedo).toBe(false);
    expect(result.current.canUndo).toBe(true);
  });

  it("a new change after undo clears the redo stack", async () => {
    let value = "a";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    value = "b";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    act(() => {
      result.current.undo();
    });
    expect(result.current.canRedo).toBe(true);

    value = "c";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });

    expect(result.current.canRedo).toBe(false);
  });

  it("does not re-record undo's own restore as a new past entry", async () => {
    let value = "a";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    value = "b";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    expect(result.current.canUndo).toBe(true);

    // undo() moves the poll's baseline back to "a" synchronously. The caller (BlockListEditor, in
    // real usage) then writes "a" back into whatever getSnapshot reads — simulated here by setting
    // `value` back to "a" ourselves, exactly as the restore should observe it on the next tick.
    act(() => {
      result.current.undo();
    });
    value = "a";

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    // If the restore had been mistaken for a fresh edit, canUndo would have flipped back to true
    // (a spurious push of "b") and/or canRedo would have been cleared.
    expect(result.current.canUndo).toBe(false);
    expect(result.current.canRedo).toBe(true);
  });

  it("respects the limit option", async () => {
    let value = "0";
    const { result } = renderHook(() => useBlockHistory(() => value, OPTS));
    // The baseline latches on the first poll tick, not at mount (see use-block-history.ts's own
    // comment on why) — advance past that first tick before making any change, so the *true* initial
    // value becomes the baseline rather than being lost to a change that raced ahead of it.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(OPTS.pollMs);
    });

    for (let i = 1; i <= 5; i++) {
      value = String(i);
      await act(async () => {
        await vi.advanceTimersByTimeAsync(300);
      });
    }

    // limit: 3 — five settled changes should cap the stack at 3 undoable steps.
    let undone = 0;
    while (result.current.canUndo) {
      act(() => {
        result.current.undo();
      });
      undone++;
      if (undone > 10) break; // safety net against an infinite loop if the cap doesn't hold
    }
    expect(undone).toBe(3);
  });
});
