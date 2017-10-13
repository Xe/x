package main

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	nj := sdl.NumJoysticks()
	log.Printf("%d joysticks detected", nj)

	for i := 0; i < nj; i++ {
		j := sdl.JoystickOpen(sdl.JoystickID(i))
		defer j.Close()

		log.Printf("%d joystick name: %s, %d buttons, %d axes, %d hats", i, j.Name(), j.NumButtons(), j.NumAxes(), j.NumHats())

		for ii := 0; ii < j.NumButtons(); ii++ {
			log.Printf("%d joystick button %d: %v", i, ii, j.GetButton(ii))
		}

		for ii := 0; ii < j.NumAxes(); ii++ {
			log.Printf("%d joystick axis %d: %v", i, ii, j.GetAxis(ii))
		}

		for ii := 0; ii < j.NumHats(); ii++ {
			log.Printf("%d joystick hat %d: %v", i, ii, j.GetHat(ii))
		}
	}
}
