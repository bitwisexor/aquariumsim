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
)

type Fish struct {
	x, y       float64
	vx, vy     float64
	ax, ay     float64
	flipped    bool
	flipCount  int
	lungeCount int
}

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
	bubbles []*Bubble
	weeds   []*Seaweed
	fishes  []*Fish
}

type GameWithCRTEffect struct {
	ebiten.Game

	crtShader *ebiten.Shader
}

var (
	flagCRT   = flag.Bool("crt", true, "enable CRT effects")
	crtGo     []byte
	crtShader *ebiten.Shader
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
	seabedImage      *ebiten.Image
)

func init() {
	// Main menu text
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))
	if err != nil {
		log.Fatal(err)
	}
	arcadeFaceSource = s

	shaderCode := []byte(`
    package main

    func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
        if int(mod(position.y, 2.0)) == 0 {
            color.rgb *= 0.6;
        }
        return imageSrc0At(texCoord) * color;
    }
`)
	crtShader, err = ebiten.NewShader(shaderCode)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
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

	img, _, err = ebitenutil.NewImageFromFile("seabed.png")
	if err != nil {
		log.Fatal(err)
	}
	seabedImage = ebiten.NewImageFromImage(img)

	img = ebiten.NewImage(screenWidth, screenHeight)
	bkgImage = img
}

func (g *Game) init() {
	g.cameraX = -240
	g.cameraY = 0
	g.bubbles = []*Bubble{}
	g.weeds = []*Seaweed{}
	g.fishes = []*Fish{
		{
			x:  float64(screenWidth) / 2,
			y:  float64(screenHeight) / 2,
			vx: 0,
			vy: 0,
		},
	}
	g.spawnWeeds()
}

func NewGame(crt bool) ebiten.Game {
	g := &Game{}
	g.init()

	if crt {
		return &GameWithCRTEffect{Game: g}
	}

	return g
}

func (g *Game) Update() error {
	g.randomWalk()
	if rand.Intn(60) == 0 {
		g.spawnBubble()
	}

	for _, bubble := range g.bubbles {
		bubble.y += bubble.vy
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op_fish, op_bub, op_sw := &ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}
	op_fish.GeoM.Scale(3, 3)

	fish := g.fishes[0]

	if fish.flipped {
		op_fish.GeoM.Scale(-1, 1)                                   // Flip the image horizontally
		op_fish.GeoM.Translate(float64(fishImage.Bounds().Dx()), 0) // Adjust position after flip
	}

	op_fish.GeoM.Translate(fish.x, fish.y)

	plBlue := color.RGBA{R: 65, G: 105, B: 225, A: 255}
	bkgImage.Fill(plBlue)

	screen.DrawImage(bkgImage, nil)
	screen.DrawImage(fishImage, op_fish)

	for _, seaweed := range g.weeds {
		op_sw.GeoM.Reset()
		op_sw.GeoM.Translate(seaweed.x, seaweed.y)
		screen.DrawImage(seaweedImage, op_sw)
	}

	for _, bubble := range g.bubbles {
		op_bub.GeoM.Reset()
		op_bub.GeoM.Translate(bubble.x, bubble.y)
		screen.DrawImage(bubblesImage, op_bub)
	}
}

func (g *Game) Layout(screenWidth, screenHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *GameWithCRTEffect) Draw(screen *ebiten.Image) {
	img := ebiten.NewImage(screenWidth, screenHeight)
	g.Game.Draw(img)

	options := ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{img},
	}

	screen.DrawRectShader(screenWidth, screenHeight, crtShader, &options)
}

func (g *Game) randomWalk() {
	fish := g.fishes[0]

	// Introduce a chance for the fish to lunge
	if fish.lungeCount <= 0 && rand.Float64() < 0.05 {
		fish.ax = (rand.Float64() - 0.5) * 4.0 // Lunge with a stronger acceleration
		fish.ay = (rand.Float64() - 0.5) * 4.0
		fish.lungeCount = 20 // Set lunge duration
	} else {
		fish.ax = (rand.Float64() - 0.5) * 0.1 // Normal slight acceleration
		fish.ay = (rand.Float64() - 0.5) * 0.1
	}

	// Apply acceleration to velocity
	fish.vx += fish.ax
	fish.vy += fish.ay

	// Update position with the new velocity
	fish.x += fish.vx
	fish.y += fish.vy

	// Decrease the lunge count if active
	if fish.lungeCount > 0 {
		fish.lungeCount--
	}

	// Flip the fish only if the movement is significantly different and sustained
	if fish.vx > 0.2 && fish.flipped {
		fish.flipped = false
	} else if fish.vx < -0.2 && !fish.flipped {
		fish.flipped = true
	}

	// Boundary check to keep the fish within the screen
	if fish.x < 0 {
		fish.x = 0
		fish.vx = -fish.vx
	} else if fish.x > screenWidth {
		fish.x = screenWidth
		fish.vx = -fish.vx
	}
	if fish.y < 0 {
		fish.y = 0
		fish.vy = -fish.vy
	} else if fish.y > screenHeight {
		fish.y = screenHeight
		fish.vy = -fish.vy
	}
}

func (g *Game) spawnWeeds() {
	num_spawn := rand.Intn(21) + 20
	for i := 0; i < num_spawn; i++ {
		sw := &Seaweed{
			x: rand.Float64() * screenWidth,
			y: float64(screenHeight - seaweedImage.Bounds().Dy()),
		}
		g.weeds = append(g.weeds, sw)
	}
}

func (g *Game) spawnBubble() {
	bu := &Bubble{
		x:  rand.Float64() * screenWidth,
		y:  screenHeight,
		vy: -0.15,
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
