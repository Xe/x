#!/usr/bin/env nix-shell
#! nix-shell -p python310 -i python

import json
import sqlite3

con = sqlite3.connect("pokemans.db")

help_text = """Pokemon Legends Arceus Pokedex Tracker from Within

Commands:
  catch <id|name>
  complete <id|name>
  see <id|name>
  whatsleft [seen|catch|complete]
"""

print(help_text)

while True:
    command = input("> ")
    command = command.split()

    match command[0]:
        case "help":
            print(help_text)
        case "catch":
            names = map(lambda x: (x,),command[1:])
            cur = con.cursor()
            cur.executemany("UPDATE pokemon SET seen=true, caught=true WHERE name = ?1 COLLATE NOCASE", names)
            con.commit()
        case "complete":
            names = map(lambda x: (x,),command[1:])
            cur = con.cursor()
            cur.executemany("UPDATE pokemon SET seen=true, caught=true, complete=true WHERE name = ?1 COLLATE NOCASE", names)
            con.commit()
        case "see":
            names = map(lambda x: (x,),command[1:])
            cur = con.cursor()
            cur.executemany("UPDATE pokemon SET seen=true WHERE name = ?1 COLLATE NOCASE", names)
            con.commit()
        case "whatsleft":
            if len(command) == 1:
                cur = con.cursor()
                for pokemon in cur.execute("SELECT dex_id, name FROM pokemon WHERE caught = 0"):
                    print("%s: %s" % (pokemon[0], pokemon[1]))
                continue
            kind = command[1]
