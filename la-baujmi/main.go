package main

import (
	"log"

	"github.com/mndrix/golog"
)

func main() {
	m := golog.NewMachine().Consult(`
toki(jan_Kesi).
toki(jan_Pola).
toki(jan_Kesi, jan_Pola).
toki(jan_Kesi, toki_pona).
`)
	if m.CanProve(`toki(jan_Kesi).`) {
		log.Printf("toki(jan_Kesi). -> jan Kesi li toki.")
	}

	solutions := m.ProveAll(`toki(jan_Kesi, X).`)
	for _, solution := range solutions {
		log.Printf("jan_Kesi li toki e %s", solution.ByName_("X").String())
	}
}
