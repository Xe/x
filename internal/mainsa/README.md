# ma Insa

This package contains constants and conversion functions for the paracosm that I created, `ma Insa` (lit: land inside).

I really created this paracosm by traveling around to different parts of it (or: where things would be if it did exist clearly, etc) then asking and answering two questions:

- what _is there_?
- and what is it _like_?

For example, let's consider `ma telo seli` (lit: land of hot water) in the paracosm:

What is there? A hottub. It has a radius of 3 meters. There is a pool deck around it that looks like hard rock, but is actually soft foam. The water is heated to 39 degrees celsius (102 f). The water has a yin-yang style circulation pattern to out around the edges of the tub. The water is unchlorinated. There is seating on the inside of the tub as well as a jaccuzi style air bubble system. The water level is always stable no matter how many people enter the tub, but it can fit at least a dozen people. Leaving it doesn't leave clothes or skin feeling overly wet.  
What is it like? Peaceful, the sounds of the air bubbles rising to the surface is a very constant noise, but not something that grates on the ears. The feeling of the heat of the water warms you to the core. The steam of the water meeting the air around it (~80f / ~27c) is very interesting to inhale deeply.

NPC's in the paracosm speak [Toki Pona](http://tokipona.net/tp/default.aspx) and will not understand other languages unless that is relevant to the scenario in question.

## Dates

A date, or `tenpo nimi`, is a combination of the following information:

- the year
- the season
- the month/cycle
- the week
- the day
- the remainder

Days in ma Insa are 8 material plane hours long. A week is 3 days. A month is 3 weeks. A season is 3 months. A year is four seasons. The arbitrary zero date for ma Insa's time is `10/07/2018 @ 12:00am (UTC)` or the unix time `1538870400`.

The tool `cmd/mainsanow` will print the current ma Insa datetime:

```console
$ mainsanow
2018/10/19 05:05:19 suli 0 tawa seli sike poka linja wan suno tu awen 4:05
```
