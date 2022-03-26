package game

import (
	"fmt"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	mplusNormalFont font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (g *Game) spewDebug(screen *ebiten.Image) {
	g.matchMu.Lock()
	defer g.matchMu.Unlock()

	lines := []string{
		fmt.Sprintf("emu fps: %.0f", g.mainCore.GBA().Sync().FPSTarget()),
		fmt.Sprintf("fps:     %.0f", ebiten.CurrentFPS()),
		fmt.Sprintf("ping:    %s", g.medianDelay()),
	}
	if g.match != nil && g.match.battle != nil {
		lines = append(lines, fmt.Sprintf("is p2:   %t", g.match.battle.isP2))
	}
	text.Draw(screen, strings.Join(lines, "\n"), mplusNormalFont, 2, 14, color.RGBA{0x00, 0xff, 0x00, 0xff})
}
