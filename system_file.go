package main

import (
	"fmt"

	rl "github.com/lachee/raylib-goplus/raylib"
)

var (
	// ShowDebug enables debug overlays when true
	ShowDebug = false
)

type UIFileSystem struct {
	BasicSystem

	Camera rl.Camera2D
	target rl.Vector2

	// Used for relational mouse movement
	mouseX, mouseY, mouseLastX, mouseLastY int

	// workaround for resizing after AddSystem call has been made
	hasDoneFirstFrameResize bool

	cursor rl.Vector2
}

func NewUIFileSystem() *UIFileSystem {
	s := &UIFileSystem{
		Camera:                  rl.Camera2D{Zoom: 8.0},
		hasDoneFirstFrameResize: false,
	}

	// Screen edges, left and top aren't needed since 0 stays constant
	screenRight := NewBlock(rl.NewRectangle(
		float32(rl.GetScreenWidth()),
		0,
		0,
		0,
	))
	if res, ok := screenRight.GetResizeable(); ok {
		res.OnResize = func(entity *Entity) {
			if mov, ok := entity.GetMoveable(); ok {
				// Move bounds to right side of the screen
				mov.Bounds.X = float32(rl.GetScreenWidth())
			}
		}
	}
	screenBottom := NewBlock(rl.NewRectangle(
		float32(rl.GetScreenWidth()),
		0,
		0,
		0,
	))
	if res, ok := screenBottom.GetResizeable(); ok {
		res.OnResize = func(entity *Entity) {
			if mov, ok := entity.GetMoveable(); ok {
				// Move bounds to right side of the screen
				mov.Bounds.Y = float32(rl.GetScreenHeight())
			}
		}
	}

	// // UI Components
	menu := NewMenuUI(rl.NewRectangle(
		0,
		0,
		float32(rl.GetScreenWidth()),
		UIFontSize*2))

	editors := NewEditorsUI(rl.NewRectangle(
		0,
		0,
		float32(rl.GetScreenWidth()),
		UIFontSize*2))
	editors.Snap([]SnapData{
		{menu, SideTop, SideBottom},
	})

	rgb := NewRGBUI(rl.NewRectangle(
		0,
		0,
		128*1.5,
		128*1.5+UIButtonHeight*1.5))
	rgb.Snap([]SnapData{
		{screenRight, SideRight, SideLeft},
	})

	palette := NewPaletteUI(rl.NewRectangle(
		0,
		0,
		128,
		128*1.5+UIButtonHeight*0.5))
	palette.Snap([]SnapData{
		{rgb, SideRight, SideLeft},
	})

	currentColor := NewCurrentColorUI(rl.NewRectangle(
		0,
		0,
		128,
		UIButtonHeight))
	currentColor.Snap([]SnapData{
		{rgb, SideRight, SideLeft},
		{palette, SideTop, SideBottom},
	})

	tools := NewToolsUI(rl.NewRectangle(
		0,
		0,
		128*1.5+UIButtonHeight*2,
		UIButtonHeight))
	tools.Snap([]SnapData{
		{currentColor, SideTop, SideBottom},
		{currentColor, SideLeft, SideLeft},
	})

	layers := NewLayersUI(rl.NewRectangle(
		0,
		0,
		128*2.5,
		128*2))
	layers.Snap([]SnapData{
		{screenRight, SideRight, SideLeft},
		{tools, SideTop, SideBottom},
	})
	if res, ok := layers.GetResizeable(); ok {
		res.OnResize = func(entity *Entity) {
			// Resize container and inner list to fill the remaining height
			if mov, ok := entity.GetMoveable(); ok {
				mov.Bounds.Height = float32(rl.GetScreenHeight()) - mov.Bounds.Y

				if lmov, ok := layerList.GetMoveable(); ok {
					lmov.Bounds.Height = float32(rl.GetScreenHeight()) - mov.Bounds.Y - UIButtonHeight
					if ldraw, ok := layerList.GetDrawable(); ok {
						if lparentDrawable, ok := ldraw.DrawableType.(*DrawableParent); ok {
							// Make a new texture for rendering the items to
							lparentDrawable.Texture = rl.LoadRenderTexture(int(lmov.Bounds.Width), int(lmov.Bounds.Height))
						}
					}
				}
			}
		}
	}

	NewResizeUI()

	return s
}

func (s *UIFileSystem) Draw() {
	layer := CurrentFile.GetCurrentLayer()

	// Draw
	rl.BeginTextureMode(layer.Canvas)
	if !layer.hasInitialFill {
		CurrentFile.ClearBackground(layer.InitialFillColor)
		layer.hasInitialFill = true
	}
	rl.EndTextureMode()

	// Draw temp layer
	rl.BeginTextureMode(CurrentFile.Layers[len(CurrentFile.Layers)-1].Canvas)
	// LeftTool draws last as it's more important
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		CurrentFile.RightTool.DrawPreview(int(s.cursor.X), int(s.cursor.Y))

	} else {
		CurrentFile.LeftTool.DrawPreview(int(s.cursor.X), int(s.cursor.Y))
	}

	rl.EndTextureMode()

	// Draw layers
	rl.BeginMode2D(s.Camera)
	for _, layer := range CurrentFile.Layers {
		if !layer.Hidden {
			rl.DrawTextureRec(layer.Canvas.Texture,
				rl.NewRectangle(0, 0, float32(layer.Canvas.Texture.Width), -float32(layer.Canvas.Texture.Height)),
				rl.NewVector2(-float32(layer.Canvas.Texture.Width)/2, -float32(layer.Canvas.Texture.Height)/2),
				rl.White)
		}
	}

	// Grid drawing
	// TODO use a high resolution texture to draw grids, then we won't need to draw lines each draw call
	if CurrentFile.DrawGrid {
		for x := 0; x <= CurrentFile.CanvasWidth; x += CurrentFile.TileWidth {
			rl.DrawLine(
				-CurrentFile.CanvasWidth/2+x,
				-CurrentFile.CanvasHeight/2,
				-CurrentFile.CanvasWidth/2+x,
				CurrentFile.CanvasHeight/2,
				rl.White)
		}
		for y := 0; y <= CurrentFile.CanvasHeight; y += CurrentFile.TileHeight {
			rl.DrawLine(
				-CurrentFile.CanvasWidth/2,
				-CurrentFile.CanvasHeight/2+y,
				CurrentFile.CanvasWidth/2,
				-CurrentFile.CanvasHeight/2+y,
				rl.White)
		}
	}

	// Show outline for canvas resize preview
	if CurrentFile.DoingResize {
		var x, y float32
		w := float32(CurrentFile.CanvasWidthResizePreview)
		h := float32(CurrentFile.CanvasHeightResizePreview)

		// Move offset
		dw := (w - float32(CurrentFile.CanvasWidth)) / 2
		dh := (h - float32(CurrentFile.CanvasHeight)) / 2

		switch CurrentFile.CanvasDirectionResizePreview {
		case ResizeTL:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 + dw
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 + dh
		case ResizeTC:
			x = -float32(CurrentFile.CanvasWidthResizePreview) / 2
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 + dh
		case ResizeTR:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 - dw
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 + dh
		case ResizeCL:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 + dw
			y = -float32(CurrentFile.CanvasHeightResizePreview) / 2
		case ResizeCC:
			x = -float32(CurrentFile.CanvasWidthResizePreview) / 2
			y = -float32(CurrentFile.CanvasHeightResizePreview) / 2
		case ResizeCR:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 - dw
			y = -float32(CurrentFile.CanvasHeightResizePreview) / 2
		case ResizeBL:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 + dw
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 - dh
		case ResizeBC:
			x = -float32(CurrentFile.CanvasWidthResizePreview) / 2
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 - dh
		case ResizeBR:
			x = -float32(CurrentFile.CanvasWidthResizePreview)/2 - dw
			y = -float32(CurrentFile.CanvasHeightResizePreview)/2 - dh
		}

		rl.DrawRectangleLinesEx(
			rl.NewRectangle(x, y, w, h),
			1,
			rl.Red,
		)
	}

	rl.EndMode2D()

	// Debug text
	if ShowDebug {
		rl.DrawText(fmt.Sprintf("CanvasWidthResizePreview: %v", CurrentFile.CanvasWidthResizePreview), 0, 80, 20, rl.White)
		rl.DrawText(fmt.Sprintf("CanvasHeightResizePreview: %v", CurrentFile.CanvasHeightResizePreview), 0, 100, 20, rl.White)
		rl.DrawText(fmt.Sprintf("TileWidthResizePreview: %v", CurrentFile.TileWidthResizePreview), 0, 120, 20, rl.White)
		rl.DrawText(fmt.Sprintf("TileHeightResizePreview: %v", CurrentFile.TileHeightResizePreview), 0, 140, 20, rl.White)

		rl.DrawText(fmt.Sprintf("UIInteractableCapturedInput: %v", UIInteractableCapturedInput), 0, 160, 20, rl.White)
		rl.DrawText(fmt.Sprintf("UIEntityCapturedInput: %v", UIEntityCapturedInput), 0, 180, 20, rl.White)
		rl.DrawText(fmt.Sprintf("Current layer: %d", CurrentFile.CurrentLayer), 0, 200, 20, rl.White)
		rl.DrawText(fmt.Sprintf("HistoryOffset: %d", CurrentFile.historyOffset), 0, 220, 20, rl.White)
		rl.DrawText(fmt.Sprintf("History Len: %d", len(CurrentFile.History)), 0, 240, 20, rl.White)
		// for y, history := range CurrentFile.History {
		// 	str := fmt.Sprintf("Layer: %d, Diff: %d",
		// 		history.LayerIndex,
		// 		len(history.PixelState))
		// 	rl.DrawText(str, 20, 20*y+260, 20, rl.White)
		// }
	}
}

func recursiveResize(entity *Entity) {
	if res, ok := entity.GetResizeable(); ok {
		if len(res.SnappedTo) > 0 {
			for _, snapData := range res.SnappedTo {
				recursiveResize(snapData.Parent)

				if childMoveable, ok := entity.GetMoveable(); ok {
					if parentMoveable, ok := snapData.Parent.GetMoveable(); ok {
						switch snapData.SnapSideChild {
						case SideLeft:
							childMoveable.Bounds.X = parentMoveable.Bounds.X
							if snapData.SnapSideParent == SideRight {
								childMoveable.Bounds.X += parentMoveable.Bounds.Width
							}
						case SideRight:
							childMoveable.Bounds.X = parentMoveable.Bounds.X
							if snapData.SnapSideParent == SideLeft {
								childMoveable.Bounds.X -= childMoveable.Bounds.Width
							}
						case SideTop:
							childMoveable.Bounds.Y = parentMoveable.Bounds.Y
							if snapData.SnapSideParent == SideBottom {
								childMoveable.Bounds.Y += parentMoveable.Bounds.Height
							}
						case SideBottom:
							childMoveable.Bounds.Y = parentMoveable.Bounds.Y
							if snapData.SnapSideParent == SideTop {
								childMoveable.Bounds.Y -= childMoveable.Bounds.Height
							}
						}
					}
				}
			}
		}

		entity.FlowChildren()

		if res.OnResize != nil {
			res.OnResize(entity)
		}
	}
}

func (s *UIFileSystem) Resize() {
	s.Camera.Offset.X = float32(rl.GetScreenWidth()) / 2
	s.Camera.Offset.Y = float32(rl.GetScreenHeight()) / 2

	s.hasDoneFirstFrameResize = true

	for _, result := range s.Scene.QueryTag(s.Scene.Tags["resizeable"], s.Scene.Tags["moveable"]) {
		_ = result
		recursiveResize(result.Entity)
	}

}

func (s *UIFileSystem) Update(dt float32) {
	// Move target
	if rl.IsWindowResized() || s.hasDoneFirstFrameResize == false {
		s.Resize()
	}

	layer := CurrentFile.GetCurrentLayer()
	s.mouseX = rl.GetMouseX()
	s.mouseY = rl.GetMouseY()

	// Scroll towards the cursor's location
	if !UIHasControl {
		scrollAmount := rl.GetMouseWheelMove()
		if scrollAmount != 0 {
			// TODO scroll scalar in config (0.1)
			s.target.X += ((float32(s.mouseX) - float32(rl.GetScreenWidth())/2) / (s.Camera.Zoom * 10)) * float32(scrollAmount)
			s.target.Y += ((float32(s.mouseY) - float32(rl.GetScreenHeight())/2) / (s.Camera.Zoom * 10)) * float32(scrollAmount)
			s.Camera.Target = s.target
			s.Camera.Zoom += float32(scrollAmount) * 0.1 * s.Camera.Zoom
		}
	}

	if rl.IsMouseButtonDown(rl.MouseMiddleButton) {
		s.target.X += float32(s.mouseLastX-s.mouseX) / s.Camera.Zoom
		s.target.Y += float32(s.mouseLastY-s.mouseY) / s.Camera.Zoom
	}
	s.mouseLastX = s.mouseX
	s.mouseLastY = s.mouseY
	s.Camera.Target = s.target

	s.cursor = rl.GetScreenToWorld2D(rl.GetMousePosition(), s.Camera)
	s.cursor = s.cursor.Add(rl.NewVector2(float32(layer.Canvas.Texture.Width)/2, float32(layer.Canvas.Texture.Height)/2))
	if !UIHasControl {
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			// Fires once
			if CurrentFile.HasDoneMouseUpLeft {
				// Create new history action
				CurrentFile.AppendHistory(HistoryPixel{make(map[IntVec2]PixelStateData), CurrentFile.CurrentLayer})
			}
			CurrentFile.HasDoneMouseUpLeft = false

			// Repeated action
			CurrentFile.LeftTool.MouseDown(int(s.cursor.X), int(s.cursor.Y), rl.MouseLeftButton)
		} else {
			// Always fires once
			if CurrentFile.HasDoneMouseUpLeft == false {
				CurrentFile.HasDoneMouseUpLeft = true
				CurrentFile.LeftTool.MouseUp(int(s.cursor.X), int(s.cursor.Y), rl.MouseLeftButton)
			}
		}

		if rl.IsMouseButtonDown(rl.MouseRightButton) {
			if CurrentFile.HasDoneMouseUpRight {
				CurrentFile.AppendHistory(HistoryPixel{make(map[IntVec2]PixelStateData), CurrentFile.CurrentLayer})
			}
			CurrentFile.HasDoneMouseUpRight = false
			CurrentFile.RightTool.MouseDown(int(s.cursor.X), int(s.cursor.Y), rl.MouseRightButton)
		} else {
			if CurrentFile.HasDoneMouseUpRight == false {
				CurrentFile.HasDoneMouseUpRight = true
				CurrentFile.RightTool.MouseUp(int(s.cursor.X), int(s.cursor.Y), rl.MouseRightButton)
			}
		}
	}
}
