package main

import (
	"bytes"
	"flag"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
  "github.com/aquilax/go-perlin"
)

type Bubble struct {
	x, y float64
	vy   float64
}

type Seaweed struct {
	x, y float64
}

type Game struct {
	cameraX int
	cameraY int
	bubbles []*Bubble // Slice to hold multiple bubbles
	weeds   []*Seaweed
}

var (
 flagCRT = flag.Bool("crt", false, "enable CRT effects")
 crtGo []byte

)
const (
	screenWidth  = 640
	screenHeight = 480
	maxAngle     = 256
	fontSize     = 34
)

var (
	fishImage        *ebiten.Image
	bubblesImage     *ebiten.Image
	arcadeFaceSource *text.GoTextFaceSource
	bkgImage         *ebiten.Image
	seaweedImage     *ebiten.Image
)

func init() { // Main menu text
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))
	if err != nil {
		log.Fatal(err)
	}
	arcadeFaceSource = s
}

func init() { // load image files into variables
	img, _, err := ebitenutil.NewImageFromFile("./mirroredfish.png")
	if err != nil {
		log.Fatal(err)
	}
	fishImage = ebiten.NewImageFromImage(img)

	img, _, err = ebitenutil.NewImageFromFile("./seaweed.png")
	if err != nil {
		log.Fatal(err)
	}
	seaweedImage = ebiten.NewImageFromImage(img)

	img, _, err = ebitenutil.NewImageFromFile("./bulbous.png")
	if err != nil {
		log.Fatal(err)
	}
	bubblesImage = ebiten.NewImageFromImage(img)

	img = ebiten.NewImage(screenWidth, screenHeight)
	bkgImage = img

}

func (g *Game) init() {
	g.cameraX = -240
	g.cameraY = 0
	g.bubbles = []*Bubble{} // Initialize bubble slice/
	g.weeds = []*Seaweed{}

	g.spawnWeeds()
}

func NewGame(crt bool) ebiten.Game {
	g := &Game{}
	g.init() // Initialize game state

	if crt {
		return &GameWithCRTEffect{Game: g}
	}

	return g
}

func (g *Game) Update() error {
	// Spawn a new bubble at random intervals
	if rand.Intn(60) == 0 { // Approximately once per second at 60 FPS
		g.spawnBubble()
	}

	// Update the position of each bubble
	for _, bubble := range g.bubbles {
		bubble.y += bubble.vy
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op_fish, op_bub, op_sw := &ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}
	op_fish.GeoM.Scale(3, 3)
	op_bub.GeoM.Scale(2, 2) // Adjust the scale as needed

	plBlue := color.RGBA{R: 65, G: 105, B: 225, A: 255} // Current blue

	screenWidth, screenHeight := screen.Size() //used for calculating position
	spriteWidth, spriteHeight := fishImage.Size()

	x := (screenWidth - spriteWidth*3) / 2
	y := (screenHeight - spriteHeight*3) / 2

	op_fish.GeoM.Translate(float64(x), float64(y))

	bkgImage.Fill(plBlue) // Fill background

	screen.DrawImage(bkgImage, nil)      // Draw the background
	screen.DrawImage(fishImage, op_fish) // Draw the fish

	for _, seaweed := range g.weeds {
		op_sw.GeoM.Reset()
		op_sw.GeoM.Translate(seaweed.x, seaweed.y)
		screen.DrawImage(seaweedImage, op_sw) // Draw the seaweed using its options
	}

	// Draw bubbles
	for _, bubble := range g.bubbles {
		op_bub.GeoM.Reset()
		op_bub.GeoM.Translate(bubble.x, bubble.y)
		screen.DrawImage(bubblesImage, op_bub)
	}
}

func (g *Game) Layout(screenWidth, screenHeight int) (int, int) {
	return screenWidth, screenHeight
}

type GameWithCRTEffect struct {
	ebiten.Game

	crtShader *ebiten.Shader
}

func (g *Game) spawnWeeds() {
	num_spawn := rand.Intn(21) + 20
	for i := 0; i < num_spawn; i++ {
		sw := &Seaweed{
			x: rand.Float64() * screenWidth,
			y: float64(screenHeight - seaweedImage.Bounds().Dy()), // Position at the bottom
		}
		g.weeds = append(g.weeds, sw)
	}
}

func (g *Game) spawnBubble() {
	bu := &Bubble{
		x:  rand.Float64() * screenWidth,
		y:  screenHeight,
		vy: -0.15, // Negative vy makes the bubble move upwards
	}
	g.bubbles = append(g.bubbles, bu)
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Aquarium")
	if err := ebiten.RunGame(NewGame(*flagCRT)); err != nil {
		panic(err)
	}
}
