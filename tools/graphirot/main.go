package main

import (
	"os"

	"github.com/mzohreva/GoGraphviz/graphviz"
)

func main() {
	g := graphviz.Graph{}

	keter := g.AddNode("Keter")
	binah := g.AddNode("Binah")
	chokmah := g.AddNode("Chokmah")
	daat := g.AddNode("Da'at")
	gevurah := g.AddNode("Gevurah")
	hesed := g.AddNode("Hesed")
	tiferet := g.AddNode("Tilferet")
	hod := g.AddNode("Hod")
	netzah := g.AddNode("Netzah")
	yesod := g.AddNode("Yesod")
	malkhut := g.AddNode("Malkhut")

	g.NodeAttribute(keter, "group", "g1")
	g.NodeAttribute(binah, "group", "g2")
	g.NodeAttribute(chokmah, "group", "g2")
	g.NodeAttribute(daat, "group", "g3")
	g.NodeAttribute(gevurah, "group", "g4")
	g.NodeAttribute(hesed, "group", "g4")
	g.NodeAttribute(tiferet, "group", "g5")
	g.NodeAttribute(hod, "group", "g6")
	g.NodeAttribute(netzah, "group", "g6")
	g.NodeAttribute(yesod, "group", "g7")
	g.NodeAttribute(malkhut, "group", "g8")

	invisLink := func(a, b int) {
		e := g.AddEdge(a, b, "")
		g.EdgeAttribute(e, "style", "invis")
	}
	invisLink(daat, keter)
	invisLink(daat, binah)
	invisLink(daat, chokmah)
	invisLink(daat, tiferet)

	g.AddEdge(keter, binah, "")
	g.AddEdge(keter, chokmah, "")
	g.AddEdge(keter, tiferet, "")

	g.AddEdge(binah, chokmah, "")
	g.AddEdge(binah, keter, "")
	g.AddEdge(binah, gevurah, "")
	g.AddEdge(binah, tiferet, "")
	g.AddEdge(binah, hesed, "")

	g.AddEdge(chokmah, keter, "")
	g.AddEdge(chokmah, binah, "")
	g.AddEdge(chokmah, tiferet, "")
	g.AddEdge(chokmah, hesed, "")

	g.AddEdge(gevurah, binah, "")
	g.AddEdge(gevurah, chokmah, "")
	g.AddEdge(gevurah, hesed, "")
	g.AddEdge(gevurah, tiferet, "")
	g.AddEdge(gevurah, hod, "")

	g.AddEdge(hesed, chokmah, "")
	g.AddEdge(hesed, binah, "")
	g.AddEdge(hesed, gevurah, "")
	g.AddEdge(hesed, hod, "")
	g.AddEdge(hesed, netzah, "")

	g.AddEdge(tiferet, keter, "")
	g.AddEdge(tiferet, binah, "")
	g.AddEdge(tiferet, gevurah, "")
	g.AddEdge(tiferet, hesed, "")
	g.AddEdge(tiferet, hod, "")
	g.AddEdge(tiferet, netzah, "")
	g.AddEdge(tiferet, yesod, "")

	g.AddEdge(hod, gevurah, "")
	g.AddEdge(hod, tiferet, "")
	g.AddEdge(hod, netzah, "")
	g.AddEdge(hod, yesod, "")

	g.AddEdge(netzah, hesed, "")
	g.AddEdge(netzah, tiferet, "")
	g.AddEdge(netzah, hod, "")
	g.AddEdge(netzah, yesod, "")

	g.AddEdge(yesod, hod, "")
	g.AddEdge(yesod, tiferet, "")
	g.AddEdge(yesod, netzah, "")
	g.AddEdge(yesod, malkhut, "")

	g.MakeSameRank(binah, chokmah)
	g.MakeSameRank(gevurah, hesed)
	g.MakeSameRank(hod, netzah)

	g.GenerateDOT(os.Stdout)
}
