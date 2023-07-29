package parsetorrentname

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

var updateGoldenFiles = flag.Bool("update", false, "update golden files in testdata/")

var testData = []string{
	"The Walking Dead S05E03 720p HDTV x264-ASAP[ettv]",
	"Hercules (2014) 1080p BrRip H264 - YIFY",
	"Dawn.of.the.Planet.of.the.Apes.2014.HDRip.XViD-EVO",
	"The Big Bang Theory S08E06 HDTV XviD-LOL [eztv]",
	"22 Jump Street (2014) 720p BrRip x264 - YIFY",
	"Hercules.2014.EXTENDED.1080p.WEB-DL.DD5.1.H264-RARBG",
	"Hercules.2014.Extended.Cut.HDRip.XViD-juggs[ETRG]",
	"Hercules (2014) WEBDL DVDRip XviD-MAX",
	"WWE Hell in a Cell 2014 PPV WEB-DL x264-WD -={SPARROW}=-",
	"UFC.179.PPV.HDTV.x264-Ebi[rartv]",
	"Marvels Agents of S H I E L D S02E05 HDTV x264-KILLERS [eztv]",
	"X-Men.Days.of.Future.Past.2014.1080p.WEB-DL.DD5.1.H264-RARBG",
	"Guardians Of The Galaxy 2014 R6 720p HDCAM x264-JYK",
	"Marvel's.Agents.of.S.H.I.E.L.D.S02E01.Shadows.1080p.WEB-DL.DD5.1",
	"Marvels Agents of S.H.I.E.L.D. S02E06 HDTV x264-KILLERS[ettv]",
	"Guardians of the Galaxy (CamRip / 2014)",
	"The.Walking.Dead.S05E03.1080p.WEB-DL.DD5.1.H.264-Cyphanix[rartv]",
	"Brave.2012.R5.DVDRip.XViD.LiNE-UNiQUE",
	"Lets.Be.Cops.2014.BRRip.XViD-juggs[ETRG]",
	"These.Final.Hours.2013.WBBRip XViD",
	"Downton Abbey 5x06 HDTV x264-FoV [eztv]",
	"Annabelle.2014.HC.HDRip.XViD.AC3-juggs[ETRG]",
	"Lucy.2014.HC.HDRip.XViD-juggs[ETRG]",
	"The Flash 2014 S01E04 HDTV x264-FUM[ettv]",
	"South Park S18E05 HDTV x264-KILLERS [eztv]",
	"The Flash 2014 S01E03 HDTV x264-LOL[ettv]",
	"The Flash 2014 S01E01 HDTV x264-LOL[ettv]",
	"Lucy 2014 Dual-Audio WEBRip 1400Mb",
	"Teenage Mutant Ninja Turtles (HdRip / 2014)",
	"Teenage Mutant Ninja Turtles (unknown_release_type / 2014)",
	"The Simpsons S26E05 HDTV x264 PROPER-LOL [eztv]",
	"2047 - Sights of Death (2014) 720p BrRip x264 - YIFY",
	"Two and a Half Men S12E01 HDTV x264 REPACK-LOL [eztv]",
	"Dinosaur 13 2014 WEBrip XviD AC3 MiLLENiUM",
	"Teenage.Mutant.Ninja.Turtles.2014.HDRip.XviD.MP3-RARBG",
	"Dawn.Of.The.Planet.of.The.Apes.2014.1080p.WEB-DL.DD51.H264-RARBG",
	"Teenage.Mutant.Ninja.Turtles.2014.720p.HDRip.x264.AC3.5.1-RARBG",
	"Gotham.S01E05.Viper.WEB-DL.x264.AAC",
	"Into.The.Storm.2014.1080p.WEB-DL.AAC2.0.H264-RARBG",
	"Lucy 2014 Dual-Audio 720p WEBRip",
	"Into The Storm 2014 1080p BRRip x264 DTS-JYK",
	"Sin.City.A.Dame.to.Kill.For.2014.1080p.BluRay.x264-SPARKS",
	"WWE Monday Night Raw 3rd Nov 2014 HDTV x264-Sir Paul",
	"Jack.And.The.Cuckoo-Clock.Heart.2013.BRRip XViD",
	"WWE Hell in a Cell 2014 HDTV x264 SNHD",
	"Dracula.Untold.2014.TS.XViD.AC3.MrSeeN-SiMPLE",
	"The Missing 1x01 Pilot HDTV x264-FoV [eztv]",
	"Doctor.Who.2005.8x11.Dark.Water.720p.HDTV.x264-FoV[rartv]",
	"Gotham.S01E07.Penguins.Umbrella.WEB-DL.x264.AAC",
	"One Shot [2014] DVDRip XViD-ViCKY",
	"The Shaukeens 2014 Hindi (1CD) DvDScr x264 AAC...Hon3y",
	"The Shaukeens (2014) 1CD DvDScr Rip x264 [DDR]",
	"Annabelle.2014.1080p.PROPER.HC.WEBRip.x264.AAC.2.0-RARBG",
	"Interstellar (2014) CAM ENG x264 AAC-CPG",
	"Guardians of the Galaxy (2014) Dual Audio DVDRip AVI",
	"Eliza Graves (2014) Dual Audio WEB-DL 720p MKV x264",
	"WWE Monday Night Raw 2014 11 10 WS PDTV x264-RKOFAN1990 -={SPARR",
	"Sons.of.Anarchy.S01E03",
	"doctor_who_2005.8x12.death_in_heaven.720p_hdtv_x264-fov",
	"breaking.bad.s01e01.720p.bluray.x264-reward",
	"Game of Thrones - 4x03 - Breaker of Chains",
	"[720pMkv.Com]_sons.of.anarchy.s05e10.480p.BluRay.x264-GAnGSteR",
	"[ www.Speed.cd ] -Sons.of.Anarchy.S07E07.720p.HDTV.X264-DIMENSION",
	"Community.s02e20.rus.eng.720p.Kybik.v.Kybe",
	"The.Jungle.Book.2016.3D.1080p.BRRip.SBS.x264.AAC-ETRG",
	"Ant-Man.2015.3D.1080p.BRRip.Half-SBS.x264.AAC-m2g",
	"Ice.Age.Collision.Course.2016.READNFO.720p.HDRIP.X264.AC3.TiTAN",
	"Red.Sonja.Queen.Of.Plagues.2016.BDRip.x264-W4F[PRiME]",
	"The Purge: Election Year (2016) HC - 720p HDRiP - 900MB - ShAaNi",
	"War Dogs (2016) HDTS 600MB - NBY",
	"The Hateful Eight (2015) 720p BluRay - x265 HEVC - 999MB - ShAaN",
	"The.Boss.2016.UNRATED.720p.BRRip.x264.AAC-ETRG",
	"Return.To.Snowy.River.1988.iNTERNAL.DVDRip.x264-W4F[PRiME]",
	"Akira (2016) - UpScaled - 720p - DesiSCR-Rip - Hindi - x264 - AC3 - 5.1 - Mafiaking - M2Tv",
	"Ben Hur 2016 TELESYNC x264 AC3 MAXPRO",
	"The.Secret.Life.of.Pets.2016.HDRiP.AAC-LC.x264-LEGi0N",
	"[HorribleSubs] Clockwork Planet - 10 [480p].mkv",
	"[HorribleSubs] Detective Conan - 862 [1080p].mkv",
	"thomas.and.friends.s19e09_s20e14.convert.hdtv.x264-w4f[eztv].mkv",
	"Blade.Runner.2049.2017.1080p.WEB-DL.DD5.1.H264-FGT-[rarbg.to]",
	"2012(2009).1080p.Dual Audio(Hindi+English) 5.1 Audios",
	"2012 (2009) 1080p BrRip x264 - 1.7GB - YIFY",
	"2012 2009 x264 720p Esub BluRay 6.0 Dual Audio English Hindi GOPISAHI",
}

func TestParser(t *testing.T) {
	for i, fname := range testData {
		t.Run(fmt.Sprintf("golden_file_%03d", i), func(t *testing.T) {
			tor, err := Parse(fname)
			if err != nil {
				t.Fatalf("test %v: parser error:\n  %v", i, err)
			}

			goldenFilename := filepath.Join("testdata", fmt.Sprintf("golden_file_%03d.json", i))

			if *updateGoldenFiles {
				buf, err := json.MarshalIndent(tor, "", "  ")
				if err != nil {
					t.Fatalf("error marshaling result: %v", err)
				}

				if err = ioutil.WriteFile(goldenFilename, buf, 0644); err != nil {
					t.Fatalf("unable to update golden file: %v", err)
				}
			}

			buf, err := ioutil.ReadFile(goldenFilename)
			if err != nil {
				t.Fatalf("error loading golden file: %v", err)
			}

			var want TorrentInfo
			err = json.Unmarshal(buf, &want)
			if err != nil {
				t.Fatalf("error unmarshalling golden file %v: %v", goldenFilename, err)
			}

			if !reflect.DeepEqual(*tor, want) {
				t.Fatalf("test %v: wrong result for %q\nwant:\n  %v\ngot:\n  %v", i, fname, want, *tor)
			}
		})
	}
}
