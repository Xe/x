#!/usr/bin/env nix-shell
#! nix-shell -p python310 -i python

import json
import sqlite3

con = sqlite3.connect("pokemans.db")

cur = con.cursor()
cur.execute("""
CREATE TABLE pokemon
 ( dex_id   TEXT PRIMARY KEY
 , name     TEXT UNIQUE
 , seen     BOOLEAN NOT NULL DEFAULT FALSE
 , caught   BOOLEAN NOT NULL DEFAULT FALSE
 , complete BOOLEAN NOT NULL DEFAULT FALSE
 );""")

with open("./pokemans.json", "r") as fin:
    data = fin.read().rstrip()
    mans = json.loads(data)
    
    for man in mans:
        cur.execute("INSERT INTO pokemon(dex_id, name) VALUES (?1, ?2)", (man["id"], man["name"]))

con.commit()
con.close()
