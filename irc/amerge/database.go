package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	NoSuchAcctErr  = errors.New("There is no such account by that name")
	NoSuchChanErr  = errors.New("There is no such channel by that name")
	NoSuchGroupErr = errors.New("There is no such group by that name")
)

type Database struct {
	Version    string
	ModuleDeps []*Line
	LastUID    string

	LastKID int
	LastXID int
	LastQID int

	Accounts     map[string]*Account
	Channels     map[string]*Channel
	Bots         map[string]*Bot
	Groups       map[string]*Group
	Names        map[string]*Name
	Badwords     []Badword
	Klines       []Kline
	ConnectInfos []ConnectInfo
	HostOffers   []HostOffer
	HostRequests []HostRequest

	ClonesExemptions []ClonesExemption
	Rwatches         []Line

	lines []*Line
	file  *bufio.Scanner
}

func NewDatabase(fname string) (db *Database, err error) {
	fin, err := os.Open(fname)
	if err != nil {
		return
	}

	db = &Database{
		Accounts: make(map[string]*Account),
		Channels: make(map[string]*Channel),
		Bots:     make(map[string]*Bot),
		Groups:   make(map[string]*Group),
		Names:    make(map[string]*Name),
	}

	db.file = bufio.NewScanner(fin)

	for db.file.Scan() {
		rawline := db.file.Text()

		l := &Line{}
		split := strings.Split(rawline, " ")

		l.Verb = split[0]
		l.Args = split[1:]

		db.lines = append(db.lines, l)

		switch l.Verb {
		case "DBV": // Database version
			db.Version = l.Args[0]

		case "MDEP": // Module dependency
			db.ModuleDeps = append(db.ModuleDeps, l)

		case "LUID": // Last used UID for accounts
			db.LastUID = l.Args[0]

		case "MU": // Create a user account
			a := &Account{
				Name:         l.Args[1],
				UID:          l.Args[0],
				Email:        l.Args[3],
				Password:     l.Args[2],
				Regtime:      l.Args[4],
				LastSeenTime: l.Args[5],

				Metadata: make(map[string]string),
			}

			db.Accounts[strings.ToUpper(a.Name)] = a

		case "MDU": // User metadata
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			account.Metadata[l.Args[1]] = strings.Join(l.Args[2:], " ")

		case "AC": // Account access rule (prevents nickserv protections for a mask)
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			ac := Access{
				AccountName: l.Args[0],
				Mask:        l.Args[1],
			}

			account.AccessList = append(account.AccessList, ac)

		case "MI": // MemoServ IGNORE for a user
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			account.Ignores = append(account.Ignores, l.Args[1])

		case "MN": // Account nickname in nick group
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			gn := GroupedNick{
				Account:  l.Args[0],
				Name:     l.Args[1],
				Regtime:  l.Args[2],
				Seentime: l.Args[3],
			}

			account.Nicks = append(account.Nicks, gn)

		case "MCFP": // Certificate Fingerprint
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			account.CertFP = append(account.CertFP, l.Args[1])

		case "ME": // Memo in user's inbox
			account, err := db.GetAccount(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read account %s but got %#v???", l.Args[0], err)
			}

			m := Memo{
				Inbox:    l.Args[0],
				From:     l.Args[1],
				Time:     l.Args[2],
				Read:     l.Args[3] == "1",
				Contents: strings.Join(l.Args[4:], " "),
			}

			account.Memos = append(account.Memos, m)

		case "MC": // Create a channel
			mlockon, err := strconv.ParseInt(l.Args[4], 16, 0)
			if err != nil {
				panic(err)
			}

			c := &Channel{
				Name:     l.Args[0],
				Regtime:  l.Args[1],
				Seentime: l.Args[2],
				Flags:    l.Args[3],
				MlockOn:  int(mlockon),

				Metadata:       make(map[string]string),
				AccessMetadata: make(map[string]AccessMetadata),
			}

			db.Channels[strings.ToUpper(l.Args[0])] = c

		case "CA": // ChanAcs
			c, err := db.GetChannel(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read channel %s but got %#v???", l.Args[0], err)
			}

			ca := ChanAc{
				Channel:     l.Args[0],
				Account:     l.Args[1],
				FlagSet:     l.Args[2],
				DateGranted: l.Args[3],
				WhoGranted:  l.Args[4],
			}

			c.Access = append(c.Access, ca)

		case "MDC": // Channel metadata
			c, err := db.GetChannel(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read channel %s but got %#v???", l.Args[0], err)
			}

			c.Metadata[l.Args[1]] = strings.Join(l.Args[2:], " ")

		case "MDA": // Channel-based entity key->value
			c, err := db.GetChannel(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read channel %s but got %#v???", l.Args[0], err)
			}

			amd := AccessMetadata{
				ChannelName: l.Args[0],
				Entity:      l.Args[1],
				Key:         l.Args[2],
				Value:       l.Args[3],
			}

			c.AccessMetadata[strings.ToUpper(amd.Key)] = amd

		case "NAM":
			nam := &Name{
				Name: l.Args[0],

				Metadata: make(map[string]string),
			}

			db.Names[strings.ToUpper(nam.Name)] = nam

		case "MDN":
			nam, ok := db.Names[strings.ToUpper(l.Args[0])]
			if !ok {
				panic("Atheme is broken with things")
			}

			nam.Metadata[l.Args[1]] = strings.Join(l.Args[2:], " ")

		case "KID": // Biggest kline id used
			kid, err := strconv.ParseInt(l.Args[0], 10, 0)
			if err != nil {
				panic("atheme is broken with KID " + l.Args[0])
			}

			db.LastKID = int(kid)

		case "XID": // Biggest xline id used
			xid, err := strconv.ParseInt(l.Args[0], 10, 0)
			if err != nil {
				panic("atheme is broken with XID " + l.Args[0])
			}

			db.LastXID = int(xid)

		case "QID": // Biggest qline id used
			qid, err := strconv.ParseInt(l.Args[0], 10, 0)
			if err != nil {
				panic("atheme is broken with QID " + l.Args[0])
			}

			db.LastQID = int(qid)

		case "KL": // kline
			id, err := strconv.ParseInt(l.Args[0], 10, 0)
			if err != nil {
				panic(err)
			}

			kl := Kline{
				ID:       int(id),
				User:     l.Args[1],
				Host:     l.Args[2],
				Duration: l.Args[3],
				DateSet:  l.Args[4],
				WhoSet:   l.Args[5],
				Reason:   strings.Join(l.Args[6:], " "),
			}

			db.Klines = append(db.Klines, kl)

		case "BOT": // BotServ bot
			bot := &Bot{
				Nick:         l.Args[0],
				User:         l.Args[1],
				Host:         l.Args[2],
				IsPrivate:    l.Args[3] == "1",
				CreationDate: l.Args[4],
				Gecos:        l.Args[5],
			}

			db.Bots[strings.ToUpper(bot.Nick)] = bot

		case "BW": // BADWORDS entry
			bw := Badword{
				Mask:    l.Args[0],
				TimeSet: l.Args[1],
				Setter:  l.Args[2],
			}

			if len(l.Args) == 5 {
				bw.Channel = l.Args[3]
				bw.Action = l.Args[4]
			} else {
				bw.Setter = bw.Setter + " " + l.Args[3]
				bw.Channel = l.Args[4]
				bw.Action = l.Args[5]
			}

			db.Badwords = append(db.Badwords, bw) // TODO: move this to Channel?

		case "GRP": // Group
			g := &Group{
				UID:          l.Args[0],
				Name:         l.Args[1],
				CreationDate: l.Args[2],
				Flags:        l.Args[3],

				Metadata: make(map[string]string),
			}

			db.Groups[strings.ToUpper(l.Args[1])] = g

		case "GACL": // Group access list
			g, err := db.GetGroup(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read group %s but got %#v???", l.Args[0], err)
			}

			gacl := GroupACL{
				GroupName:   l.Args[0],
				AccountName: l.Args[1],
				Flags:       l.Args[2],
			}

			g.ACL = append(g.ACL, gacl)

		case "MDG": // Group Metadata
			g, err := db.GetGroup(l.Args[0])
			if err != nil {
				log.Panicf("Tried to read group %s but got %#v???", l.Args[0], err)
			}

			g.Metadata[l.Args[1]] = strings.Join(l.Args[2:], " ")

		case "CLONES-EX": // CLONES exemptions
			ce := ClonesExemption{
				IP:     l.Args[0],
				Min:    l.Args[1],
				Max:    l.Args[2],
				Expiry: l.Args[3],
				Reason: strings.Join(l.Args[4:], " "),
			}

			db.ClonesExemptions = append(db.ClonesExemptions, ce)

		case "LI": // InfoServ INFO posts
			ci := ConnectInfo{
				Creator:      l.Args[0],
				Topic:        l.Args[1],
				CreationDate: l.Args[2],
				Body:         strings.Join(l.Args[3:], " "),
			}

			db.ConnectInfos = append(db.ConnectInfos, ci)

		case "HO": // Vhost offer
			var ho HostOffer

			if len(l.Args) == 3 {
				ho = HostOffer{
					Vhost:        l.Args[0],
					CreationDate: l.Args[1],
					Creator:      l.Args[2],
				}
			} else {
				ho = HostOffer{
					Group:        l.Args[0],
					Vhost:        l.Args[1],
					CreationDate: l.Args[2],
					Creator:      l.Args[3],
				}
			}

			db.HostOffers = append(db.HostOffers, ho)

		case "HR": // Vhost request
			hr := HostRequest{
				Account:     l.Args[0],
				Vhost:       l.Args[1],
				RequestTime: l.Args[2],
			}

			db.HostRequests = append(db.HostRequests, hr)

		// Verbs to ignore
		case "":

		default:
			fmt.Printf("%#v\n", l)
		}
	}

	return
}

func (db *Database) GetAccount(name string) (*Account, error) {
	account, ok := db.Accounts[strings.ToUpper(name)]
	if !ok {
		return nil, NoSuchAcctErr
	}

	return account, nil
}

func (db *Database) GetChannel(name string) (*Channel, error) {
	channel, ok := db.Channels[strings.ToUpper(name)]
	if !ok {
		return nil, NoSuchChanErr
	}

	return channel, nil
}

func (db *Database) GetGroup(name string) (*Group, error) {
	group, ok := db.Groups[strings.ToUpper(name)]
	if !ok {
		return nil, NoSuchGroupErr
	}

	return group, nil
}

func (db *Database) GetBot(name string) (*Bot, error) {
	group, ok := db.Bots[strings.ToUpper(name)]
	if !ok {
		return nil, NoSuchGroupErr
	}

	return group, nil
}

type Line struct {
	Verb string
	Args []string
}

type Account struct {
	Name     string
	Email    string
	Flags    string
	Kind     string
	UID      string
	Password string

	Regtime      string
	LastSeenTime string

	Metadata   map[string]string
	Nicks      []GroupedNick
	Memos      []Memo
	CertFP     []string
	AccessList []Access
	Ignores    []string
}

type Access struct {
	AccountName string
	Mask        string
}

type Name struct {
	Name     string
	Metadata map[string]string
}

type GroupedNick struct {
	Account  string
	Name     string
	Regtime  string
	Seentime string
}

type Memo struct {
	Inbox    string
	From     string // an account name
	Time     string
	Read     bool
	Contents string
}

type Channel struct {
	Name       string
	Regtime    string
	Seentime   string
	Flags      string
	MlockOn    int
	MlockOff   int
	MlockLimit int
	MlockKey   string

	Access         []ChanAc
	Metadata       map[string]string
	AccessMetadata map[string]AccessMetadata
}

type AccessMetadata struct {
	ChannelName string
	Entity      string
	Key         string
	Value       string
}

type ChanAc struct {
	Channel     string
	Account     string
	FlagSet     string
	DateGranted string
	WhoGranted  string
}

type Kline struct {
	ID       int
	User     string
	Host     string
	Duration string
	DateSet  string
	WhoSet   string
	Reason   string
}

type Bot struct {
	Nick         string
	User         string
	Host         string
	IsPrivate    bool
	CreationDate string
	Gecos        string
}

type Badword struct {
	Mask    string
	TimeSet string
	Setter  string // can be Foo or Foo (Bar)
	Channel string
	Action  string
}

type Group struct {
	UID          string
	Name         string
	CreationDate string
	Flags        string

	ACL      []GroupACL
	Metadata map[string]string
}

type GroupACL struct {
	GroupName   string
	AccountName string
	Flags       string
}

type ConnectInfo struct {
	Creator      string
	Topic        string
	CreationDate string
	Body         string
}

type HostOffer struct { // if args number is 3 no group
	Group        string
	Vhost        string
	CreationDate string
	Creator      string
}

type HostRequest struct {
	Account     string
	Vhost       string
	RequestTime string
}

type ClonesExemption struct {
	IP     string
	Min    string
	Max    string
	Expiry string
	Reason string
}
