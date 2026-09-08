package render

import "testing"

func TestTitleColorsAreDistinctAndNonWhite(t *testing.T) {
	const factCount = 17
	if len(titleColors) < factCount {
		t.Fatalf("title palette has %d colors for %d facts", len(titleColors), factCount)
	}

	seen := make(map[[4]uint32]struct{}, len(titleColors))
	for i, color := range titleColors[:factCount] {
		r, g, b, a := color.RGBA()
		key := [4]uint32{r, g, b, a}
		if _, ok := seen[key]; ok {
			t.Fatalf("title color %d duplicates an earlier title color", i)
		}
		seen[key] = struct{}{}
		if r == 0xffff && g == 0xffff && b == 0xffff {
			t.Fatalf("title color %d is white", i)
		}
	}
}
