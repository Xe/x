package main

import (
	"context"
	"log"
	"time"

	"github.com/Xe/x/sdl/joypad"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	// XXX: ASSUMPTION: only the 8bitdo zero controller is connected and it is id 0
	if sdl.NumJoysticks() != 1 {
		log.Fatal("please make sure the 8bitdo zero is the only controller connected.")
	}

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	rect := sdl.Rect{0, 0, 800, 600}
	surface.FillRect(&rect, 0xffff0000)
	window.UpdateSurface()

	j := sdl.JoystickOpen(sdl.JoystickID(0))
	defer j.Close()
	ez := joypad.NewEightBitdoZero(j)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loopInputs(ctx, ez)
}

func loopInputs(ctx context.Context, gp joypad.Gamepad) {
	t := time.NewTicker(time.Second / 60)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l, r := gp.ShoulderButtons()

			if l {
				log.Println("switch to virtual desktop to the left")
				continue
			}

			if r {
				log.Println("switch to virtual desktop to the right")
				continue
			}
		}
	}
}
