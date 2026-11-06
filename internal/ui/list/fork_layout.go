package list

// GroupedItem can omit the gap before a related item.
type GroupedItem interface {
	GroupedWith(next Item) bool
}

// GapAfter returns the spacing after an item, accounting for item grouping.
func (l *List) GapAfter(idx int) int {
	if idx < 0 || idx >= len(l.items) {
		return 0
	}
	if idx+1 < len(l.items) {
		if item, ok := l.items[idx].(GroupedItem); ok && item.GroupedWith(l.items[idx+1]) {
			return 0
		}
	}
	return max(0, l.gap)
}

func (l *List) cacheItemOffset(idx, offset int) {
	if len(l.itemOffsets) != len(l.items)+1 {
		l.itemOffsets = make([]int, len(l.items)+1)
	}
	l.itemOffsets[idx] = offset
}

// CachedOffset reuses the starting lines measured by TotalHeight. If those
// measurements are stale, it falls back to Offset without measuring the
// entire list. Callers must refresh heights after content changes before
// using the cached result, as they do for TotalHeight itself.
func (l *List) CachedOffset() int {
	if l.offsetIdx <= 0 {
		return l.offsetLine
	}
	if !l.totalHeightValid || len(l.itemOffsets) != len(l.items)+1 {
		return l.Offset()
	}
	return l.itemOffsets[min(l.offsetIdx, len(l.items))] + l.offsetLine
}
