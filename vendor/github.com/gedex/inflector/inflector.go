// Copyright 2013 Akeda Bagus <admin@gedex.web.id>. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package inflector pluralizes and singularizes English nouns.

There are only two exported functions: `Pluralize` and `Singularize`.

	s := "People"
	fmt.Println(inflector.Singularize(s)) // will print "Person"

	s2 := "octopus"
	fmt.Println(inflector.Pluralize(s2)) // will print "octopuses"

*/
package inflector

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Rule represents name of the inflector rule, can be
// Plural or Singular
type Rule int

const (
	Plural = iota
	Singular
)

// InflectorRule represents inflector rule
type InflectorRule struct {
	Rules               []*ruleItem
	Irregular           []*irregularItem
	Uninflected         []string
	compiledIrregular   *regexp.Regexp
	compiledUninflected *regexp.Regexp
	compiledRules       []*compiledRule
}

type ruleItem struct {
	pattern     string
	replacement string
}

type irregularItem struct {
	word        string
	replacement string
}

// compiledRule represents compiled version of Inflector.Rules.
type compiledRule struct {
	replacement string
	*regexp.Regexp
}

// threadsafe access to rules and caches
var mutex sync.Mutex
var rules = make(map[Rule]*InflectorRule)

// Words that should not be inflected
var uninflected = []string{
	`Amoyese`, `bison`, `Borghese`, `bream`, `breeches`, `britches`, `buffalo`,
	`cantus`, `carp`, `chassis`, `clippers`, `cod`, `coitus`, `Congoese`,
	`contretemps`, `corps`, `debris`, `diabetes`, `djinn`, `eland`, `elk`,
	`equipment`, `Faroese`, `flounder`, `Foochowese`, `gallows`, `Genevese`,
	`Genoese`, `Gilbertese`, `graffiti`, `headquarters`, `herpes`, `hijinks`,
	`Hottentotese`, `information`, `innings`, `jackanapes`, `Kiplingese`,
	`Kongoese`, `Lucchese`, `mackerel`, `Maltese`, `.*?media`, `mews`, `moose`,
	`mumps`, `Nankingese`, `news`, `nexus`, `Niasese`, `Pekingese`,
	`Piedmontese`, `pincers`, `Pistoiese`, `pliers`, `Portuguese`, `proceedings`,
	`rabies`, `rice`, `rhinoceros`, `salmon`, `Sarawakese`, `scissors`,
	`sea[- ]bass`, `series`, `Shavese`, `shears`, `siemens`, `species`, `swine`,
	`testes`, `trousers`, `trout`, `tuna`, `Vermontese`, `Wenchowese`, `whiting`,
	`wildebeest`, `Yengeese`,
}

// Plural words that should not be inflected
var uninflectedPlurals = []string{
	`.*[nrlm]ese`, `.*deer`, `.*fish`, `.*measles`, `.*ois`, `.*pox`, `.*sheep`,
	`people`,
}

// Singular words that should not be inflected
var uninflectedSingulars = []string{
	`.*[nrlm]ese`, `.*deer`, `.*fish`, `.*measles`, `.*ois`, `.*pox`, `.*sheep`,
	`.*ss`,
}

type cache map[string]string

// Inflected words that already cached for immediate retrieval from a given Rule
var caches = make(map[Rule]cache)

// map of irregular words where its key is a word and its value is the replacement
var irregularMaps = make(map[Rule]cache)

func init() {

	rules[Plural] = &InflectorRule{
		Rules: []*ruleItem{
			{`(?i)(s)tatus$`, `${1}${2}tatuses`},
			{`(?i)(quiz)$`, `${1}zes`},
			{`(?i)^(ox)$`, `${1}${2}en`},
			{`(?i)([m|l])ouse$`, `${1}ice`},
			{`(?i)(matr|vert|ind)(ix|ex)$`, `${1}ices`},
			{`(?i)(x|ch|ss|sh)$`, `${1}es`},
			{`(?i)([^aeiouy]|qu)y$`, `${1}ies`},
			{`(?i)(hive)$`, `$1s`},
			{`(?i)(?:([^f])fe|([lre])f)$`, `${1}${2}ves`},
			{`(?i)sis$`, `ses`},
			{`(?i)([ti])um$`, `${1}a`},
			{`(?i)(p)erson$`, `${1}eople`},
			{`(?i)(m)an$`, `${1}en`},
			{`(?i)(c)hild$`, `${1}hildren`},
			{`(?i)(buffal|tomat)o$`, `${1}${2}oes`},
			{`(?i)(alumn|bacill|cact|foc|fung|nucle|radi|stimul|syllab|termin|vir)us$`, `${1}i`},
			{`(?i)us$`, `uses`},
			{`(?i)(alias)$`, `${1}es`},
			{`(?i)(ax|cris|test)is$`, `${1}es`},
			{`s$`, `s`},
			{`^$`, ``},
			{`$`, `s`},
		},
		Irregular: []*irregularItem{
			{`atlas`, `atlases`},
			{`beef`, `beefs`},
			{`brother`, `brothers`},
			{`cafe`, `cafes`},
			{`child`, `children`},
			{`cookie`, `cookies`},
			{`corpus`, `corpuses`},
			{`cow`, `cows`},
			{`ganglion`, `ganglions`},
			{`genie`, `genies`},
			{`genus`, `genera`},
			{`graffito`, `graffiti`},
			{`hoof`, `hoofs`},
			{`loaf`, `loaves`},
			{`man`, `men`},
			{`money`, `monies`},
			{`mongoose`, `mongooses`},
			{`move`, `moves`},
			{`mythos`, `mythoi`},
			{`niche`, `niches`},
			{`numen`, `numina`},
			{`occiput`, `occiputs`},
			{`octopus`, `octopuses`},
			{`opus`, `opuses`},
			{`ox`, `oxen`},
			{`penis`, `penises`},
			{`person`, `people`},
			{`sex`, `sexes`},
			{`soliloquy`, `soliloquies`},
			{`testis`, `testes`},
			{`trilby`, `trilbys`},
			{`turf`, `turfs`},
			{`potato`, `potatoes`},
			{`hero`, `heroes`},
			{`tooth`, `teeth`},
			{`goose`, `geese`},
			{`foot`, `feet`},
		},
	}
	prepare(Plural)

	rules[Singular] = &InflectorRule{
		Rules: []*ruleItem{
			{`(?i)(s)tatuses$`, `${1}${2}tatus`},
			{`(?i)^(.*)(menu)s$`, `${1}${2}`},
			{`(?i)(quiz)zes$`, `$1`},
			{`(?i)(matr)ices$`, `${1}ix`},
			{`(?i)(vert|ind)ices$`, `${1}ex`},
			{`(?i)^(ox)en`, `$1`},
			{`(?i)(alias)(es)*$`, `$1`},
			{`(?i)(alumn|bacill|cact|foc|fung|nucle|radi|stimul|syllab|termin|viri?)i$`, `${1}us`},
			{`(?i)([ftw]ax)es`, `$1`},
			{`(?i)(cris|ax|test)es$`, `${1}is`},
			{`(?i)(shoe|slave)s$`, `$1`},
			{`(?i)(o)es$`, `$1`},
			{`ouses$`, `ouse`},
			{`([^a])uses$`, `${1}us`},
			{`(?i)([m|l])ice$`, `${1}ouse`},
			{`(?i)(x|ch|ss|sh)es$`, `$1`},
			{`(?i)(m)ovies$`, `${1}${2}ovie`},
			{`(?i)(s)eries$`, `${1}${2}eries`},
			{`(?i)([^aeiouy]|qu)ies$`, `${1}y`},
			{`(?i)(tive)s$`, `$1`},
			{`(?i)([lre])ves$`, `${1}f`},
			{`(?i)([^fo])ves$`, `${1}fe`},
			{`(?i)(hive)s$`, `$1`},
			{`(?i)(drive)s$`, `$1`},
			{`(?i)(^analy)ses$`, `${1}sis`},
			{`(?i)(analy|diagno|^ba|(p)arenthe|(p)rogno|(s)ynop|(t)he)ses$`, `${1}${2}sis`},
			{`(?i)([ti])a$`, `${1}um`},
			{`(?i)(p)eople$`, `${1}${2}erson`},
			{`(?i)(m)en$`, `${1}an`},
			{`(?i)(c)hildren$`, `${1}${2}hild`},
			{`(?i)(n)ews$`, `${1}${2}ews`},
			{`eaus$`, `eau`},
			{`^(.*us)$`, `$1`},
			{`(?i)s$`, ``},
		},
		Irregular: []*irregularItem{
			{`foes`, `foe`},
			{`waves`, `wave`},
			{`curves`, `curve`},
			{`atlases`, `atlas`},
			{`beefs`, `beef`},
			{`brothers`, `brother`},
			{`cafes`, `cafe`},
			{`children`, `child`},
			{`cookies`, `cookie`},
			{`corpuses`, `corpus`},
			{`cows`, `cow`},
			{`ganglions`, `ganglion`},
			{`genies`, `genie`},
			{`genera`, `genus`},
			{`graffiti`, `graffito`},
			{`hoofs`, `hoof`},
			{`loaves`, `loaf`},
			{`men`, `man`},
			{`monies`, `money`},
			{`mongooses`, `mongoose`},
			{`moves`, `move`},
			{`mythoi`, `mythos`},
			{`niches`, `niche`},
			{`numina`, `numen`},
			{`occiputs`, `occiput`},
			{`octopuses`, `octopus`},
			{`opuses`, `opus`},
			{`oxen`, `ox`},
			{`penises`, `penis`},
			{`people`, `person`},
			{`sexes`, `sex`},
			{`soliloquies`, `soliloquy`},
			{`testes`, `testis`},
			{`trilbys`, `trilby`},
			{`turfs`, `turf`},
			{`potatoes`, `potato`},
			{`heroes`, `hero`},
			{`teeth`, `tooth`},
			{`geese`, `goose`},
			{`feet`, `foot`},
		},
	}
	prepare(Singular)
}

// prepare rule, e.g., compile the pattern.
func prepare(r Rule) error {
	var reString string

	switch r {
	case Plural:
		// Merge global uninflected with singularsUninflected
		rules[r].Uninflected = merge(uninflected, uninflectedPlurals)
	case Singular:
		// Merge global uninflected with singularsUninflected
		rules[r].Uninflected = merge(uninflected, uninflectedSingulars)
	}

	// Set InflectorRule.compiledUninflected by joining InflectorRule.Uninflected into
	// a single string then compile it.
	reString = fmt.Sprintf(`(?i)(^(?:%s))$`, strings.Join(rules[r].Uninflected, `|`))
	rules[r].compiledUninflected = regexp.MustCompile(reString)

	// Prepare irregularMaps
	irregularMaps[r] = make(cache, len(rules[r].Irregular))

	// Set InflectorRule.compiledIrregular by joining the irregularItem.word of Inflector.Irregular
	// into a single string then compile it.
	vIrregulars := make([]string, len(rules[r].Irregular))
	for i, item := range rules[r].Irregular {
		vIrregulars[i] = item.word
		irregularMaps[r][item.word] = item.replacement
	}
	reString = fmt.Sprintf(`(?i)(.*)\b((?:%s))$`, strings.Join(vIrregulars, `|`))
	rules[r].compiledIrregular = regexp.MustCompile(reString)

	// Compile all patterns in InflectorRule.Rules
	rules[r].compiledRules = make([]*compiledRule, len(rules[r].Rules))
	for i, item := range rules[r].Rules {
		rules[r].compiledRules[i] = &compiledRule{item.replacement, regexp.MustCompile(item.pattern)}
	}

	// Prepare caches
	caches[r] = make(cache)

	return nil
}

// merge slice a and slice b
func merge(a []string, b []string) []string {
	result := make([]string, len(a)+len(b))
	copy(result, a)
	copy(result[len(a):], b)

	return result
}

// Pluralize returns string s in plural form.
func Pluralize(s string) string {
	return getInflected(Plural, s)
}

// Singularize returns string s in singular form.
func Singularize(s string) string {
	return getInflected(Singular, s)
}

func getInflected(r Rule, s string) string {
	mutex.Lock()
	defer mutex.Unlock()
	if v, ok := caches[r][s]; ok {
		return v
	}

	// Check for irregular words
	if res := rules[r].compiledIrregular.FindStringSubmatch(s); len(res) >= 3 {
		var buf bytes.Buffer

		buf.WriteString(res[1])
		buf.WriteString(s[0:1])
		buf.WriteString(irregularMaps[r][strings.ToLower(res[2])][1:])

		// Cache it then returns
		caches[r][s] = buf.String()
		return caches[r][s]
	}

	// Check for uninflected words
	if rules[r].compiledUninflected.MatchString(s) {
		caches[r][s] = s
		return caches[r][s]
	}

	// Check each rule
	for _, re := range rules[r].compiledRules {
		if re.MatchString(s) {
			caches[r][s] = re.ReplaceAllString(s, re.replacement)
			return caches[r][s]
		}
	}

	// Returns unaltered
	caches[r][s] = s
	return caches[r][s]
}
