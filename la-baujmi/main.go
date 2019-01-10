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
command(ilo_Kesi, toki(ziho, jan_Kesi)).
`)
	if m.CanProve(`toki(jan_Kesi).`) {
		log.Printf("toki(jan_Kesi). -> jan Kesi li toki.")
	}

	solutions := m.ProveAll(`toki(jan_Kesi, X).`)
	for _, solution := range solutions {
		log.Printf("jan_Kesi li toki e %s", solution.ByName_("X").String())
	}

	solutions = m.ProveAll(`command(X, toki(ziho, jan_Kesi)).`)
	for _, solution := range solutions {
		log.Printf("%s o, toki e jan_Kesi", solution.ByName_("X").String())
	}
}
