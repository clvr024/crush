package list

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachedOffsetReusesHeightMeasurements(t *testing.T) {
	t.Parallel()

	l := NewList()
	l.SetSize(80, 20)
	l.SetGap(1)
	for i := range 1000 {
		l.AppendItems(newMultiLineItem(strconv.Itoa(i), 1))
	}
	callbacks := 0
	l.RegisterRenderCallback(func(_, _ int, item Item) Item {
		callbacks++
		return item
	})
	require.Equal(t, 1999, l.TotalHeight())
	callbacks = 0

	for _, idx := range []int{500, 999, 750, 1, 0} {
		l.ScrollToIndex(idx)
		require.Equal(t, 2*idx, l.CachedOffset())
	}
	require.Zero(t, callbacks, "cached offsets must not revisit offscreen items")
}

func TestCachedOffsetRefreshesLayoutMeasurements(t *testing.T) {
	t.Parallel()

	a := newMultiLineItem("a", 2)
	b := newMultiLineItem("b", 3)
	c := newMultiLineItem("c", 1)
	l := NewList(a, b, c)
	l.SetSize(80, 2)
	l.SetGap(1)
	l.ScrollToIndex(2)
	check := func(want int) {
		t.Helper()
		require.Equal(t, want, l.CachedOffset())
		l.TotalHeight()
		require.Equal(t, want, l.CachedOffset())
	}
	check(7)

	a.height = 4
	a.Bump()
	l.Invalidate(a)
	check(9)

	a.height = 1
	a.Bump()
	l.Prewarm(0, 1)
	check(6)

	l.SetGap(2)
	check(8)
	l.SetSize(40, 2)
	check(8)
	l.PrependItems(newMultiLineItem("intro", 2))
	check(12)
	l.RemoveItem(0)
	check(8)
	l.SetItems(c, a, b)
	check(6)

	l.ScrollToIndex(2)
	l.ScrollBy(1)
	check(7)
	l.SetItems()
	check(0)
}

func TestCachedOffsetPreservesLazyRendering(t *testing.T) {
	t.Parallel()

	l := NewList(newMultiLineItem("a", 1), newMultiLineItem("b", 1), newMultiLineItem("c", 1))
	l.SetSize(80, 1)
	l.SetGap(1)
	l.ScrollToIndex(1)
	visited := []int{}
	l.RegisterRenderCallback(func(idx, _ int, item Item) Item {
		visited = append(visited, idx)
		return item
	})

	require.Equal(t, 2, l.CachedOffset())
	require.Equal(t, []int{0}, visited, "a cold offset must not render the entire list")
	require.False(t, l.totalHeightValid)
}

func BenchmarkCachedOffset(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			l := NewList()
			l.SetSize(80, 20)
			l.SetGap(1)
			for i := range n {
				l.AppendItems(newMultiLineItem(strconv.Itoa(i), 1))
			}
			l.ScrollToBottom()
			l.TotalHeight()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				l.CachedOffset()
			}
		})
	}
}
