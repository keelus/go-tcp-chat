package main

import (
	"go-tcp-chat/common"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type ScreenUI struct {
	Screen tcell.Screen
	Prompt string
}

func (ui *ScreenUI) Draw(broadcastBuffer []common.Broadcast) {
	ui.drawChat(broadcastBuffer)
	ui.drawSeparator()
	ui.drawPrompt(ui.Prompt)
	ui.Screen.Show()
	ui.Screen.Clear()
}

func (ui *ScreenUI) EmitStr(x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		ui.Screen.SetContent(x, y, c, comb, style)
		x += w
	}
}

func (ui *ScreenUI) drawPrompt(msg string) {
	_, h := ui.Screen.Size()
	ui.EmitStr(0, h-1, tcell.StyleDefault, msg)
}

func (ui *ScreenUI) drawSeparator() {
	w, h := ui.Screen.Size()
	separator := ""
	for i := 0; i < w; i++ {
		separator = separator + "="
	}
	ui.EmitStr(0, h-2, tcell.StyleDefault, separator)
}

func (ui *ScreenUI) drawChat(broadcastBuffer []common.Broadcast) {
	_, maxEntries := ui.Screen.Size()
	maxEntries -= 2

	entriesFrom := int(math.Max(0, float64(len(broadcastBuffer)-maxEntries)))
	entriesTo := len(broadcastBuffer)

	entriesDrawn := 0
	for i := entriesFrom; i < entriesTo; i++ {
		if entriesDrawn == maxEntries {
			break
		}
		renderedBroadcast, tcellStyle := broadcastBuffer[i].RenderBroadcast()
		ui.EmitStr(0, entriesDrawn, tcellStyle, renderedBroadcast)
		entriesDrawn += 1
	}
}
