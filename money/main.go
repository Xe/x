package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	csvImport = flag.String("import", "", "csv file to import")
	dataLoc   = flag.String("dataloc", "/home/xena/.local/share/within/money/data.db", "location of datastore on disk")
)

const (
	BoADate = "01/02/2006"
	Schema  = `CREATE TABLE IF NOT EXISTS Transactions(
	id          INTEGER PRIMARY KEY,
	date        TEXT,
	description TEXT,
	amount      REAL,
	balance     REAL,
	hash        TEXT UNIQUE);`
)

type Transaction struct {
	Date           time.Time `json:"date"`
	Description    string    `json:"description"`
	Amount         float64   `json:"amount"`
	RunningBalance float64   `json:"running"`
	Hash           string    `json:"-"`
}

type WorldState struct {
	StartingMoney float64       `json:"starting"`
	EndingMoney   float64       `json:"ending"`
	Income        float64       `json:"income"`
	Expenses      float64       `json:"expenses"`
	Transactions  []Transaction `json:"transactions"`
}

func (w *WorldState) Summary() {
	fmt.Println("Summary of imported data:")
	wr := new(tabwriter.Writer)

	wr.Init(os.Stdout, 5, 8, 2, '\t', 0)

	fmt.Fprintln(wr, "Income\tExpenses\tStarting\tEnding")
	fmt.Fprintf(wr, "$%f\t$%f\t$%f\t$%f", w.Income, w.Expenses, w.StartingMoney, w.EndingMoney)
	wr.Flush()
	fmt.Println()
}

func floatOrDie(s string) float64 {
	res, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}

	return res
}

func getthirdValue(r *csv.Reader) float64 {
	line, err := r.Read()
	if err != nil {
		panic(err)
	}

	if line[2] == "" {
		log.Fatal("Improper CSV")
	}

	return floatOrDie(line[2])
}

func MakeTransaction(date, description, amount, running string) (t Transaction) {
	d, err := time.Parse(BoADate, date)
	if err != nil {
		panic(err)
	}

	t = Transaction{
		Date:           d,
		Description:    description,
		Amount:         floatOrDie(amount),
		RunningBalance: floatOrDie(running),
		Hash:           fmt.Sprintf("%x", md5.Sum([]byte(date+description+amount+running))),
	}

	return
}

func MakeWorld(fin io.Reader) *WorldState {
	w := &WorldState{}

	r := csv.NewReader(fin)

	dateline, err := r.Read()
	if err != nil {
		panic(err)
	}
	if dateline[0] != "Description" {
		log.Fatal("Improper CSV")
	}

	w.StartingMoney = getthirdValue(r)
	w.Income = getthirdValue(r)
	w.Expenses = getthirdValue(r)
	w.EndingMoney = getthirdValue(r)

	r.Read()
	r.Read()

	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, record := range records {
		w.Transactions = append(w.Transactions, MakeTransaction(
			record[0],
			record[1],
			record[2],
			record[3],
		))
	}

	return w
}

func main() {
	flag.Parse()

	doImport := false

	if *csvImport != "" {
		doImport = true
	}

	if *dataLoc == "" {
		log.Fatal("main: need database location")
	}

	if doImport {
		importData()
	}
}

func importData() {
	fin, err := os.Open(*csvImport)
	if err != nil {
		log.Fatal(err)
	}

	w := MakeWorld(fin)
	w.Summary()

	db, err := sql.Open("sqlite3", *dataLoc)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(Schema)
	if err != nil {
		log.Fatal(err)
	}

	insertStmt, err := db.Prepare("INSERT INTO Transactions VALUES (NULL, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}

	for _, tx := range w.Transactions {
		_, err := insertStmt.Exec(tx.Date, tx.Description, tx.Amount, tx.RunningBalance, tx.Hash)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
