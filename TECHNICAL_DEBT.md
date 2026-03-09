# Technical Debt Tracking

This file tracks TODO items in the codebase. Each TODO comment references an item here using the format `// TODO(TD-N): Description` where N is the item number below.

## Active Items

### TD-1: Add placeholder support to TextInput widget
**File:** `concretewidgets.go:248`  
**Priority:** Medium  
**Description:** The public `TextInput.SetPlaceholder()` method accepts a placeholder string but doesn't pass it to the internal widget implementation. Need to extend `internal/ui/widgets` TextInput to support placeholder rendering (grayed-out text when input is empty).  
**Impact:** User-facing API exists but feature is non-functional  
**Effort:** ~1-2 hours (add placeholder field to internal widget, render in Draw method with dimmed color)  
**Related:** None  

### TD-2: Implement proper child management for ScrollView
**File:** `concretewidgets.go:361`  
**Priority:** Medium  
**Description:** The public `ScrollView.Add(child PublicWidget)` method is a stub. Need to extend `internal/ui/widgets.ScrollContainer` to accept PublicWidget children (currently only works with internal widgets). Requires bridging PublicWidget interface to internal widget representation.  
**Impact:** ScrollView widget cannot hold child widgets, limiting its utility  
**Effort:** ~2-3 hours (add child container to ScrollContainer, implement layout pass for children)  
**Related:** None  

### TD-3: Theme system integration for Panel widget
**File:** `layout.go:388`  
**Priority:** Low  
**Description:** Panel widget currently hardcodes `DefaultDark()` theme instead of reading from `App.theme`. Once the App-level theme system is implemented, Panel should respect the global theme setting while allowing per-widget style overrides.  
**Impact:** Panels ignore app-wide theme, always use dark theme  
**Effort:** ~30 minutes (change to `app.theme.Panel()` once theme system exists)  
**Related:** Blocked by App.theme field implementation (not yet designed)  

### TD-4: Full Wayland event reading and dispatch
**File:** `app.go:1471`  
**Priority:** High  
**Description:** The Wayland display server currently only sends requests but doesn't read events from the compositor. This prevents handling user input (keyboard, mouse), window state changes, or other compositor events. Need to:
  1. Implement wire protocol event parser in `internal/wayland/wire`
  2. Add event dispatch loop to `App.pollEvents()` for Wayland path
  3. Map Wayland events (wl_keyboard, wl_pointer, etc.) to wain Event types
  4. Test with real Wayland compositor (Weston, Sway)  
**Impact:** Wayland apps cannot receive user input or respond to compositor events  
**Effort:** ~8-12 hours (wire parsing 2h, dispatch 3h, event mapping 3h, testing 4h)  
**Related:** ROADMAP Phase 9.2 "Wayland Event Reading" — currently incomplete  

## Completed Items

(None yet)

---

**Last Updated:** 2026-03-09  
**Total Active Items:** 4  
**Total Completed Items:** 0
