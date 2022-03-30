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
	lines := []string{
		fmt.Sprintf("emu fps: %.0f", g.mainCore.GBA().Sync().FPSTarget()),
		fmt.Sprintf("fps:     %.0f", ebiten.CurrentFPS()),
	}

	match := g.Match()
	if match != nil {
		battle := match.Battle()
		if battle != nil {
			lines = append(lines,
				fmt.Sprintf("ping:    %s", match.MedianDelay()),
				fmt.Sprintf("is p2:   %t", battle.IsP2()),
				fmt.Sprintf("lag:     %2d (max %2d)", battle.Lag(), match.RunaheadTicksAllowed()),
			)
		}
	}
	text.Draw(screen, strings.Join(lines, "\n"), mplusNormalFont, 2, 14, color.RGBA{0xff, 0x00, 0xff, 0xff})
}
