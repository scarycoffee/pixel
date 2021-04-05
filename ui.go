package main

import (
	"fmt"
	"log"
	"time"

	rl "github.com/lachee/raylib-goplus/raylib"
)

// UI is the interface for UI elements (they handle their own components + states)
type UI interface {
}

var (
	// UIHasControl lets the program know if input should go to the UI or not
	UIHasControl = false
	// UIIsInputtingText allows click events to cancel out text input
	UIIsInputtingText = false
	// UIInteractableCapturedInput is the current Interactable with control
	UIInteractableCapturedInput *Interactable
	// UIEntityCapturedInput is the current Entity with control
	UIEntityCapturedInput *Entity
	// UIComponentWithControl is the current ui component with control
	// UIComponentWithControl UIComponent
	// isInited is a flag to record if InitUI has been called
	isInited = false
	// Font is the font used
	Font *rl.Font
	// UIFontSize is the size of the font
	UIFontSize float32 = 14

	uiCamera               = rl.Camera2D{Zoom: 1}
	mouseX, mouseY         int
	mouseLastX, mouseLastY = -1, -1

	// Ecs stuffs
	scene                                                               *Scene
	moveable, resizeable, interactable, hoverable, drawable, scrollable *Component
	renderSystem                                                        *UIRenderSystem
	controlSystem                                                       *UIControlSystem
	fileSystem                                                          *UIFileSystem
)

const (
	// MouseButtonNone is for when no mouse button event is needed, but up event hasn't happened
	MouseButtonNone = rl.MouseButton(3)
)

// Moveable gives a component a position and dimensions
type Moveable struct {
	// Bounds is the position and dimensions of the component
	Bounds rl.Rectangle
	// OrigBounds is used when repositioning the element (stops offset stacking)
	OrigBounds rl.Rectangle
	// Offset values from scrolling
	Offset rl.Vector2
	// LayoutTag is how the elements should be arranged
	LayoutTag LayoutTag
}

// Resizeable allows a component to be resized and stores some callbacks
type Resizeable struct {
}

// Interactable is for storing all callbacks which can be procced by user inputs
// The callbacks are optional
type Interactable struct {
	// ButtonDown keeps track of if a button is down
	ButtonDown rl.MouseButton

	// ButtonDownAt is the time when the button was pressed
	// Used to allow drag events after a certain amount of time has elapsed
	ButtonDownAt time.Time

	// ButtonReleased is used to prevent multiple up events from firing if the
	// component has an OnKeyPress event stalling execution
	ButtonReleased bool

	// OnMouseDown fires every frame the mouse button is down on the element
	// isHeld can be used to work out if a drag event should happen, or if only
	// one down event should be executed etc
	OnMouseDown func(entity *Entity, button rl.MouseButton, isHeld bool)
	// OnMouseUp fires once when the mouse is released (doesn't fire if mouse
	// is released while not within the bounds! Draggable should be used for
	// this kind of event instead)
	OnMouseUp func(entity *Entity, button rl.MouseButton)

	// OnScroll is for mouse wheel actions
	OnScroll func(direction int)

	// OnKeyPress is called when a key is released
	OnKeyPress func(entity *Entity, key rl.Key)
}

// ScrollDirection states the scroll direction of the component
type ScrollDirection int

const (
	// ScrollDirectionVertical is for vertical scrolling
	ScrollDirectionVertical ScrollDirection = iota
	// ScrollDirectionHorizontal is for horizontal scrolling
	ScrollDirectionHorizontal
)

// LayoutTag states which direction the children elements should flow in
type LayoutTag int

const (
	// FlowDirectionNone doesn't reflow elements
	FlowDirectionNone LayoutTag = 1 << iota
	// FlowDirectionVertical flows vertically
	FlowDirectionVertical
	// FlowDirectionVerticalReversed flows vertically, in reverse order
	FlowDirectionVerticalReversed
	// FlowDirectionHorizontal flows horizontally
	FlowDirectionHorizontal
	// FlowDirectionHorizontalReversed flows horizontally, in reverse order
	FlowDirectionHorizontalReversed
)

// Scrollable allows an element to render its children elements with an offset
type Scrollable struct {
	// ScrollDirection states which way the content should scroll
	ScrollDirection ScrollDirection
	// ScrollOffset is how much the content should be offset
	ScrollOffset int

	// TODO stuff for rendering scrollbars differently
}

// Hoverable stores the hovered and seleceted states
type Hoverable struct {
	Hovered  bool
	Selected bool

	// Split selection to display which tool/color is bound to which mouse button
	// TODO implement
	SelectedLeft  bool
	SelectedRight bool
}

// Drawable handles all drawing related information
type Drawable struct {
	// DrawableType can be DrawableText, DrawableTexture or DrawableParent
	DrawableType interface{}

	// IsChild prevents normal rendering and instead renders to its
	// DrawableParent Texture
	IsChild bool
}

// DrawableText draws text
type DrawableText struct {
	Label string
}

// SetTexture sets the texture of a DrawableTexture to the path given.
// Doesn't cache, so it's probably not very efficient.
func (d *DrawableTexture) SetTexture(path string) {
	d.Texture = rl.LoadTexture(path)
}

// DrawableTexture draws a texture
type DrawableTexture struct {
	Texture rl.Texture2D
}

// NewDrawableTexture returns a pointer to a DrawableTexture
func NewDrawableTexture(texturePath string) *DrawableTexture {
	d := &DrawableTexture{
		Texture: rl.LoadTexture(texturePath),
	}
	return d
}

// DrawableRenderTexture is like DrawableTexture, but it's intended to be used
// with rl.BeginTextureMode
type DrawableRenderTexture struct {
	Texture rl.RenderTexture2D
}

// DrawableParent draws its children to its texture if IsPassthrough is true
type DrawableParent struct {
	// If true, doesn't draw to the Texture
	IsPassthrough bool
	Texture       rl.RenderTexture2D

	Children []*Entity
}

// InitUI must be called before UI is used
func InitUI(keymap Keymap) {
	isInited = true
	Font = rl.LoadFont("./res/fonts/prstartk.ttf")

	scene = NewScene()

	moveable = scene.NewComponent("moveable")
	resizeable = scene.NewComponent("resizeable")
	interactable = scene.NewComponent("interactable")
	scrollable = scene.NewComponent("scrollable")
	hoverable = scene.NewComponent("hoverable")
	drawable = scene.NewComponent("drawable")

	drawable.SetDestructor(func(e *Entity, data interface{}) {
		d, ok := data.(*Drawable)
		if ok {
			switch t := d.DrawableType.(type) {
			case *DrawableParent:
				if !t.IsPassthrough {
					t.Texture.Unload()
				}
			case *DrawableTexture:
				t.Texture.Unload()
			}
		}
	})

	scene.BuildTag("moveable", moveable)
	scene.BuildTag("resizeable", resizeable)
	scene.BuildTag("interactable", interactable)
	scene.BuildTag("scrollable", scrollable)
	scene.BuildTag("hoverable", hoverable)
	scene.BuildTag("drawable", drawable)
	scene.BuildTag("basic", drawable, moveable, hoverable)
	scene.BuildTag("basicControl", drawable, moveable, hoverable, interactable)

	controlSystem = NewUIControlSystem(keymap)
	renderSystem = NewUIRenderSystem()
	fileSystem = NewUIFileSystem()

	scene.AddSystem(controlSystem)
	scene.AddSystem(renderSystem)
	scene.AddSystem(fileSystem)
}

// DestroyUI calls the destructor on every entity/component
func DestroyUI() {
	scene.Destroy()
}

// UpdateUI updates the systems (excluding the RenderSystem)
func UpdateUI() {
	controlSystem.Update(rl.GetFrameTime())
	fileSystem.Update(rl.GetFrameTime())
}

// DrawUI draws the RenderSystem
func DrawUI() {
	fileSystem.Draw()   // draw layer canvases etc
	renderSystem.Draw() // draw ui components
}

// GetChildren returns a copy of all of the children entities from an entity
func (e *Entity) GetChildren() ([]*Entity, error) {
	if result, err := scene.QueryID(e.ID); err == nil {
		drawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
		drawableParent, ok := drawable.DrawableType.(*DrawableParent)

		if ok {
			return drawableParent.Children[:], nil
		}
	}
	return nil, fmt.Errorf("No children")
}

// RemoveChild removes a child from the DrawableParent and returns true if
// something was removed
func (e *Entity) RemoveChild(child *Entity) bool {
	if result, err := scene.QueryID(e.ID); err == nil {
		drawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
		drawableParent, ok := drawable.DrawableType.(*DrawableParent)

		if ok {
			for i, c := range drawableParent.Children {
				if c.ID == child.ID {
					drawableParent.Children = append(drawableParent.Children[:i], drawableParent.Children[i+1:]...)
					return true
				}
			}
		}
	}
	return false
}

// RemoveChildren removes all of the children from an entity
func (e *Entity) RemoveChildren() error {
	children, err := e.GetChildren()
	if err != nil {
		return err
	}

	for i := len(children) - 1; i > -1; i-- {
		child := children[i]
		e.RemoveChild(child)
	}

	return nil
}

func (e *Entity) DestroyNested() {
	if result, err := scene.QueryID(e.ID); err == nil {
		drawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
		drawableParent, ok := drawable.DrawableType.(*DrawableParent)
		if !ok {
			e.Destroy()
			return
		}

		for _, child := range drawableParent.Children {
			child.DestroyNested()
		}
		drawableParent.Children = nil
	}
}

// PushChild adds a child to a drawables children list
func (e *Entity) PushChild(child *Entity) (*Entity, error) {
	var err error
	if result, err := scene.QueryID(e.ID); err == nil {
		parentDrawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
		parentMoveable := result.Components[scene.ComponentsMap["moveable"]].(*Moveable)

		if result, err := scene.QueryID(child.ID); err == nil {
			childDrawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)

			childDrawable.IsChild = true

			switch typed := parentDrawable.DrawableType.(type) {
			case *DrawableParent:
				found := false
				for _, c := range typed.Children {
					if c == child {
						found = true
					}
				}
				if !found {
					if parentMoveable.LayoutTag == FlowDirectionHorizontalReversed || parentMoveable.LayoutTag == FlowDirectionVerticalReversed {
						typed.Children = append([]*Entity{child}, typed.Children...)
					} else {
						typed.Children = append(typed.Children, child)
					}
				}
			default:
				panic("Entity doesn't support child elements (make sure to only add children to boxes or scrolls!)")
			}

			switch typed := childDrawable.DrawableType.(type) {
			case *DrawableParent:
				for _, passChild := range typed.Children {
					child.PushChild(passChild)
				}
			}
		}
	}
	return nil, err
}

// FlowChildren aligns the children based on their LayoutTag and alignment
// options
func (e *Entity) FlowChildren() {
	if result, err := scene.QueryID(e.ID); err == nil {
		parentDrawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
		parentMoveable := result.Components[scene.ComponentsMap["moveable"]].(*Moveable)

		children := make([]*Entity, 0, 16)

		switch typed := parentDrawable.DrawableType.(type) {

		case *DrawableParent:
			children = typed.Children
		default:
			panic("Entity doesn't support flowing as it doesn't have child elements (must be a box or scroll!)")
		}

		var fixNested func(e *Entity, parentDrawable *Drawable, parentMoveable *Moveable)
		fixNested = func(e *Entity, parentDrawable *Drawable, parentMoveable *Moveable) {
			var children []*Entity

			switch typed := parentDrawable.DrawableType.(type) {
			case *DrawableParent:
				children = typed.Children
			default:
				return
			}

			var offset rl.Vector2
			for _, child := range children {
				if result, err := scene.QueryID(child.ID); err == nil {
					childDrawable := result.Components[scene.ComponentsMap["drawable"]].(*Drawable)
					childMoveable := result.Components[scene.ComponentsMap["moveable"]].(*Moveable)

					childMoveable.Bounds.X = parentMoveable.Bounds.X
					childMoveable.Bounds.Y = parentMoveable.Bounds.Y

					if parentMoveable.LayoutTag&FlowDirectionVertical == FlowDirectionVertical ||
						parentMoveable.LayoutTag&FlowDirectionVerticalReversed == FlowDirectionVerticalReversed {

						// Wrap
						if offset.Y >= parentMoveable.Bounds.Height {
							offset.Y = 0
							offset.X += childMoveable.Bounds.Width
						}

						childMoveable.Bounds.X += offset.X
						childMoveable.Bounds.Y += offset.Y
						offset.Y += childMoveable.Bounds.Height

					} else if parentMoveable.LayoutTag&FlowDirectionHorizontal == FlowDirectionHorizontal ||
						parentMoveable.LayoutTag&FlowDirectionHorizontalReversed == FlowDirectionHorizontalReversed {

						// Wrap
						if offset.X >= parentMoveable.Bounds.Width {
							offset.X = 0
							offset.Y += childMoveable.Bounds.Height
						}

						childMoveable.Bounds.X += offset.X
						childMoveable.Bounds.Y += offset.Y
						offset.X += childMoveable.Bounds.Width
					}

					// Reset the OrigBounds
					childMoveable.OrigBounds.X = childMoveable.Bounds.X
					childMoveable.OrigBounds.Y = childMoveable.Bounds.Y

					fixNested(child, childDrawable, childMoveable)
				}
			}
		}

		for _, child := range children {
			fixNested(child, parentDrawable, parentMoveable)
		}

	}
}

// NewRenderTexture creates a render texture
func NewRenderTexture(
	bounds rl.Rectangle,
	onMouseUp func(entity *Entity, button rl.MouseButton),
	onMouseDown func(entity *Entity, button rl.MouseButton, isHeld bool),
) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, FlowDirectionHorizontal}).
		AddComponent(hoverable, &Hoverable{Selected: false}).
		AddComponent(interactable, &Interactable{ButtonDown: MouseButtonNone, ButtonReleased: true, OnMouseUp: onMouseUp, OnMouseDown: onMouseDown}).
		AddComponent(drawable, &Drawable{DrawableType: &DrawableRenderTexture{rl.LoadRenderTexture(int(bounds.Width), int(bounds.Height))}})
	e.Name = "buttonTexture"
	return e
}

// NewButtonTexture creates a button which renders a texture
func NewButtonTexture(
	bounds rl.Rectangle,
	texturePath string,
	selected bool,
	onMouseUp func(entity *Entity, button rl.MouseButton),
	onMouseDown func(entity *Entity, button rl.MouseButton, isHeld bool),
) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, FlowDirectionHorizontal}).
		AddComponent(hoverable, &Hoverable{Selected: selected}).
		AddComponent(interactable, &Interactable{ButtonDown: MouseButtonNone, ButtonReleased: true, OnMouseUp: onMouseUp, OnMouseDown: onMouseDown}).
		AddComponent(drawable, &Drawable{DrawableType: NewDrawableTexture(texturePath)})
	e.Name = "buttonTexture"
	return e
}

// NewButtonText creates a button which renders text
func NewButtonText(bounds rl.Rectangle,
	label string,
	selected bool,
	onMouseUp func(entity *Entity, button rl.MouseButton),
	onMouseDown func(entity *Entity, button rl.MouseButton, isHeld bool),
) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, FlowDirectionHorizontal}).
		AddComponent(hoverable, &Hoverable{Selected: selected}).
		AddComponent(interactable, &Interactable{ButtonDown: MouseButtonNone, ButtonReleased: true, OnMouseUp: onMouseUp, OnMouseDown: onMouseDown}).
		AddComponent(drawable, &Drawable{DrawableType: &DrawableText{label}})
	e.Name = "buttonText: " + label
	return e
}

// NewInput creates a button which renders text and can be edited
func NewInput(
	bounds rl.Rectangle,
	label string,
	selected bool,
	onMouseUp func(entity *Entity, button rl.MouseButton),
	onMouseDown func(entity *Entity, button rl.MouseButton, isHeld bool),
	onKeyPress func(entity *Entity, key rl.Key),
) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, FlowDirectionHorizontal}).
		AddComponent(hoverable, &Hoverable{Selected: selected}).
		AddComponent(interactable, &Interactable{ButtonDown: MouseButtonNone, ButtonReleased: true, OnMouseUp: onMouseUp, OnMouseDown: onMouseDown, OnKeyPress: onKeyPress}).
		AddComponent(drawable, &Drawable{DrawableType: &DrawableText{label}})
	e.Name = "buttonText: " + label
	return e
}

// prepareChildren moves children elements etc
func prepareChildren(entity *Entity, children []*Entity) {
	for _, child := range children {
		_, err := entity.PushChild(child)
		if err != nil {
			log.Println(err)
		}
	}
}

// NewBox creates a box which can store children
func NewBox(bounds rl.Rectangle, children []*Entity, flowDirection LayoutTag) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, flowDirection}).
		AddComponent(hoverable, &Hoverable{Selected: false}).
		AddComponent(drawable, &Drawable{DrawableType: &DrawableParent{
			IsPassthrough: true,
			Children:      children,
		}})
	e.Name = "box"
	prepareChildren(e, children)
	e.FlowChildren()
	return e
}

// NewScrollableList creates a box, but it can scroll. Reversed is if the items
// order should be reversed
func NewScrollableList(bounds rl.Rectangle, children []*Entity, flowDirection LayoutTag) *Entity {
	e := scene.NewEntity(nil).
		AddComponent(moveable, &Moveable{bounds, bounds, rl.Vector2{}, flowDirection}).
		AddComponent(hoverable, &Hoverable{Selected: false}).
		AddComponent(scrollable, &Scrollable{}).
		AddComponent(drawable, &Drawable{DrawableType: &DrawableParent{
			IsPassthrough: false,
			Texture:       rl.LoadRenderTexture(int(bounds.Width), int(bounds.Height)),
		}})
	e.Name = "scroll"
	prepareChildren(e, children)
	e.FlowChildren()
	return e
}
