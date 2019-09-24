// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package walk

const spacerWindowClass = `\o/ Walk_Spacer_Class \o/`

func init() {
	AppendToWalkInit(func() {
		MustRegisterWindowClass(spacerWindowClass)
	})
}

type Spacer struct {
	WidgetBase
	sizeHint          Size
	layoutFlags       LayoutFlags
	greedyLocallyOnly bool
}

type SpacerCfg struct {
	LayoutFlags       LayoutFlags
	SizeHint          Size
	GreedyLocallyOnly bool
}

func NewSpacerWithCfg(parent Container, cfg *SpacerCfg) (*Spacer, error) {
	return newSpacer(parent, cfg.LayoutFlags, cfg.SizeHint, cfg.GreedyLocallyOnly)
}

func newSpacer(parent Container, layoutFlags LayoutFlags, sizeHint Size, greedyLocallyOnly bool) (*Spacer, error) {
	s := &Spacer{
		layoutFlags:       layoutFlags,
		sizeHint:          sizeHint,
		greedyLocallyOnly: greedyLocallyOnly,
	}

	if err := InitWidget(
		s,
		parent,
		spacerWindowClass,
		0,
		0); err != nil {
		return nil, err
	}

	return s, nil
}

func NewHSpacer(parent Container) (*Spacer, error) {
	return newSpacer(parent, ShrinkableHorz|ShrinkableVert|GrowableHorz|GreedyHorz, Size{}, false)
}

func NewHSpacerFixed(parent Container, width int) (*Spacer, error) {
	return newSpacer(parent, 0, Size{width, 0}, false)
}

func NewVSpacer(parent Container) (*Spacer, error) {
	return newSpacer(parent, ShrinkableHorz|ShrinkableVert|GrowableVert|GreedyVert, Size{}, false)
}

func NewVSpacerFixed(parent Container, height int) (*Spacer, error) {
	return newSpacer(parent, 0, Size{0, height}, false)
}

func (s *Spacer) CreateLayoutItem(ctx *LayoutContext) LayoutItem {
	return &spacerLayoutItem{
		idealSize:         s.sizeHint,
		layoutFlags:       s.layoutFlags,
		greedyLocallyOnly: s.greedyLocallyOnly,
	}
}

type spacerLayoutItem struct {
	LayoutItemBase
	idealSize         Size
	layoutFlags       LayoutFlags
	greedyLocallyOnly bool
}

func (li *spacerLayoutItem) LayoutFlags() LayoutFlags {
	return li.layoutFlags
}

func (li *spacerLayoutItem) IdealSize() SizePixels {
	return SizeFrom96DPI(li.idealSize, li.ctx.dpi)
}

func (li *spacerLayoutItem) MinSize() SizePixels {
	return SizeFrom96DPI(li.idealSize, li.ctx.dpi)
}
