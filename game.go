package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gopkg.in/validator.v2"
)

func NewGame(cfg GameConfig) (*Game, error) {
	// validate config
	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %+v", cfg)
	}
	// load exit button image
	exitButtonImage, _, err := ebitenutil.NewImageFromFile(cfg.ExitButtonImagePath)
	if err != nil {
		return nil, fmt.Errorf("unable to load exit button image due: %w", err)
	}
	// load actions
	actions := make([]Action, 0, len(cfg.ActionSources))
	for _, actSrc := range cfg.ActionSources {
		acts, err := actSrc.ToActions()
		if err != nil {
			return nil, fmt.Errorf("unable to convert source into action due: %w", err)
		}
		actions = append(actions, acts...)
	}
	// adjust window properties
	ebiten.SetWindowSize(cfg.ScreenDimension.Width, cfg.ScreenDimension.Height)
	ebiten.SetWindowDecorated(false)
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowFloating(true)
	// determine window start position, we start on center of the main display
	maxScreenWidth, maxScreenHeight := ebiten.ScreenSizeInFullscreen()
	windowPos := Point{
		X: (maxScreenWidth / 2) - cfg.ScreenDimension.Width,
		Y: (maxScreenHeight / 2) - cfg.ScreenDimension.Height,
	}

	// initialize game
	g := &Game{
		actions:         actions,
		currActionIdx:   0,
		exitButtonImage: exitButtonImage,
		windowPos:       windowPos,
		screenDimension: cfg.ScreenDimension,
	}

	return g, nil
}

type GameConfig struct {
	ActionSources       []ActionSource `validate:"min=1"`
	ExitButtonImagePath string         `validate:"min=1"`
	ScreenDimension     Dimension      `validate:"nonzero"`
}

func (c GameConfig) Validate() error {
	return validator.Validate(c)
}

type Game struct {
	actions          []Action
	currActionIdx    int
	exitButtonImage  *ebiten.Image
	displayImgTick   int
	windowPos        Point
	lastLeftClickPos Point
	screenDimension  Dimension
}

func (g *Game) Update() error {
	// get current cursor position
	cursorX, cursorY := ebiten.CursorPosition()
	cursorPos := Point{X: cursorX, Y: cursorY}
	// increment display image tick
	g.incrDisplayImgTick()
	// check whether user click the exit button
	g.handleExitIfNecessary(cursorPos)
	// update window position
	g.updateWindowPosOnLeftClick(cursorPos)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// set window position according to calculation
	ebiten.SetWindowPosition(g.windowPos.X, g.windowPos.Y)
	// draw character image
	screen.DrawImage(g.getDisplayImage(), nil)
	// draw exit button, we want to position it on top right
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(float64(g.screenDimension.Width-g.exitButtonImage.Bounds().Dx()), 16)
	screen.DrawImage(g.exitButtonImage, opt)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) incrDisplayImgTick() {
	g.displayImgTick++
}

func (g *Game) getDisplayImage() *ebiten.Image {
	currAction := g.actions[g.currActionIdx]
	imgIdx := (g.displayImgTick / 40) % len(currAction.Images)
	animLoopCount := (g.displayImgTick / 40) / len(currAction.Images)
	defer func() {
		// if animation loop has finished, determine next action
		if imgIdx == 0 && animLoopCount > 0 {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			g.currActionIdx = r.Intn(len(g.actions))
			g.displayImgTick = 0
		}
	}()
	return &currAction.Images[imgIdx]
}

func (g *Game) handleExitIfNecessary(cursorPos Point) {
	// check if the cursor position is above exit button
	isAboveButton := g.isCursorAboveExitButton(cursorPos)
	if isAboveButton && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// when the cursor is above exit button & user click it, this means user
		// want to exit the program, so we do it.
		os.Exit(0)
	}
}

func (g *Game) isCursorAboveExitButton(cursorPos Point) bool {
	btnDimension := Dimension{
		Width:  g.exitButtonImage.Bounds().Dx(),
		Height: g.exitButtonImage.Bounds().Dy(),
	}
	return cursorPos.X >= (g.screenDimension.Width-btnDimension.Width) &&
		cursorPos.X <= g.screenDimension.Width &&
		cursorPos.Y >= 16 && cursorPos.Y <= btnDimension.Height+16
}

func (g *Game) updateWindowPosOnLeftClick(cursorPos Point) {
	// if left mouse button is clicked, this means user is currently trying to drag
	// the cat, this means we need to make the window position follow this with some
	// adjustments.
	isLeftClick := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if isLeftClick {
		// get current position of the mouse cursor, in ebitengine we could only get
		// cursor position relative to the game window, so later we need some adjustment
		// to this cursor position since what we need is actual mouse cursor position
		isJustPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		if isJustPressed {
			// record the position of left click when it is just happening, this is useful
			// for our window position final adjustment
			g.lastLeftClickPos = cursorPos
		}

		// get actual position of cursor
		curWindowX, curWindowY := ebiten.WindowPosition()
		actualCursorPos := Point{
			X: curWindowX + cursorPos.X,
			Y: curWindowY + cursorPos.Y,
		}

		// update window position to follow actual cursor position with some adjustment
		g.windowPos.X = actualCursorPos.X - g.lastLeftClickPos.X
		g.windowPos.Y = actualCursorPos.Y - g.lastLeftClickPos.Y
	}
}