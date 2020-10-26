package main

import (
	"bufio"
	"fmt"
	"github.com/eiannone/keyboard"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

const TIE = "|-o-|"
const TIE_FWD = "/-o-/"
const TIE_BWD = "\\-o-\\"
const TICK_TIMEOUT = 250

type Face int

const (
	Forward Face = iota
	Backward
	Up
	Down
	Idle
)

type Position struct {
	X int
	Y int
}

type Screen struct {
	Width int
	Height int
	Rows []string
	Position Position
	Buffer io.Writer
	Sprite string
}

func keyWatch(ch chan<- *keyboard.Key) {
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for {
		_, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}
		ch <- &key
		if key == keyboard.KeyEsc {
			close(ch)
		}
	}
}

func keyListen(ch chan<- *keyboard.Key) {
	for {
		ch <- nil
		time.Sleep(time.Millisecond * TICK_TIMEOUT)
	}
}

func updateScreenWithPosition(ch <-chan Face, screen Screen) {
	updateScreen(screen)
	horizontalRatio := int(math.Round(math.Max(float64(screen.Height), float64(screen.Width)) / math.Min(float64(screen.Height), float64(screen.Width))))
	for face := range ch {
		switch face {
		case Forward:
			if screen.Position.X + len(screen.Sprite) + horizontalRatio > screen.Width {
				screen.Position.X = 0
			} else {
				screen.Position.X += horizontalRatio
			}
			screen.Sprite = TIE_FWD
			updateScreen(screen)
		case Backward:
			if screen.Position.X - horizontalRatio < 0 {
				screen.Position.X = screen.Width - len(screen.Sprite)
			} else {
				screen.Position.X -= horizontalRatio
			}
			screen.Sprite = TIE_BWD
			updateScreen(screen)
		case Up:
			if screen.Position.Y - 1 < 0 {
				screen.Position.Y = screen.Height - 1
			} else {
				screen.Position.Y -= 1
			}
			screen.Sprite = TIE
			updateScreen(screen)
		case Down:
			if screen.Position.Y + 1 > screen.Height {
				screen.Position.Y = 0
			} else {
				screen.Position.Y += 1
			}
			screen.Sprite = TIE
			updateScreen(screen)
		case Idle:
			screen.Sprite = TIE
			updateScreen(screen)
		default:
			screen.Sprite = TIE
			updateScreen(screen)
		}
	}
}

func updateScreen(screen Screen) {
	for i := range screen.Rows {
		screen.Rows[i] = strings.Repeat(" ", screen.Width)
	}

	for i := 0; i < len(screen.Rows); i++ {
		if i == screen.Position.Y {
			str := strings.Repeat(" ", screen.Position.X)
			str += screen.Sprite
			str += strings.Repeat(" ", screen.Width - len(screen.Sprite) - screen.Position.X)
			screen.Rows[i] = str
		} else {
			screen.Rows[i] = strings.Repeat(" ", screen.Width)
		}
	}

	blitScreen(screen)
}

func blitScreen(screen Screen) {
	fmt.Printf("\u001b[%dD", screen.Height)
	fmt.Printf("\u001b[%dA", screen.Width)
	for i := 0; i < screen.Height; i++ {
		fmt.Println(screen.Rows[i])
	}
}

func getTermSize() Screen {
	width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	return Screen{
		Width: width,
		Height: height - 1,
	}
}

func getInitialPosition(screen Screen) Position {
	horizontal := int(math.Round(float64((screen.Width - len(TIE)) / 2)))
	vertical := int(math.Round(float64(screen.Height / 2)))
	return Position{
		X: horizontal,
		Y: vertical,
	}
}

func prepScreen() Screen {
	screen := getTermSize()
	initialPosition := getInitialPosition(screen)
	var rows []string

	for i := 0; i < screen.Height; i++ {
		if i == initialPosition.Y {
			str := strings.Repeat(" ", initialPosition.X)
			str += TIE
			str += strings.Repeat(" ", screen.Width - len(TIE) - initialPosition.X)
			rows = append(rows, str)
		} else {
			rows = append(rows, strings.Repeat(" ", screen.Width))
		}
	}

	screen.Rows = rows
	screen.Position = initialPosition
	screen.Buffer = bufio.NewWriter(os.Stdout)
	screen.Sprite = TIE

	return screen
}

func mainLoop(ch <-chan *keyboard.Key) {
	posChan := make(chan Face)
	screen := prepScreen()
	go updateScreenWithPosition(posChan, screen)

	lastTick := time.Now()
	for key := range ch {
		if key == nil {
			if time.Since(lastTick).Milliseconds() >= TICK_TIMEOUT {
				posChan <- Idle
			}
		} else if *key == keyboard.KeyArrowLeft {
			posChan <- Backward
		} else if *key == keyboard.KeyArrowRight {
			posChan <- Forward
		} else if *key == keyboard.KeyArrowUp {
			posChan <- Up
		} else if *key == keyboard.KeyArrowDown {
			posChan <- Down
		} else if *key == keyboard.KeyEsc {
			close(posChan)
			break
		} else {
			posChan <- Idle
		}
		lastTick = time.Now()
	}
}



func main()  {
	keyChan := make(chan *keyboard.Key)
	go keyWatch(keyChan)
	go keyListen(keyChan)
	mainLoop(keyChan)
}
