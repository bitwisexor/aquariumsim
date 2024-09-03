package main

import (
	"bytes"
	"flag"
	"image/color"
	_ "image/png"
	"log"
	"math"
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
	bored      bool
	flipCount  int
	lungeCount int
}

type Bubble struct {
	x, y  float64
	vy    float64
	scale float64
}

type Seaweed struct {
	x, y float64
}

type Game struct {
	cameraX  int
	cameraY  int
	bubbleCD int
	bubbles  []*Bubble
	weeds    []*Seaweed
	fishes   []*Fish
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
	screenWidth  = 340
	screenHeight = 224
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

func (f *Fish) hitbox() (left, right, top, bottom float64) {
	width := float64(fishImage.Bounds().Dx()) * 2
	height := float64(fishImage.Bounds().Dy()) * 2
	left = f.x
	right = f.x + width
	top = f.y
	bottom = f.y + height
	return
}

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

	img, _, err = ebitenutil.NewImageFromFile("./altbubble1.png")
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

	if g.bubbleCD > 0 {
		g.bubbleCD--
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.bubbleCD <= 0 {
		x, y := ebiten.CursorPosition()
		g.spawnBubbleAt(float64(x), float64(y))
		g.bubbleCD = 40 // Frames to wait before spawning
	}

	for _, bubble := range g.bubbles {
		bubble.y += bubble.vy
	}

	g.giveChase() // Fish chases bubbles

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op_fish, op_sw := &ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}
	op_fish.GeoM.Scale(2, 2)
	op_sw.GeoM.Scale(-1, -1)

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

	// Draw bottom / seabed
	seabedWidth := seabedImage.Bounds().Dx()

	for x := 0; x < screenWidth; x += seabedWidth {
		op_sw.GeoM.Reset()
		op_sw.GeoM.Translate(float64(x), float64(screenHeight-seabedImage.Bounds().Dy()))
		screen.DrawImage(seabedImage, op_sw)
	}

	for _, seaweed := range g.weeds {
		op_sw.GeoM.Reset()
		op_sw.GeoM.Translate(seaweed.x, seaweed.y)
		screen.DrawImage(seaweedImage, op_sw)
	}

	for _, bubble := range g.bubbles {
		op_bub := &ebiten.DrawImageOptions{}
		op_bub.GeoM.Scale(bubble.scale, bubble.scale)
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
	seabedHeight := float64(seabedImage.Bounds().Dy())

	// Introduce a chance for the fish to lunge
	if fish.lungeCount <= 0 && rand.Float64() < 0.02 {
		fish.ax = (rand.Float64() - 0.7) * 0.008 // Lunge with a stronger acceleration
		fish.ay = (rand.Float64() - 0.9) * 0.005 // Reduced vertical movement during lunge
		fish.lungeCount = rand.Intn(4) + 1       // Set lunge duration
	} else {
		fish.ax = (rand.Float64() - 0.5) * 0.01  // Normal slight acceleration
		fish.ay = (rand.Float64() - 0.5) * 0.009 // Reduced vertical movement
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
	if fish.vx > 0.1 && fish.flipped {
		fish.flipped = false
	} else if fish.vx < -0.1 && !fish.flipped {
		fish.flipped = true
	}

	// Boundary check to keep the fish within the screen
	if fish.x < 0 {
		fish.x = 0
		fish.vx = -fish.vx
	} else if fish.x > screenWidth-30 {
		fish.x = screenWidth - 30
		fish.vx = -fish.vx
	}
	if fish.y < 0 {
		fish.y = 0
		fish.vy = -fish.vy
	} else if fish.y > screenHeight-seabedHeight {
		fish.y = screenHeight - seabedHeight
		fish.vy = -fish.vy
	}
}

func (g *Game) checkCollisions() {
}

func (g *Game) giveChase() { // Fish gives chase of bubble objects
	fish := g.fishes[0]
	if len(g.bubbles) == 0 {
		fish.ax = (rand.Float64() - 0.5) * 0.01
		fish.ay = (rand.Float64() - 0.5) * 0.009
		return // If no bubbles
	}

	closestBub := g.bubbles[0]
	minDistance := distance(fish.x, fish.y, closestBub.x, closestBub.y)

	// Find closest bubble
	for _, bubble := range g.bubbles {
		dist := distance(fish.x, fish.y, bubble.x, bubble.y)
		if dist < minDistance {
			minDistance = dist
			closestBub = bubble
		}
	}

	if closestBub.y < 0 || closestBub.y > screenHeight || closestBub.x < 0 || closestBub.x > screenWidth {
		fish.ax = (rand.Float64() - 0.5) * 0.01
		fish.ay = (rand.Float64() - 0.5) * 0.009
		return
	}

	fish.ax = (closestBub.x - fish.x) * 0.0001
	fish.ay = (closestBub.y - fish.y) * 0.0001

	fish.vx += fish.ax
	fish.vy += fish.ay

	fish.x += fish.vx
	fish.y += fish.vy
}

func distance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

func (g *Game) spawnWeeds() {
	num_spawn := rand.Intn(11) + 20
	for i := 0; i < num_spawn; i++ {
		sw := &Seaweed{
			x: rand.Float64() * screenWidth,
			y: float64(screenHeight - seaweedImage.Bounds().Dy()),
		}
		g.weeds = append(g.weeds, sw)
	}
}

func (g *Game) spawnBubbleAt(x, y float64) {
	scale := rand.Float64()*0.5 + 0.5

	bu := &Bubble{
		x:     x,
		y:     y,
		vy:    -0.3,
		scale: scale,
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
