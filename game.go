package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"strings"
	"time"

	"embed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	gameWidth         = 480
	gameHeight        = 800
	defaultJumpHeight = 40
	maxLevels         = 5
)

var (
	//go:embed typewriter.ttf
	fs            embed.FS
	fontFace      font.Face
	hudFontFace   font.Face
	titleFontFace font.Face
)

func init() {
	fontData, err := fs.ReadFile("typewriter.ttf")
	if err != nil {
		log.Fatal(err)
	}

	fnt, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72

	fontFace, err = opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}

	hudFontFace, err = opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    20,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}

	titleFontFace, err = opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    48,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

type Level struct {
	minSpeed   float64
	maxSpeed   float64
	textLength int
	waveLength int
}

type Game struct {
	enemies      []*Enemy
	waveCount    int
	currentLevel int
	levels       []*Level
	redlineY     float32
	gameOver     bool
	gameWon      bool
	started      bool
	gameOverText string
	winText      string
	keys         []ebiten.Key
	keyMap       map[ebiten.Key]string
}

type Enemy struct {
	game        *Game
	text        string
	xPos, yPos  float64
	discreteY   int
	jumpHeight  int
	speed       float64
	killed      bool
	textArray   []string
	partialText string
}

var letters = "abcdefghijklmnopqrstuvwxyz"

func generateText(l int) string {
	var result string

	for i := 0; i < l; i++ {
		idx := rand.Intn(len(letters) - 1)
		result += string(letters[idx])
	}

	return result
}

func reverse(s []string) []string {
	r := make([]string, len(s))
	copy(r, s)

	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}

	return r
}

func (g *Game) AddEnemy(xPos float64) {
	e := &Enemy{
		game:   g,
		xPos:   xPos,
		killed: false,
	}

	g.enemies = append(g.enemies, e)

	for _, e := range g.enemies {
		tl := g.levels[g.currentLevel].textLength
		min := g.levels[g.currentLevel].minSpeed
		max := g.levels[g.currentLevel].maxSpeed
		e.Reset(generateText(tl), rand.Float64()*(max-min)+min, defaultJumpHeight)
	}
}

func (e *Enemy) Reset(text string, speed float64, jumpHeight int) {
	e.text = text
	e.yPos = 40
	e.discreteY = 40
	e.jumpHeight = jumpHeight
	e.speed = speed
	e.killed = false
	e.textArray = reverse(strings.Split(text, ""))
}

func (e *Enemy) checkDamage(l string) {
	if len(e.textArray) == 0 {
		e.killed = true

		return
	}

	if e.textArray[0] == l {
		e.textArray = e.textArray[1:]
	}
}

func (e *Enemy) Update() {
	if len(e.game.keys) > 0 {
		l, ok := e.game.keyMap[e.game.keys[0]]
		if ok {
			e.checkDamage(l)
		}
	}

	if e.killed {
		tl := e.game.levels[e.game.currentLevel].textLength
		min := e.game.levels[e.game.currentLevel].minSpeed
		max := e.game.levels[e.game.currentLevel].maxSpeed
		e.game.waveCount -= 1

		if e.game.waveCount > 0 {
			e.Reset(generateText(tl), rand.Float64()*(max-min)+min, defaultJumpHeight)
		}
	}

	e.partialText = strings.Join(reverse(e.textArray), "")
	e.yPos += e.speed
	e.discreteY = int(e.yPos) / e.jumpHeight * e.jumpHeight
}

func (e *Enemy) CheckGameOver(lineY float64) bool {
	return e.yPos >= lineY
}

func (e *Enemy) Draw(screen *ebiten.Image) {
	text.Draw(screen, e.text, fontFace, int(e.xPos), e.discreteY, color.RGBA{212, 212, 212, 255})
	text.Draw(screen, e.partialText, fontFace, int(e.xPos), e.discreteY, color.RGBA{0, 0, 0, 255})
}

func (g *Game) Update() error {
	if !g.gameOver {
		if !g.started {
			if ebiten.IsKeyPressed(ebiten.KeyEnter) {
				g.started = true
			}
		} else {
			if g.waveCount < 0 {
				g.currentLevel += 1

				if g.currentLevel >= len(g.levels) {
					g.gameOver = true
					g.gameWon = true
				} else {
					g.waveCount = g.levels[g.currentLevel].waveLength
				}
			}

			if !g.gameOver {
				g.keys = inpututil.AppendPressedKeys(g.keys[:0])

				for _, e := range g.enemies {
					if e.CheckGameOver(float64(g.redlineY)) {
						g.gameOver = true
						break
					}

					e.Update()
				}
			}
		}
	} else {
		g.currentLevel = 0
		g.waveCount = g.levels[g.currentLevel].waveLength

		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			g.gameOver = false
			g.enemies = make([]*Enemy, 0)
			g.AddEnemy(60)
			g.AddEnemy(300)
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{255, 255, 255, 255})

	if g.gameOver {
		t := g.gameOverText
		if g.gameWon {
			t = g.winText
		}

		text.Draw(screen, t, fontFace, 120, 240, color.RGBA{0, 0, 0, 255})
		text.Draw(screen, "press <Enter> to restart", fontFace, 20, 360, color.RGBA{0, 0, 0, 255})
	} else {
		if g.started {
			for _, e := range g.enemies {
				e.Draw(screen)
			}

			vector.StrokeLine(screen, 0, g.redlineY, gameWidth, g.redlineY, 1.0, color.RGBA{255, 0, 0, 255}, true)
			text.Draw(screen, fmt.Sprintf("level: %d", g.currentLevel+1), hudFontFace, 10, 730, color.RGBA{0, 0, 0, 255})
			text.Draw(screen, fmt.Sprintf("strings left: %d", g.waveCount), hudFontFace, 10, 780, color.RGBA{0, 0, 0, 255})
		} else {
			text.Draw(screen, "REVERSE TYPIST", titleFontFace, 20, 100, color.RGBA{0, 0, 0, 255})
			text.Draw(screen, "RULES", hudFontFace, 200, 300, color.RGBA{0, 0, 0, 255})
			text.Draw(screen, "1. Type strings in reverse order", hudFontFace, 20, 340, color.RGBA{0, 0, 0, 255})
			text.Draw(screen, "2. Strings must not cross red line", hudFontFace, 20, 380, color.RGBA{0, 0, 0, 255})
			text.Draw(screen, "press <Enter> to start", hudFontFace, 100, 730, color.RGBA{0, 0, 0, 255})
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return gameWidth, gameHeight
}

func main() {
	ebiten.SetWindowSize(gameWidth, gameHeight)
	ebiten.SetWindowTitle("Reverse Typist")

	levels := make([]*Level, maxLevels)
	initTxtLen := 3
	initWaveLen := 10

	for i := 0; i < maxLevels; i++ {
		levels[i] = &Level{
			minSpeed:   0.5,
			maxSpeed:   0.8 + float64(i)*0.1,
			textLength: initTxtLen + i,
			waveLength: initWaveLen + i*5,
		}
	}

	curLevel := 0
	waveCount := levels[curLevel].waveLength

	keyMap := make(map[ebiten.Key]string)

	for i := 0; i < len(letters); i++ {
		keyMap[ebiten.Key(i)] = string(letters[i])
	}

	g := &Game{
		levels:       levels,
		currentLevel: curLevel,
		waveCount:    waveCount,
		redlineY:     700,
		winText:      "you won",
		gameOverText: "game over",
		keyMap:       keyMap,
	}

	g.AddEnemy(60)
	g.AddEnemy(300)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
