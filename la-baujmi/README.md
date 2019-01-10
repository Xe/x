# la baujmi

Combination of:

> bangu
> 
> x1 is a/the language/dialect used by x2 to express/communicate x3 (si'o/du'u, not quote).

> jimpe
> 
> x1 understands/comprehends fact/truth x2 (du'u) about subject x3; x1 understands (fi) x3.

This is an attempt to create a tool that can understand language. 

At first, [Toki Pona](http://tokipona.net) will be used. At a high level a toki pona sentence consists of four main parts:

- context phrase
- subject + descriptors
- verb + descriptors
- object + descriptors

You can describe a sentence as a form of predicate relation between those four parts. If you are told "Stacy purchased a tool for strange-plant", you can later then ask the program who purchased a tool for strange-plant.

Because a Toki Pona sentence always matches the following form:

```
[<name> o,] [context la] <subject> [li <verb> [e <object>]]
```

And the particle `seme` fills in the part of a question that you don't know. So from this we can fill in the blanks with prolog.

Consider the following:

```
jan Kesi li toki.
Cadey is speaking
toki(jan_Kesi).

jan Kesi en jan Pola li toki.
Cadey and Pola are speaking.
toki(jan_Kesi).
toki(jan_Pola).

jan Kesi li toki e jan Pola.
Cadey is talking about Pola
toki(jan_Kesi, jan_Pola).

jan Kesi li toki e toki pona.
Cadey is talking about toki pona.
toki(jan_Kesi, toki_pona).
```

And then we can ask prolog questions about this sentence:

```
seme li toki?
```

```
> toki(X).
toki(jan_Kesi).
jan Kesi li toki.
```
