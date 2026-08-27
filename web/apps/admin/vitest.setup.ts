// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import "@testing-library/jest-dom/vitest";

// jsdom has no ResizeObserver; @dnd-kit/core references it when measuring droppable/sortable nodes
// even outside an actual drag gesture, so mounting DndContext/useSortable needs this polyfilled.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver ??= ResizeObserverStub;

// jsdom doesn't implement scrollIntoView (no real layout engine); cmdk calls it when the
// keyboard-selected item changes.
Element.prototype.scrollIntoView ??= () => {};
