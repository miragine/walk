// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package walk

import (
	"sort"
	"sync"
)

type gridLayoutCell struct {
	row        int
	column     int
	widgetBase *WidgetBase
}

type gridLayoutSection struct {
	greedyNonSpacerCount int
	greedySpacerCount    int
}

type gridLayoutWidgetInfo struct {
	cell     *gridLayoutCell
	spanHorz int
	spanVert int
	minSize  SizePixels
}

type GridLayout struct {
	LayoutBase
	rowStretchFactors    []int
	columnStretchFactors []int
	widgetBase2Info      map[*WidgetBase]*gridLayoutWidgetInfo
	cells                [][]gridLayoutCell
}

func NewGridLayout() *GridLayout {
	l := &GridLayout{
		LayoutBase: LayoutBase{
			margins96dpi: Margins{9, 9, 9, 9},
			spacing96dpi: 6,
		},
		widgetBase2Info: make(map[*WidgetBase]*gridLayoutWidgetInfo),
	}
	l.layout = l

	return l
}

func (l *GridLayout) sufficientStretchFactors(stretchFactors []int, required int) []int {
	oldLen := len(stretchFactors)
	if oldLen < required {
		if cap(stretchFactors) < required {
			temp := make([]int, required, maxi(required, len(stretchFactors)*2))
			copy(temp, stretchFactors)
			stretchFactors = temp
		} else {
			stretchFactors = stretchFactors[:required]
		}

		for i := oldLen; i < len(stretchFactors); i++ {
			stretchFactors[i] = 1
		}
	}

	return stretchFactors
}

func (l *GridLayout) ensureSufficientSize(rows, columns int) {
	l.rowStretchFactors = l.sufficientStretchFactors(l.rowStretchFactors, rows)
	l.columnStretchFactors = l.sufficientStretchFactors(l.columnStretchFactors, columns)

	if len(l.cells) < len(l.rowStretchFactors) {
		if cap(l.cells) < cap(l.rowStretchFactors) {
			temp := make([][]gridLayoutCell, len(l.rowStretchFactors), cap(l.rowStretchFactors))
			copy(temp, l.cells)
			l.cells = temp
		} else {
			l.cells = l.cells[:len(l.rowStretchFactors)]
		}
	}

	for i := 0; i < len(l.cells); i++ {
		if len(l.cells[i]) < len(l.columnStretchFactors) {
			if cap(l.cells[i]) < cap(l.columnStretchFactors) {
				temp := make([]gridLayoutCell, len(l.columnStretchFactors))
				copy(temp, l.cells[i])
				l.cells[i] = temp
			} else {
				l.cells[i] = l.cells[i][:len(l.columnStretchFactors)]
			}
		}
	}

	// FIXME: Not sure if this works.
	for wb, info := range l.widgetBase2Info {
		l.widgetBase2Info[wb].cell = &l.cells[info.cell.row][info.cell.column]
	}
}

func (l *GridLayout) RowStretchFactor(row int) int {
	if row < 0 {
		// FIXME: Should we rather return an error?
		return -1
	}

	if row >= len(l.rowStretchFactors) {
		return 1
	}

	return l.rowStretchFactors[row]
}

func (l *GridLayout) SetRowStretchFactor(row, factor int) error {
	if row < 0 {
		return newError("row must be >= 0")
	}

	if factor != l.RowStretchFactor(row) {
		if l.container == nil {
			return newError("container required")
		}
		if factor < 1 {
			return newError("factor must be >= 1")
		}

		l.ensureSufficientSize(row+1, len(l.columnStretchFactors))

		l.rowStretchFactors[row] = factor

		l.container.RequestLayout()
	}

	return nil
}

func (l *GridLayout) ColumnStretchFactor(column int) int {
	if column < 0 {
		// FIXME: Should we rather return an error?
		return -1
	}

	if column >= len(l.columnStretchFactors) {
		return 1
	}

	return l.columnStretchFactors[column]
}

func (l *GridLayout) SetColumnStretchFactor(column, factor int) error {
	if column < 0 {
		return newError("column must be >= 0")
	}

	if factor != l.ColumnStretchFactor(column) {
		if l.container == nil {
			return newError("container required")
		}
		if factor < 1 {
			return newError("factor must be >= 1")
		}

		l.ensureSufficientSize(len(l.rowStretchFactors), column+1)

		l.columnStretchFactors[column] = factor

		l.container.RequestLayout()
	}

	return nil
}

func rangeFromGridLayoutWidgetInfo(info *gridLayoutWidgetInfo) RectangleGrid {
	return RectangleGrid{
		X:      info.cell.column,
		Y:      info.cell.row,
		Width:  info.spanHorz,
		Height: info.spanVert,
	}
}

func (l *GridLayout) setWidgetOnCells(widget Widget, r RectangleGrid) {
	var wb *WidgetBase
	if widget != nil {
		wb = widget.AsWidgetBase()
	}

	for row := r.Y; row < r.Y+r.Height; row++ {
		for col := r.X; col < r.X+r.Width; col++ {
			l.cells[row][col].widgetBase = wb
		}
	}
}

func (l *GridLayout) Range(widget Widget) (r RectangleGrid, ok bool) {
	if widget == nil {
		return RectangleGrid{}, false
	}

	info := l.widgetBase2Info[widget.AsWidgetBase()]

	if info == nil ||
		l.container == nil ||
		!l.container.Children().containsHandle(widget.Handle()) {
		return RectangleGrid{}, false
	}

	return rangeFromGridLayoutWidgetInfo(info), true
}

func (l *GridLayout) SetRange(widget Widget, r RectangleGrid) error {
	if widget == nil {
		return newError("widget required")
	}
	if l.container == nil {
		return newError("container required")
	}
	if !l.container.Children().containsHandle(widget.Handle()) {
		return newError("widget must be child of container")
	}
	if r.X < 0 || r.Y < 0 {
		return newError("range.X and range.Y must be >= 0")
	}
	if r.Width < 1 || r.Height < 1 {
		return newError("range.Width and range.Height must be >= 1")
	}

	wb := widget.AsWidgetBase()

	info := l.widgetBase2Info[wb]
	if info == nil {
		info = new(gridLayoutWidgetInfo)
	} else {
		l.setWidgetOnCells(nil, rangeFromGridLayoutWidgetInfo(info))
	}

	l.ensureSufficientSize(r.Y+r.Height, r.X+r.Width)

	cell := &l.cells[r.Y][r.X]
	cell.row = r.Y
	cell.column = r.X

	if info.cell == nil {
		// We have to do this _after_ calling ensureSufficientSize().
		l.widgetBase2Info[wb] = info
	}

	info.cell = cell
	info.spanHorz = r.Width
	info.spanVert = r.Height

	l.setWidgetOnCells(widget, r)

	return nil
}

func (l *GridLayout) CreateLayoutItem(ctx *LayoutContext) ContainerLayoutItem {
	wb2Item := make(map[*WidgetBase]LayoutItem)

	var children []LayoutItem

	cells := make([][]gridLayoutItemCell, len(l.cells))
	for row, srcCols := range l.cells {
		dstCols := make([]gridLayoutItemCell, len(srcCols))
		cells[row] = dstCols

		for col, srcCell := range srcCols {
			dstCell := &dstCols[col]

			dstCell.row = row
			dstCell.column = col
			if srcCell.widgetBase != nil {
				item, ok := wb2Item[srcCell.widgetBase]
				if !ok {
					item = createLayoutItemForWidgetWithContext(srcCell.widgetBase.window.(Widget), ctx)
					children = append(children, item)
					wb2Item[srcCell.widgetBase] = item

				}
				dstCell.item = item
			}
		}
	}

	item2Info := make(map[LayoutItem]*gridLayoutItemInfo, len(l.widgetBase2Info))
	for wb, info := range l.widgetBase2Info {
		item := wb2Item[wb]
		var cell *gridLayoutItemCell
		if info.cell != nil {
			cell = &cells[info.cell.row][info.cell.column]
		}
		item2Info[item] = &gridLayoutItemInfo{
			cell:     cell,
			spanHorz: info.spanHorz,
			spanVert: info.spanVert,
			minSize:  info.minSize,
		}
	}

	return &gridLayoutItem{
		ContainerLayoutItemBase: ContainerLayoutItemBase{
			children: children,
		},
		size2MinSize:         make(map[SizePixels]SizePixels),
		rowStretchFactors:    append([]int(nil), l.rowStretchFactors...),
		columnStretchFactors: append([]int(nil), l.columnStretchFactors...),
		item2Info:            item2Info,
		cells:                cells,
	}
}

type gridLayoutItem struct {
	ContainerLayoutItemBase
	mutex                sync.Mutex
	size2MinSize         map[SizePixels]SizePixels
	rowStretchFactors    []int
	columnStretchFactors []int
	item2Info            map[LayoutItem]*gridLayoutItemInfo
	cells                [][]gridLayoutItemCell
	minSize              SizePixels
}

type gridLayoutItemInfo struct {
	cell     *gridLayoutItemCell
	spanHorz int
	spanVert int
	minSize  SizePixels
}

type gridLayoutItemCell struct {
	row    int
	column int
	item   LayoutItem
}

func (*gridLayoutItem) stretchFactorsTotal(stretchFactors []int) int {
	total := 0

	for _, v := range stretchFactors {
		total += maxi(1, v)
	}

	return total
}

func (li *gridLayoutItem) LayoutFlags() LayoutFlags {
	var flags LayoutFlags

	if len(li.children) == 0 {
		return ShrinkableHorz | ShrinkableVert | GrowableHorz | GrowableVert
	} else {
		for _, item := range li.children {
			if s, ok := item.(*spacerLayoutItem); ok && s.greedyLocallyOnly || !shouldLayoutItem(item) {
				continue
			}

			wf := item.LayoutFlags()

			if wf&GreedyHorz != 0 && item.Geometry().MaxSize.Width > 0 {
				wf &^= GreedyHorz
			}
			if wf&GreedyVert != 0 && item.Geometry().MaxSize.Height > 0 {
				wf &^= GreedyVert
			}

			flags |= wf
		}
	}

	return flags
}

func (li *gridLayoutItem) IdealSize() SizePixels {
	return li.MinSize()
}

func (li *gridLayoutItem) MinSize() SizePixels {
	if len(li.cells) == 0 {
		return SizePixels{}
	}

	return li.MinSizeForSize(li.geometry.ClientSize)
}

func (li *gridLayoutItem) HeightForWidth(width Pixel) Pixel {
	return li.MinSizeForSize(SizePixels{width, li.geometry.ClientSize.Height}).Height
}

func (li *gridLayoutItem) MinSizeForSize(size SizePixels) SizePixels {
	if len(li.cells) == 0 {
		return SizePixels{}
	}

	li.mutex.Lock()
	defer li.mutex.Unlock()

	if min, ok := li.size2MinSize[size]; ok {
		return min
	}

	ws := make([]Pixel, len(li.cells[0]))

	for row := 0; row < len(li.cells); row++ {
		for col := 0; col < len(ws); col++ {
			item := li.cells[row][col].item
			if item == nil {
				continue
			}

			if !shouldLayoutItem(item) {
				continue
			}

			min := li.MinSizeEffectiveForChild(item)
			info := li.item2Info[item]

			if info.spanHorz == 1 {
				ws[col] = maxPixel(ws[col], min.Width)
			}
		}
	}

	widths := li.sectionSizesForSpace(Horizontal, size.Width, nil)
	heights := li.sectionSizesForSpace(Vertical, size.Height, widths)

	for row := range heights {
		var wg sync.WaitGroup
		var mutex sync.Mutex
		var maxHeight Pixel

		for col := range widths {
			item := li.cells[row][col].item
			if item == nil {
				continue
			}

			if !shouldLayoutItem(item) {
				continue
			}

			if info := li.item2Info[item]; info.spanVert == 1 {
				if hfw, ok := item.(HeightForWidther); ok && hfw.HasHeightForWidth() {
					wg.Add(1)

					go func() {
						height := hfw.HeightForWidth(li.spannedWidth(info, widths))

						mutex.Lock()
						maxHeight = maxPixel(maxHeight, height)
						mutex.Unlock()

						wg.Done()
					}()
				} else {
					height := li.MinSizeEffectiveForChild(item).Height

					mutex.Lock()
					maxHeight = maxPixel(maxHeight, height)
					mutex.Unlock()
				}
			}
		}

		wg.Wait()

		heights[row] = maxHeight
	}

	marginsPixels := MarginsFrom96DPI(li.margins, li.ctx.dpi)
	spacingPixels := IntFrom96DPI(li.spacing, li.ctx.dpi)

	width := marginsPixels.HNear + marginsPixels.HFar
	height := marginsPixels.VNear + marginsPixels.VFar

	for i, w := range ws {
		if w > 0 {
			if i > 0 {
				width += spacingPixels
			}
			width += w
		}
	}
	for i, h := range heights {
		if h > 0 {
			if i > 0 {
				height += spacingPixels
			}
			height += h
		}
	}

	if width > 0 && height > 0 {
		li.size2MinSize[size] = SizePixels{width, height}
	}

	return SizePixels{width, height}
}

func (li *gridLayoutItem) spannedWidth(info *gridLayoutItemInfo, widths []Pixel) Pixel {
	spacingPixels := IntFrom96DPI(li.spacing, li.ctx.dpi)

	var width Pixel

	for i := info.cell.column; i < info.cell.column+info.spanHorz; i++ {
		if w := widths[i]; w > 0 {
			width += w
			if i > info.cell.column {
				width += spacingPixels
			}
		}
	}

	return width
}

func (li *gridLayoutItem) spannedHeight(info *gridLayoutItemInfo, heights []Pixel) Pixel {
	spacingPixels := IntFrom96DPI(li.spacing, li.ctx.dpi)

	var height Pixel

	for i := info.cell.row; i < info.cell.row+info.spanVert; i++ {
		if h := heights[i]; h > 0 {
			height += h
			if i > info.cell.row {
				height += spacingPixels
			}
		}
	}

	return height
}

type gridLayoutSectionInfo struct {
	index              int
	minSize            Pixel
	maxSize            Pixel
	stretch            int
	hasGreedyNonSpacer bool
	hasGreedySpacer    bool
}

type gridLayoutSectionInfoList []gridLayoutSectionInfo

func (l gridLayoutSectionInfoList) Len() int {
	return len(l)
}

func (l gridLayoutSectionInfoList) Less(i, j int) bool {
	if l[i].hasGreedyNonSpacer == l[j].hasGreedyNonSpacer {
		if l[i].hasGreedySpacer == l[j].hasGreedySpacer {
			minDiff := l[i].minSize - l[j].minSize

			if minDiff == 0 {
				return int(l[i].maxSize)/l[i].stretch < int(l[j].maxSize)/l[j].stretch
			}

			return minDiff > 0
		}

		return l[i].hasGreedySpacer
	}

	return l[i].hasGreedyNonSpacer
}

func (l gridLayoutSectionInfoList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (li *gridLayoutItem) PerformLayout() []LayoutResultItem {
	widths := li.sectionSizesForSpace(Horizontal, li.geometry.ClientSize.Width, nil)
	heights := li.sectionSizesForSpace(Vertical, li.geometry.ClientSize.Height, widths)

	items := make([]LayoutResultItem, 0, len(li.item2Info))

	marginsPixels := MarginsFrom96DPI(li.margins, li.ctx.dpi)
	spacingPixels := IntFrom96DPI(li.spacing, li.ctx.dpi)

	for item, info := range li.item2Info {
		if !shouldLayoutItem(item) {
			continue
		}

		x := marginsPixels.HNear
		for i := 0; i < info.cell.column; i++ {
			if w := widths[i]; w > 0 {
				x += w + spacingPixels
			}
		}

		y := marginsPixels.VNear
		for i := 0; i < info.cell.row; i++ {
			if h := heights[i]; h > 0 {
				y += h + spacingPixels
			}
		}

		width := li.spannedWidth(info, widths)
		height := li.spannedHeight(info, heights)

		w := width
		h := height

		if lf := item.LayoutFlags(); lf&GrowableHorz == 0 || lf&GrowableVert == 0 {
			var s SizePixels
			if hfw, ok := item.(HeightForWidther); !ok || !hfw.HasHeightForWidth() {
				if is, ok := item.(IdealSizer); ok {
					s = is.IdealSize()
				}
			}

			max := item.Geometry().MaxSize
			if max.Width > 0 && s.Width > max.Width {
				s.Width = max.Width
			}
			if lf&GrowableHorz == 0 {
				w = s.Width
			}

			if hfw, ok := item.(HeightForWidther); ok && hfw.HasHeightForWidth() {
				h = hfw.HeightForWidth(w)
			} else {
				if max.Height > 0 && s.Height > max.Height {
					s.Height = max.Height
				}
				if lf&GrowableVert == 0 {
					h = s.Height
				}
			}
		}

		alignment := item.Geometry().Alignment
		if alignment == AlignHVDefault {
			alignment = li.alignment
		}

		if w != width {
			switch alignment {
			case AlignHCenterVNear, AlignHCenterVCenter, AlignHCenterVFar:
				x += (width - w) / 2

			case AlignHFarVNear, AlignHFarVCenter, AlignHFarVFar:
				x += width - w
			}
		}

		if h != height {
			switch alignment {
			case AlignHNearVCenter, AlignHCenterVCenter, AlignHFarVCenter:
				y += (height - h) / 2

			case AlignHNearVFar, AlignHCenterVFar, AlignHFarVFar:
				y += height - h
			}
		}

		items = append(items, LayoutResultItem{Item: item, Bounds: RectanglePixels{X: x, Y: y, Width: w, Height: h}})
	}

	return items
}

func (li *gridLayoutItem) sectionSizesForSpace(orientation Orientation, space Pixel, widths []Pixel) []Pixel {
	var stretchFactors []int
	if orientation == Horizontal {
		stretchFactors = li.columnStretchFactors
	} else {
		stretchFactors = li.rowStretchFactors
	}

	var sectionCountWithGreedyNonSpacer int
	var sectionCountWithGreedySpacer int
	var stretchFactorsTotal [3]int
	var minSizesRemaining Pixel
	minSizes := make([]Pixel, len(stretchFactors))
	maxSizes := make([]Pixel, len(stretchFactors))
	sizes := make([]Pixel, len(stretchFactors))
	sortedSections := gridLayoutSectionInfoList(make([]gridLayoutSectionInfo, len(stretchFactors)))

	for i := 0; i < len(stretchFactors); i++ {
		var otherAxisCount int
		if orientation == Horizontal {
			otherAxisCount = len(li.rowStretchFactors)
		} else {
			otherAxisCount = len(li.columnStretchFactors)
		}

		for j := 0; j < otherAxisCount; j++ {
			var item LayoutItem
			if orientation == Horizontal {
				item = li.cells[j][i].item
			} else {
				item = li.cells[i][j].item
			}

			if item == nil {
				continue
			}

			if !shouldLayoutItem(item) {
				continue
			}

			info := li.item2Info[item]
			flags := item.LayoutFlags()

			max := item.Geometry().MaxSize

			var pref SizePixels
			if hfw, ok := item.(HeightForWidther); !ok || !hfw.HasHeightForWidth() {
				if is, ok := item.(IdealSizer); ok {
					pref = is.IdealSize()
				}
			}

			if orientation == Horizontal {
				if info.spanHorz == 1 {
					minSizes[i] = maxPixel(minSizes[i], li.MinSizeEffectiveForChild(item).Width)
				}

				if max.Width > 0 {
					maxSizes[i] = maxPixel(maxSizes[i], max.Width)
				} else if pref.Width > 0 && flags&GrowableHorz == 0 {
					maxSizes[i] = maxPixel(maxSizes[i], pref.Width)
				} else {
					maxSizes[i] = 32768
				}

				if info.spanHorz == 1 && flags&GreedyHorz > 0 {
					if _, isSpacer := item.(*spacerLayoutItem); isSpacer {
						sortedSections[i].hasGreedySpacer = true
					} else {
						sortedSections[i].hasGreedyNonSpacer = true
					}
				}
			} else {
				if info.spanVert == 1 {
					if hfw, ok := item.(HeightForWidther); ok && hfw.HasHeightForWidth() {
						minSizes[i] = maxPixel(minSizes[i], hfw.HeightForWidth(li.spannedWidth(info, widths)))
					} else {
						minSizes[i] = maxPixel(minSizes[i], li.MinSizeEffectiveForChild(item).Height)
					}
				}

				if max.Height > 0 {
					maxSizes[i] = maxPixel(maxSizes[i], max.Height)
				} else if hfw, ok := item.(HeightForWidther); ok && flags&GrowableVert == 0 && hfw.HasHeightForWidth() {
					maxSizes[i] = minSizes[i]
				} else if pref.Height > 0 && flags&GrowableVert == 0 {
					maxSizes[i] = maxPixel(maxSizes[i], pref.Height)
				} else {
					maxSizes[i] = 32768
				}

				if info.spanVert == 1 && flags&GreedyVert > 0 {
					if _, isSpacer := item.(*spacerLayoutItem); isSpacer {
						sortedSections[i].hasGreedySpacer = true
					} else {
						sortedSections[i].hasGreedyNonSpacer = true
					}
				}
			}
		}

		sortedSections[i].index = i
		sortedSections[i].minSize = minSizes[i]
		sortedSections[i].maxSize = maxSizes[i]
		sortedSections[i].stretch = maxi(1, stretchFactors[i])

		minSizesRemaining += minSizes[i]

		if sortedSections[i].hasGreedyNonSpacer {
			sectionCountWithGreedyNonSpacer++
			stretchFactorsTotal[0] += stretchFactors[i]
		} else if sortedSections[i].hasGreedySpacer {
			sectionCountWithGreedySpacer++
			stretchFactorsTotal[1] += stretchFactors[i]
		} else {
			stretchFactorsTotal[2] += stretchFactors[i]
		}
	}

	sort.Stable(sortedSections)

	marginsPixels := MarginsFrom96DPI(li.margins, li.ctx.dpi)
	spacingPixels := IntFrom96DPI(li.spacing, li.ctx.dpi)

	if orientation == Horizontal {
		space -= marginsPixels.HNear + marginsPixels.HFar
	} else {
		space -= marginsPixels.VNear + marginsPixels.VFar
	}

	var spacingRemaining Pixel
	for _, max := range maxSizes {
		if max > 0 {
			spacingRemaining += spacingPixels
		}
	}
	if spacingRemaining > 0 {
		spacingRemaining -= spacingPixels
	}

	offsets := [3]int{0, sectionCountWithGreedyNonSpacer, sectionCountWithGreedyNonSpacer + sectionCountWithGreedySpacer}
	counts := [3]int{sectionCountWithGreedyNonSpacer, sectionCountWithGreedySpacer, len(stretchFactors) - sectionCountWithGreedyNonSpacer - sectionCountWithGreedySpacer}

	for i := 0; i < 3; i++ {
		stretchFactorsRemaining := stretchFactorsTotal[i]

		for j := 0; j < counts[i]; j++ {
			info := sortedSections[offsets[i]+j]
			k := info.index

			stretch := stretchFactors[k]
			min := info.minSize
			max := info.maxSize
			size := min

			if min < max {
				excessSpace := float64(space - minSizesRemaining - spacingRemaining)

				size += Pixel(excessSpace * float64(stretch) / float64(stretchFactorsRemaining))
				if size < min {
					size = min
				} else if size > max {
					size = max
				}
			}

			sizes[k] = size

			minSizesRemaining -= min
			stretchFactorsRemaining -= stretch

			space -= (size + spacingPixels)
			spacingRemaining -= spacingPixels
		}
	}

	return sizes
}
