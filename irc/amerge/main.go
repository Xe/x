package main

import (
	"flag"
	"log"
	"strings"
)

var (
	leftFname  = flag.String("left-db", "./left.db", "database to read from to compare as the left hand")
	rightFname = flag.String("right-db", "./right.db", "\" for the right hand side")
)

func main() {
	flag.Parse()

	leftDB, err := NewDatabase(*leftFname)
	if err != nil {
		panic(err)
	}

	_ = leftDB

	rightDB, err := NewDatabase(*rightFname)
	if err != nil {
		panic(err)
	}

	_ = rightDB

	result := &Database{
		Accounts: make(map[string]*Account),
		Channels: make(map[string]*Channel),
		Bots:     make(map[string]*Bot),
		Groups:   make(map[string]*Group),
		Names:    make(map[string]*Name),
	}

	_ = result

	// Compare accounts and grouped nicks in left database to names in right database
	// this is O(scary)

	for leftAccountName, acc := range leftDB.Accounts {
		for _, leftGroupedNick := range acc.Nicks {
			conflictAcc, err := rightDB.GetAccount(leftGroupedNick.Name)
			if err != nil {
				goto botcheck
			}

			if conflictAcc.Email == acc.Email {
				//log.Printf("Can ignore %s, they are the same user by email account", acc.Name)
				goto botcheck
			}
			log.Printf(
				"While trying to see if %s:%s is present in right database, found a conflict with %s",
				acc.Name, leftGroupedNick.Name, conflictAcc.Name,
			)
			log.Printf(
				"left:  %s %s %s %s",
				acc.Name, acc.Email, acc.Regtime, acc.LastSeenTime,
			)
			log.Printf(
				"right: %s %s %s %s",
				conflictAcc.Name, conflictAcc.Email, conflictAcc.Regtime, conflictAcc.LastSeenTime,
			)

		botcheck:
			//log.Printf("Checking for bot collisions for %s:%s...", acc.Name, leftGroupedNick.Name)
			conflictBot, err := rightDB.GetBot(leftGroupedNick.Name)
			if err != nil {
				goto next
			}

			if strings.ToUpper(conflictBot.Nick) == leftAccountName {
				log.Printf("Nickname %s conflicts with right's bot %s", leftGroupedNick.Name, conflictBot.Nick)
			}
		next:
		}
	}
}
