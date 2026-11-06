package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type groupedListItem struct {
	*multiLineItem
}

func (g *groupedListItem) GroupedWith(next Item) bool {
	_, ok := next.(*groupedListItem)
	return ok
}

func TestGroupedItemsLayout(t *testing.T) {
	t.Parallel()

	intro := newMultiLineItem("intro", 1)
	a := &groupedListItem{newMultiLineItem("a", 1)}
	b := &groupedListItem{newMultiLineItem("b", 1)}
	c := &groupedListItem{newMultiLineItem("c", 1)}
	end := newMultiLineItem("end", 1)
	l := NewList(intro, a, b, c, end)
	l.SetGap(1)
	l.SetSize(40, 7)

	require.Equal(t, "intro:0\n\na:0\nb:0\nc:0\n\nend:0", l.Render())
	require.Equal(t, 7, l.TotalHeight())
	require.False(t, l.Overflows(7))
	require.True(t, l.Overflows(6))
	require.True(t, l.AtBottom())

	for y, want := range []int{0, -1, 1, 2, 3, -1, 4} {
		idx, _ := l.ItemIndexAtPosition(0, y)
		require.Equal(t, want, idx, "row %d", y)
	}

	l.SetSize(40, 4)
	require.False(t, l.AtBottom())
	l.ScrollToIndex(1)
	require.Equal(t, 2, l.Offset())
	require.Equal(t, "a:0\nb:0\nc:0\n", l.Render())
	start, endIdx := l.VisibleItemIndices()
	require.Equal(t, 1, start)
	require.Equal(t, 3, endIdx)

	l.ScrollToBottom()
	require.Equal(t, 3, l.Offset())
	require.Equal(t, "b:0\nc:0\n\nend:0", l.Render())
	require.True(t, l.AtBottom())
	l.ScrollBy(-1)
	require.Equal(t, "a:0\nb:0\nc:0\n", l.Render())
	l.ScrollBy(1)
	require.Equal(t, "b:0\nc:0\n\nend:0", l.Render())

	l.ScrollToTop()
	l.SetSelected(4)
	l.ScrollToSelected()
	require.Equal(t, "b:0\nc:0\n\nend:0", l.Render())
	require.True(t, l.SelectedItemInView())

	l.SetItems(intro, a, c, end)
	require.Equal(t, 6, l.TotalHeight())
	l.SetGap(2)
	require.Equal(t, 8, l.TotalHeight())
}
