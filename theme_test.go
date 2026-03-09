package wain

import "testing"

func TestDefaultDark(t *testing.T) {
	theme := DefaultDark()

	if theme.Background != RGB(30, 30, 46) {
		t.Errorf("Background = %v, want RGB(30, 30, 46)", theme.Background)
	}
	if theme.Foreground != RGB(205, 214, 244) {
		t.Errorf("Foreground = %v, want RGB(205, 214, 244)", theme.Foreground)
	}
	if theme.Accent != RGB(137, 180, 250) {
		t.Errorf("Accent = %v, want RGB(137, 180, 250)", theme.Accent)
	}
	if theme.Border != RGB(88, 91, 112) {
		t.Errorf("Border = %v, want RGB(88, 91, 112)", theme.Border)
	}
	if theme.FontSize != 14.0 {
		t.Errorf("FontSize = %v, want 14.0", theme.FontSize)
	}
	if theme.Padding != 8 {
		t.Errorf("Padding = %v, want 8", theme.Padding)
	}
	if theme.Gap != 6 {
		t.Errorf("Gap = %v, want 6", theme.Gap)
	}
	if theme.BorderWidth != 1 {
		t.Errorf("BorderWidth = %v, want 1", theme.BorderWidth)
	}
	if theme.BorderRadius != 4 {
		t.Errorf("BorderRadius = %v, want 4", theme.BorderRadius)
	}
	if theme.Scale != 1.0 {
		t.Errorf("Scale = %v, want 1.0", theme.Scale)
	}
}

func TestDefaultLight(t *testing.T) {
	theme := DefaultLight()

	if theme.Background != RGB(245, 245, 245) {
		t.Errorf("Background = %v, want RGB(245, 245, 245)", theme.Background)
	}
	if theme.Foreground != RGB(30, 30, 30) {
		t.Errorf("Foreground = %v, want RGB(30, 30, 30)", theme.Foreground)
	}
	if theme.Accent != RGB(74, 144, 226) {
		t.Errorf("Accent = %v, want RGB(74, 144, 226)", theme.Accent)
	}
	if theme.Border != RGB(200, 200, 200) {
		t.Errorf("Border = %v, want RGB(200, 200, 200)", theme.Border)
	}
	if theme.FontSize != 14.0 {
		t.Errorf("FontSize = %v, want 14.0", theme.FontSize)
	}
}

func TestHighContrast(t *testing.T) {
	theme := HighContrast()

	if theme.Background != RGB(0, 0, 0) {
		t.Errorf("Background = %v, want RGB(0, 0, 0)", theme.Background)
	}
	if theme.Foreground != RGB(255, 255, 255) {
		t.Errorf("Foreground = %v, want RGB(255, 255, 255)", theme.Foreground)
	}
	if theme.Accent != RGB(255, 255, 0) {
		t.Errorf("Accent = %v, want RGB(255, 255, 0)", theme.Accent)
	}
	if theme.Border != RGB(255, 255, 255) {
		t.Errorf("Border = %v, want RGB(255, 255, 255)", theme.Border)
	}
	if theme.FontSize != 16.0 {
		t.Errorf("FontSize = %v, want 16.0", theme.FontSize)
	}
	if theme.Padding != 10 {
		t.Errorf("Padding = %v, want 10", theme.Padding)
	}
	if theme.BorderWidth != 2 {
		t.Errorf("BorderWidth = %v, want 2", theme.BorderWidth)
	}
	if theme.BorderRadius != 0 {
		t.Errorf("BorderRadius = %v, want 0", theme.BorderRadius)
	}
}

func TestStyleOverrideApply(t *testing.T) {
	base := DefaultDark()

	// Test empty override
	override := StyleOverride{}
	result := override.applyToTheme(base)
	if result != base {
		t.Errorf("Empty override changed theme")
	}

	// Test background override
	newBg := RGB(100, 100, 100)
	override = StyleOverride{Background: &newBg}
	result = override.applyToTheme(base)
	if result.Background != newBg {
		t.Errorf("Background = %v, want %v", result.Background, newBg)
	}
	if result.Foreground != base.Foreground {
		t.Errorf("Foreground changed unexpectedly")
	}

	// Test multiple overrides
	newFg := RGB(200, 200, 200)
	fontSize := 18.0
	padding := 12
	override = StyleOverride{
		Foreground: &newFg,
		FontSize:   &fontSize,
		Padding:    &padding,
	}
	result = override.applyToTheme(base)
	if result.Foreground != newFg {
		t.Errorf("Foreground = %v, want %v", result.Foreground, newFg)
	}
	if result.FontSize != fontSize {
		t.Errorf("FontSize = %v, want %v", result.FontSize, fontSize)
	}
	if result.Padding != padding {
		t.Errorf("Padding = %v, want %v", result.Padding, padding)
	}
	if result.Background != base.Background {
		t.Errorf("Background changed unexpectedly")
	}
}

func TestStyleOverrideAllFields(t *testing.T) {
	base := DefaultDark()

	bg := RGB(1, 1, 1)
	fg := RGB(2, 2, 2)
	accent := RGB(3, 3, 3)
	border := RGB(4, 4, 4)
	fontSize := 20.0
	padding := 15
	gap := 10
	borderWidth := 3
	borderRadius := 8

	override := StyleOverride{
		Background:   &bg,
		Foreground:   &fg,
		Accent:       &accent,
		Border:       &border,
		FontSize:     &fontSize,
		Padding:      &padding,
		Gap:          &gap,
		BorderWidth:  &borderWidth,
		BorderRadius: &borderRadius,
	}

	result := override.applyToTheme(base)

	if result.Background != bg {
		t.Errorf("Background = %v, want %v", result.Background, bg)
	}
	if result.Foreground != fg {
		t.Errorf("Foreground = %v, want %v", result.Foreground, fg)
	}
	if result.Accent != accent {
		t.Errorf("Accent = %v, want %v", result.Accent, accent)
	}
	if result.Border != border {
		t.Errorf("Border = %v, want %v", result.Border, border)
	}
	if result.FontSize != fontSize {
		t.Errorf("FontSize = %v, want %v", result.FontSize, fontSize)
	}
	if result.Padding != padding {
		t.Errorf("Padding = %v, want %v", result.Padding, padding)
	}
	if result.Gap != gap {
		t.Errorf("Gap = %v, want %v", result.Gap, gap)
	}
	if result.BorderWidth != borderWidth {
		t.Errorf("BorderWidth = %v, want %v", result.BorderWidth, borderWidth)
	}
	if result.BorderRadius != borderRadius {
		t.Errorf("BorderRadius = %v, want %v", result.BorderRadius, borderRadius)
	}
}

func TestThemeScaleCustomization(t *testing.T) {
	theme := DefaultDark()
	theme.Scale = 2.0

	if theme.Scale != 2.0 {
		t.Errorf("Scale = %v, want 2.0", theme.Scale)
	}
}

func TestThemeImmutability(t *testing.T) {
	theme1 := DefaultDark()
	theme2 := DefaultDark()

	// Modifying theme1 should not affect theme2
	theme1.Background = RGB(255, 0, 0)
	if theme2.Background == RGB(255, 0, 0) {
		t.Errorf("DefaultDark() returns shared mutable state")
	}
}
