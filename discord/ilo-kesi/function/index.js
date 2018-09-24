"use strict";
/**
 * Parsing toki pona texts into sentences and sentence parts, and then into structured sentences that reflect the
 * structure of sitelen sitelen blocks.
 *
 * @type {{parse}}
 */
var sitelenParser = function () {
    'use strict';

    /**
     * Core parser into sitelen sitelen structure.
     * @param parseable a sentence to parse
     * @returns {*[]} a structured sentence array
     */
    function getSimpleStructuredSentence(parseable) {
        var tokens = parseable.split(' '),
            prepositions = ['tawa', 'tan', 'lon', 'kepeken', 'sama', 'poka'],
            objectMarker = ['li', 'e'],
            part = {part: 'subject', tokens: []},
            sentence = [part];

        tokens.forEach(function (token, index) {
            if (objectMarker.indexOf(token) > -1 &&
                index < tokens.length - 1) {
                sentence.push({part: 'objectMarker', sep: token, tokens: []});
                part = sentence[sentence.length - 1];
                return;
            } else if (prepositions.indexOf(token) > -1 && objectMarker.indexOf(tokens[index - 1]) === -1 &&
                index < tokens.length - 1 && objectMarker.indexOf(tokens[index + 1]) === -1) {
                sentence.push({part: 'prepPhrase', sep: token, tokens: []});
                part = sentence[sentence.length - 1];
                return;
            } else if (token === 'o' && part.tokens.length > 0) {
                // the o token should be in a container when it is used to address something, not in commands
                part.part = 'address';
                part.sep = 'o';
                sentence.push({part: 'subject', tokens: []});
                part = sentence[sentence.length - 1];
                return;
            } else if (token === 'a' && part.tokens.length > 0 && part.sep) {
                // the a token should never be in a container
                sentence.push({part: 'interjection', sep: null, tokens: [token]});
                part = sentence[sentence.length - 1];
                return;
            }

            part.tokens.push(token);

            if (allowedWords.indexOf(token) === -1){
                throw {type: 'illegal token', message: 'illegal token: ' + token};
            }
        });

        // filter out empty parts
        sentence = sentence.filter(function (part) {
            return part.tokens.length > 0;
        });
        return sentence;

    }

    /**
     * Preformats a given text, so that it splits it on punctuation marks.
     * @param text  text to preformat
     * @returns {{parsable: Array, raw: Array}} parsable array of raw text and punctuation
     */
    function preformat(text) {
        var result = text.match(/[^\.!\?#]+[\.!\?#]+/g),
            punctuation = ['.', ':', '?', '!', ','];

        var parsableParts = [], rawParts = [];
        if (!result) { // allow sentence fractions without any punctuation
            result = [text + (punctuation.indexOf(text) === -1 ? '|' : '')];
            // console.log('WARNING: sentence fraction without punctuation');
        }
        result.forEach(function (sentence) {
            sentence = sentence.trim();

            var parsableSentence = [];
            parsableParts.push(parsableSentence);
            rawParts.push(sentence);

            var body = sentence.substr(0, sentence.length - 1);

            // remove the comma before the la-clause and before a repeating li clause
            body = body.replace(', la ', ' la ');
            body = body.replace(', li ', ' li ');

            // split on context separators comma and colon
            var laparts = body.split(/ la /);
            laparts.forEach(function (lapart, index) {
                var colonparts = lapart.split(/:/);
                colonparts.forEach(function (colonpart, index) {
                    var commaparts = colonpart.split(/,/);
                    commaparts.forEach(function (commapart, index) {
                        commapart = commapart.trim();

                        parsableSentence.push({content: commapart});
                        if (index < commaparts.length - 1) {
                            parsableSentence.push({punctuation: ['comma']});
                        }
                    });

                    if (index < colonparts.length - 1) {
                        parsableSentence.push({punctuation: ['colon']});
                    }
                });
                if (laparts.length === 2 && index === 0) {
                    parsableSentence.push({punctuation: ['la']});
                }
            });

            var terminator = sentence.substr(-1);
            switch (terminator) {
                case '.':
                    parsableSentence.push({punctuation: ['period']});
                    break;
                case ':':
                    parsableSentence.push({punctuation: ['colon']});
                    break;
                case '!':
                    parsableSentence.push({punctuation: ['exclamation']});
                    break;
                case '?':
                    parsableSentence.push({punctuation: ['question']});
                    break;
                case '#':
                    parsableSentence.push({punctuation: ['banner']});
                    break;
                default:
                    break;
            }

        });
        return {parsable: parsableParts, raw: rawParts};
    }

    /**
     * Split proper names into Toki Pona syllables. It is assumed that the proper name follows standard Toki Pona rules.
     * @param properName the proper name string to split into syllables
     */
    function splitProperIntoSyllables(properName) {
        if (properName.length === 0) {
            return [];
        }

        var vowels = ['o', 'u', 'i', 'a', 'e'],
            allowed = ['o', 'u', 'i', 'a', 'e', 'mo', 'mu', 'mi', 'ma', 'me', 'no', 'nu', 'ni', 'na', 'ne', 'po', 'pu', 'pi', 'pa', 'pe', 'to', 'tu', 'ta', 'te', 'ko', 'ku', 'ki', 'ka', 'ke', 'so', 'su', 'si', 'sa', 'se', 'wi', 'wa', 'we', 'lo', 'lu', 'li', 'la', 'le', 'jo', 'ju', 'ja', 'je', 'on', 'un', 'in', 'an', 'en', 'mon', 'mun', 'min', 'man', 'men', 'non', 'nun', 'nin', 'nan', 'nen', 'pon', 'pun', 'pin', 'pan', 'pen', 'ton', 'tun', 'tan', 'ten', 'kon', 'kun', 'kin', 'kan', 'ken', 'son', 'sun', 'sin', 'san', 'sen', 'win', 'wan', 'wen', 'lon', 'lun', 'lin', 'lan', 'len', 'jon', 'jun', 'jan', 'jen'],
            syllables = [],
            first = properName.substr(0, 1),
            third = properName.substr(2, 1),
            fourth = properName.substr(3, 1);

        // ponoman, monsi, akesi

        if (vowels.indexOf(first) === -1) {
            if (third === 'n' && vowels.indexOf(fourth) === -1) {
                syllables.push(properName.substr(0, 3));
                syllables = syllables.concat(splitProperIntoSyllables(properName.substr(3)));
            } else {
                syllables.push(properName.substr(0, 2));
                syllables = syllables.concat(splitProperIntoSyllables(properName.substr(2)));
            }
        } else {
            if (properName.length === 2) {
                return [properName];
            } else {
                syllables.push(first);
                syllables = syllables.concat(splitProperIntoSyllables(properName.substr(1)));
            }
        }

        syllables.forEach(function(syllable){
            if (allowed.indexOf(syllable) === -1){
                throw {type: 'illegal syllable' ,message: 'following syllabe not allowed: ' + syllable};
            }
        });

        return syllables;
    }

    /**
     * Postprocessing for the simple parses that splits the structured sentence into more structure, such as prepositional
     * phrases, proper names and the pi-construct.
     *
     * @param sentence  the structured sentence
     * @returns {*} a processed structured sentence
     */
    function postprocessing(sentence) {
        var prepositionContainers = ['lon', 'tan', 'kepeken', 'tawa', 'sama', 'poka', 'pi'],
            prepositionSplitIndex;

        // split prepositional phrases inside containers (such as the verb li-container)
        sentence.forEach(function (part, index) {
            prepositionSplitIndex = -1;
            part.tokens.forEach(function (token, tokenIndex) {
                if (prepositionContainers.indexOf(token) > -1 && tokenIndex < part.tokens.length - 1) {
                    prepositionSplitIndex = tokenIndex;
                }
            });

            if (prepositionSplitIndex > -1) {
                var newParts = [];
                if (prepositionSplitIndex > 0) {
                    newParts.push({part: part.part, tokens: part.tokens.slice(0, prepositionSplitIndex)});
                }
                newParts.push({
                    part: part.part,
                    sep: part.tokens[prepositionSplitIndex],
                    tokens: part.tokens.slice(prepositionSplitIndex + 1)
                });
                sentence[index] = {part: part.part, sep: part.sep, parts: newParts};
            }
        });

        // split proper names inside containers
        sentence.forEach(function (part, index) {
            var parts = [part];
            if (!part.tokens) {
                if (part.parts) {
                    parts = part.parts;
                } else {
                    return;
                }
            }
            parts.forEach(function (part) {
                var nameSplitIndex = [];
                part.tokens.forEach(function (token, tokenIndex) {
                    if (token.substr(0, 1).toUpperCase() === token.substr(0, 1)) {
                        nameSplitIndex.push(tokenIndex);
                    }
                });
                var last = -1;
                var newParts = [];
                nameSplitIndex.forEach(function (idx) {
                    if (idx > last + 1) {
                        newParts.push({part: part.part, tokens: part.tokens.slice(last + 1, idx)});
                    }
                    newParts.push({
                        part: part.part,
                        sep: 'cartouche',
                        tokens: splitProperIntoSyllables(part.tokens[idx].toLowerCase())
                    });
                    last = idx;
                });
                if (nameSplitIndex.length > 0 && nameSplitIndex[nameSplitIndex.length - 1] < part.tokens.length - 1) {
                    newParts.push({
                        part: part.part,
                        tokens: part.tokens.slice(nameSplitIndex[nameSplitIndex.length - 1] + 1)
                    });
                }
                if (nameSplitIndex.length > 0) {
                    sentence[index] = {part: part.part, sep: part.sep, parts: newParts};
                }
            });
        });
        return sentence;
    }

    /**
     * Main parser that processes a sentence.
     *
     * @param sentence  the input sentence
     * @returns {Array} the structured sentence
     */
    function parseSentence(sentence) {
        var structuredSentence = [];

        sentence.forEach(function (part) {
            if (part.content) {
                // find proper names
                var properNames = [];
                part.content = part.content.replace(/([A-Z][\w-]*)/g, function (item) {
                    properNames.push(item);
                    return '\'Name\'';
                });

                var value = getSimpleStructuredSentence(part.content);

                value.forEach(function (part) {
                    part.tokens.forEach(function (token, index) {
                        if (token === '\'Name\'') {
                            part.tokens[index] = properNames.shift();
                        }
                    });
                });
                structuredSentence.push.apply(structuredSentence, value);
            } else if (part.punctuation) {
                structuredSentence.push({part: 'punctuation', tokens: part.punctuation});
            }
        });

        structuredSentence = postprocessing(structuredSentence);
        return structuredSentence;
    }

    /**
     * Parser wrapper that splits a text into sentences that are parsed.
     * @param text  a full text
     * @returns {Array} an array of structured sentences
     */
    function parse(text) {
        return preformat(text.replace(/\s\s+/g, ' ')).parsable.map(function (sentence) {
            return parseSentence(sentence);
        });
    }

    return {
        parse: parse
    };
}();

var tokiPonaDictionary = [
    {name: 'a', gloss: 'ah', grammar: ['interj']},
    {name: 'akesi', category: 'animal', gloss: 'reptile', grammar: ['n']},
    {name: 'ala', gloss: 'no', grammar: ['mod', 'n', 'interj']},
    {name: 'ali', gloss: 'all', grammar: ['n', 'mod']},
    {name: 'anpa', gloss: 'under', grammar: ['n', 'mod']},
    {name: 'ante', gloss: 'different', grammar: ['n', 'mod', 'conj', 'vt']},
    {name: 'anu', category: 'separator', gloss: 'or', grammar: ['conj']},
    {name: 'awen', gloss: 'remain', grammar: ['vi', 'vt', 'mod']},
    {name: 'e', category: 'separator', gloss: 'object marker', grammar: ['sep']},
    {name: 'en', category: 'separator', gloss: 'and', grammar: ['conj']},
    {name: 'esun', gloss: 'shop', grammar: ['n']},
    {name: 'ijo', gloss: 'thing', grammar: ['n', 'mod', 'vt']},
    {name: 'ike', gloss: 'evil', grammar: ['mod', 'interj', 'n', 'vt', 'vi']},
    {name: 'ilo', gloss: 'tool', grammar: ['n']},
    {name: 'insa', gloss: 'inside', grammar: ['n', 'mod']},
    {name: 'jaki', gloss: 'dirty', grammar: ['mod', 'n', 'vt', 'interj']},
    {name: 'jan', category: 'animal', gloss: 'person', grammar: ['n', 'mod', 'vt']},
    {name: 'jelo', category: 'color', gloss: 'yellow', grammar: ['mod']},
    {name: 'jo', gloss: 'have', grammar: ['vt', 'n']},
    {name: 'kala', category: 'animal', gloss: 'fish', grammar: ['n']},
    {name: 'kalama', gloss: 'sound', grammar: ['n', 'vi', 'vt']},
    {name: 'kama', gloss: 'come', grammar: ['vi', 'n', 'mod', 'vt']},
    {name: 'kasi', gloss: 'plant', grammar: ['n']},
    {name: 'ken', gloss: 'possible', grammar: ['vi', 'n', 'vt']},
    {name: 'kepeken', gloss: 'use', grammar: ['vt', 'prep']},
    {name: 'kili', gloss: 'fruit', grammar: ['n']},
    {name: 'kin', gloss: 'also', grammar: ['mod']},
    {name: 'kiwen', gloss: 'rock', grammar: ['mod', 'n']},
    {name: 'ko', gloss: 'squishy', grammar: ['n']},
    {name: 'kon', gloss: 'soul', grammar: ['n', 'mod']},
    {name: 'kule', gloss: 'color', grammar: ['n', 'mod', 'vt']},
    {name: 'kulupu', gloss: 'group', grammar: ['n', 'mod']},
    {name: 'kute', gloss: 'listen', grammar: ['vt', 'mod']},
    {name: 'la', category: 'separator', gloss: 'in context', grammar: ['sep']},
    {name: 'lape', gloss: 'rest', grammar: ['n', 'vi', 'mod']},
    {name: 'laso', category: 'color', gloss: 'blue/green', grammar: ['mod']},
    {name: 'lawa', gloss: 'head', grammar: ['n', 'mod', 'vt']},
    {name: 'len', gloss: 'cloth', grammar: ['n']},
    {name: 'lete', gloss: 'cold', grammar: ['n', 'mod', 'vt']},
    {name: 'li', category: 'separator', gloss: 'is', grammar: ['sep']},
    {name: 'lili', gloss: 'small', grammar: ['mod', 'vt']},
    {name: 'linja', gloss: 'string', grammar: ['n']},
    {name: 'lipu', gloss: 'paper', grammar: ['n']},
    {name: 'loje', category: 'color', gloss: 'red', grammar: ['mod']},
    {name: 'lon', gloss: 'located', grammar: ['prep', 'vi']},
    {name: 'luka', gloss: 'hand', grammar: ['n']},
    {name: 'lukin', gloss: 'see', grammar: ['vt', 'vi', 'mod']},
    {name: 'lupa', gloss: 'hole', grammar: ['n']},
    {name: 'ma', gloss: 'land', grammar: ['n']},
    {name: 'mama', category: 'animal', gloss: 'parent', grammar: ['n', 'mod']},
    {name: 'mani', gloss: 'money', grammar: ['n']},
    {name: 'meli', category: 'animal', gloss: 'female', grammar: ['n', 'mod']},
    {name: 'mi', gloss: 'I/we', grammar: ['n', 'mod']},
    {name: 'mije', category: 'animal', gloss: 'male', grammar: ['n', 'mod']},
    {name: 'moku', gloss: 'food', grammar: ['n', 'vt']},
    {name: 'moli', gloss: 'death', grammar: ['n', 'vi', 'vt', 'mod']},
    {name: 'monsi', gloss: 'back', grammar: ['n', 'mod']},
    {name: 'mu', gloss: 'moo!', grammar: ['interj']},
    {name: 'mun', gloss: 'moon', grammar: ['n', 'mod']},
    {name: 'musi', gloss: 'play', grammar: ['n', 'mod', 'vi', 'vt']},
    {name: 'mute', gloss: 'many', grammar: ['mod', 'n', 'vt']},
    {name: 'namako', gloss: 'extra', grammar: ['n', 'mod']},
    {name: 'nanpa', gloss: 'number', grammar: ['n']},
    {name: 'nasa', gloss: 'crazy', grammar: ['mod', 'vt']},
    {name: 'nasin', gloss: 'manner', grammar: ['n']},
    {name: 'nena', gloss: 'bump', grammar: ['n']},
    {name: 'ni', gloss: 'this', grammar: ['mod']},
    {name: 'nimi', gloss: 'name', grammar: ['n']},
    {name: 'noka', gloss: 'leg', grammar: ['n']},
    {name: 'o', gloss: 'imperative', grammar: ['sep', 'interj']},
    {name: 'oko', gloss: 'eye', grammar: ['n']},
    {name: 'olin', gloss: 'love', grammar: ['n', 'mod', 'vt']},
    {name: 'ona', gloss: 'he/she/it', grammar: ['n', 'mod']},
    {name: 'open', gloss: 'open', grammar: ['vt']},
    {name: 'pakala', gloss: 'destroy', grammar: ['n', 'vt', 'vi', 'interj']},
    {name: 'pali', gloss: 'make', grammar: ['n', 'mod', 'vt', 'vi']},
    {name: 'palisa', gloss: 'rod', grammar: ['n']},
    {name: 'pan', gloss: 'grain', grammar: ['n']},
    {name: 'pana', gloss: 'give', grammar: ['vt', 'n']},
    {name: 'pi', category: 'separator', gloss: 'of', grammar: ['sep']},
    {name: 'pilin', gloss: 'feel', grammar: ['n', 'vi', 'vt']},
    {name: 'pimeja', category: 'color', gloss: 'black', grammar: ['mod', 'n', 'vt']},
    {name: 'pini', gloss: 'end', grammar: ['n', 'mod', 'vt']},
    {name: 'pipi', category: 'animal', gloss: 'insect', grammar: ['n']},
    {name: 'poka', gloss: 'side', grammar: ['n', 'mod', 'prep']},
    {name: 'poki', gloss: 'box', grammar: ['n']},
    {name: 'pona', gloss: 'good', grammar: ['n', 'mod', 'interj', 'vt']},
    {name: 'pu', gloss: 'toki ponist', grammar: ['n', 'mod', 'vi']},
    {name: 'sama', gloss: 'similar', grammar: ['mod', 'prep']},
    {name: 'seli', gloss: 'warm', grammar: ['n', 'mod', 'vt']},
    {name: 'selo', gloss: 'surface', grammar: ['n']},
    {name: 'seme', gloss: 'what', grammar: ['n', 'mod', 'vi']},
    {name: 'sewi', gloss: 'superior', grammar: ['n', 'mod']},
    {name: 'sijelo', gloss: 'body', grammar: ['n']},
    {name: 'sike', gloss: 'circle', grammar: ['n', 'mod']},
    {name: 'sin', gloss: 'new', grammar: ['mod', 'vt']},
    {name: 'sina', gloss: 'you', grammar: ['n', 'mod']},
    {name: 'sinpin', gloss: 'front', grammar: ['n']},
    {name: 'sitelen', gloss: 'draw', grammar: ['n', 'vt']},
    {name: 'sona', gloss: 'wisdom', grammar: ['n', 'vt', 'vi']},
    {name: 'soweli', category: 'animal', gloss: 'mammal', grammar: ['n']},
    {name: 'suli', gloss: 'big', grammar: ['mod', 'vt', 'n']},
    {name: 'suno', gloss: 'light', grammar: ['n']},
    {name: 'supa', gloss: 'table', grammar: ['n']},
    {name: 'suwi', gloss: 'sweet', grammar: ['n', 'mod', 'vt']},
    {name: 'tan', gloss: 'because', grammar: ['prep', 'n']},
    {name: 'taso', gloss: 'only', grammar: ['mod', 'conj']},
    {name: 'tawa', gloss: 'move', grammar: ['prep', 'vi', 'n', 'mod', 'vt']},
    {name: 'telo', gloss: 'liquid', grammar: ['n', 'vt']},
    {name: 'tenpo', gloss: 'time', grammar: ['n']},
    {name: 'toki', gloss: 'talking', grammar: ['n', 'mod', 'vt', 'vi', 'interj']},
    {name: 'tomo', gloss: 'house', grammar: ['n', 'mod']},
    {name: 'tu', gloss: 'two', grammar: ['mod', 'n']},
    {name: 'unpa', gloss: 'sex', grammar: ['n', 'mod', 'vt', 'vi']},
    {name: 'uta', gloss: 'mouth', grammar: ['n', 'mod']},
    {name: 'utala', gloss: 'attack', grammar: ['n', 'vt']},
    {name: 'walo', category: 'color', gloss: 'white', grammar: ['mod', 'n']},
    {name: 'wan', gloss: 'one', grammar: ['mod', 'n', 'vt']},
    {name: 'waso', category: 'animal', gloss: 'bird', grammar: ['n']},
    {name: 'wawa', gloss: 'power', grammar: ['n', 'mod', 'vt']},
    {name: 'weka', gloss: 'away', grammar: ['mod', 'n', 'vt']},
    {name: 'wile', gloss: 'need', grammar: ['vt', 'n', 'mod']},
    {name: '.', category: 'separator', gloss: 'period', type: 'punctuation', grammar: ['punct']},
    {name: '?', category: 'separator', gloss: 'question', type: 'punctuation', grammar: ['punct']},
    {name: '!', category: 'separator', gloss: 'exclamation', type: 'punctuation', grammar: ['punct']},
    {name: ':', category: 'separator', gloss: 'colon', type: 'punctuation', grammar: ['punct']},
    {name: ',', category: 'separator', gloss: 'comma', type: 'punctuation', grammar: ['punct']}
];

var allowedWords = tokiPonaDictionary.map(function(item){
    return item.name;
});
allowedWords.push('ale');
allowedWords.push("'Name'");

exports.httpResp = (req, res) => {
	switch (req.get("content-type")) {
	case "text/plain":
		break;
	default:
		res.status(400).send("Content-Type: text/plain");
	}

	res.status(200).send(sitelenParser.parse(req.body));
}
