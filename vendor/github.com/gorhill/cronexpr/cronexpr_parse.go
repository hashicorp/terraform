/*!
 * Copyright 2013 Raymond Hill
 *
 * Project: github.com/gorhill/cronexpr
 * File: cronexpr_parse.go
 * Version: 1.0
 * License: pick the one which suits you best:
 *   GPL v3 see <https://www.gnu.org/licenses/gpl.html>
 *   APL v2 see <http://www.apache.org/licenses/LICENSE-2.0>
 *
 */

package cronexpr

/******************************************************************************/

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

/******************************************************************************/

var (
	genericDefaultList = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
		20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
		30, 31, 32, 33, 34, 35, 36, 37, 38, 39,
		40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
		50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	}
	yearDefaultList = []int{
		1970, 1971, 1972, 1973, 1974, 1975, 1976, 1977, 1978, 1979,
		1980, 1981, 1982, 1983, 1984, 1985, 1986, 1987, 1988, 1989,
		1990, 1991, 1992, 1993, 1994, 1995, 1996, 1997, 1998, 1999,
		2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009,
		2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019,
		2020, 2021, 2022, 2023, 2024, 2025, 2026, 2027, 2028, 2029,
		2030, 2031, 2032, 2033, 2034, 2035, 2036, 2037, 2038, 2039,
		2040, 2041, 2042, 2043, 2044, 2045, 2046, 2047, 2048, 2049,
		2050, 2051, 2052, 2053, 2054, 2055, 2056, 2057, 2058, 2059,
		2060, 2061, 2062, 2063, 2064, 2065, 2066, 2067, 2068, 2069,
		2070, 2071, 2072, 2073, 2074, 2075, 2076, 2077, 2078, 2079,
		2080, 2081, 2082, 2083, 2084, 2085, 2086, 2087, 2088, 2089,
		2090, 2091, 2092, 2093, 2094, 2095, 2096, 2097, 2098, 2099,
	}
)

/******************************************************************************/

var (
	numberTokens = map[string]int{
		"0": 0, "1": 1, "2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9,
		"00": 0, "01": 1, "02": 2, "03": 3, "04": 4, "05": 5, "06": 6, "07": 7, "08": 8, "09": 9,
		"10": 10, "11": 11, "12": 12, "13": 13, "14": 14, "15": 15, "16": 16, "17": 17, "18": 18, "19": 19,
		"20": 20, "21": 21, "22": 22, "23": 23, "24": 24, "25": 25, "26": 26, "27": 27, "28": 28, "29": 29,
		"30": 30, "31": 31, "32": 32, "33": 33, "34": 34, "35": 35, "36": 36, "37": 37, "38": 38, "39": 39,
		"40": 40, "41": 41, "42": 42, "43": 43, "44": 44, "45": 45, "46": 46, "47": 47, "48": 48, "49": 49,
		"50": 50, "51": 51, "52": 52, "53": 53, "54": 54, "55": 55, "56": 56, "57": 57, "58": 58, "59": 59,
		"1970": 1970, "1971": 1971, "1972": 1972, "1973": 1973, "1974": 1974, "1975": 1975, "1976": 1976, "1977": 1977, "1978": 1978, "1979": 1979,
		"1980": 1980, "1981": 1981, "1982": 1982, "1983": 1983, "1984": 1984, "1985": 1985, "1986": 1986, "1987": 1987, "1988": 1988, "1989": 1989,
		"1990": 1990, "1991": 1991, "1992": 1992, "1993": 1993, "1994": 1994, "1995": 1995, "1996": 1996, "1997": 1997, "1998": 1998, "1999": 1999,
		"2000": 2000, "2001": 2001, "2002": 2002, "2003": 2003, "2004": 2004, "2005": 2005, "2006": 2006, "2007": 2007, "2008": 2008, "2009": 2009,
		"2010": 2010, "2011": 2011, "2012": 2012, "2013": 2013, "2014": 2014, "2015": 2015, "2016": 2016, "2017": 2017, "2018": 2018, "2019": 2019,
		"2020": 2020, "2021": 2021, "2022": 2022, "2023": 2023, "2024": 2024, "2025": 2025, "2026": 2026, "2027": 2027, "2028": 2028, "2029": 2029,
		"2030": 2030, "2031": 2031, "2032": 2032, "2033": 2033, "2034": 2034, "2035": 2035, "2036": 2036, "2037": 2037, "2038": 2038, "2039": 2039,
		"2040": 2040, "2041": 2041, "2042": 2042, "2043": 2043, "2044": 2044, "2045": 2045, "2046": 2046, "2047": 2047, "2048": 2048, "2049": 2049,
		"2050": 2050, "2051": 2051, "2052": 2052, "2053": 2053, "2054": 2054, "2055": 2055, "2056": 2056, "2057": 2057, "2058": 2058, "2059": 2059,
		"2060": 2060, "2061": 2061, "2062": 2062, "2063": 2063, "2064": 2064, "2065": 2065, "2066": 2066, "2067": 2067, "2068": 2068, "2069": 2069,
		"2070": 2070, "2071": 2071, "2072": 2072, "2073": 2073, "2074": 2074, "2075": 2075, "2076": 2076, "2077": 2077, "2078": 2078, "2079": 2079,
		"2080": 2080, "2081": 2081, "2082": 2082, "2083": 2083, "2084": 2084, "2085": 2085, "2086": 2086, "2087": 2087, "2088": 2088, "2089": 2089,
		"2090": 2090, "2091": 2091, "2092": 2092, "2093": 2093, "2094": 2094, "2095": 2095, "2096": 2096, "2097": 2097, "2098": 2098, "2099": 2099,
	}
	monthTokens = map[string]int{
		`1`: 1, `jan`: 1, `january`: 1,
		`2`: 2, `feb`: 2, `february`: 2,
		`3`: 3, `mar`: 3, `march`: 3,
		`4`: 4, `apr`: 4, `april`: 4,
		`5`: 5, `may`: 5,
		`6`: 6, `jun`: 6, `june`: 6,
		`7`: 7, `jul`: 7, `july`: 7,
		`8`: 8, `aug`: 8, `august`: 8,
		`9`: 9, `sep`: 9, `september`: 9,
		`10`: 10, `oct`: 10, `october`: 10,
		`11`: 11, `nov`: 11, `november`: 11,
		`12`: 12, `dec`: 12, `december`: 12,
	}
	dowTokens = map[string]int{
		`0`: 0, `sun`: 0, `sunday`: 0,
		`1`: 1, `mon`: 1, `monday`: 1,
		`2`: 2, `tue`: 2, `tuesday`: 2,
		`3`: 3, `wed`: 3, `wednesday`: 3,
		`4`: 4, `thu`: 4, `thursday`: 4,
		`5`: 5, `fri`: 5, `friday`: 5,
		`6`: 6, `sat`: 6, `saturday`: 6,
		`7`: 0,
	}
)

/******************************************************************************/

func atoi(s string) int {
	return numberTokens[s]
}

type fieldDescriptor struct {
	name         string
	min, max     int
	defaultList  []int
	valuePattern string
	atoi         func(string) int
}

var (
	secondDescriptor = fieldDescriptor{
		name:         "second",
		min:          0,
		max:          59,
		defaultList:  genericDefaultList[0:60],
		valuePattern: `0?[0-9]|[1-5][0-9]`,
		atoi:         atoi,
	}
	minuteDescriptor = fieldDescriptor{
		name:         "minute",
		min:          0,
		max:          59,
		defaultList:  genericDefaultList[0:60],
		valuePattern: `0?[0-9]|[1-5][0-9]`,
		atoi:         atoi,
	}
	hourDescriptor = fieldDescriptor{
		name:         "hour",
		min:          0,
		max:          23,
		defaultList:  genericDefaultList[0:24],
		valuePattern: `0?[0-9]|1[0-9]|2[0-3]`,
		atoi:         atoi,
	}
	domDescriptor = fieldDescriptor{
		name:         "day-of-month",
		min:          1,
		max:          31,
		defaultList:  genericDefaultList[1:32],
		valuePattern: `0?[1-9]|[12][0-9]|3[01]`,
		atoi:         atoi,
	}
	monthDescriptor = fieldDescriptor{
		name:         "month",
		min:          1,
		max:          12,
		defaultList:  genericDefaultList[1:13],
		valuePattern: `0?[1-9]|1[012]|jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec|january|february|march|april|march|april|june|july|august|september|october|november|december`,
		atoi: func(s string) int {
			return monthTokens[s]
		},
	}
	dowDescriptor = fieldDescriptor{
		name:         "day-of-week",
		min:          0,
		max:          6,
		defaultList:  genericDefaultList[0:7],
		valuePattern: `0?[0-7]|sun|mon|tue|wed|thu|fri|sat|sunday|monday|tuesday|wednesday|thursday|friday|saturday`,
		atoi: func(s string) int {
			return dowTokens[s]
		},
	}
	yearDescriptor = fieldDescriptor{
		name:         "year",
		min:          1970,
		max:          2099,
		defaultList:  yearDefaultList[:],
		valuePattern: `19[789][0-9]|20[0-9]{2}`,
		atoi:         atoi,
	}
)

/******************************************************************************/

var (
	layoutWildcard            = `^\*$|^\?$`
	layoutValue               = `^(%value%)$`
	layoutRange               = `^(%value%)-(%value%)$`
	layoutWildcardAndInterval = `^\*/(\d+)$`
	layoutValueAndInterval    = `^(%value%)/(\d+)$`
	layoutRangeAndInterval    = `^(%value%)-(%value%)/(\d+)$`
	layoutLastDom             = `^l$`
	layoutWorkdom             = `^(%value%)w$`
	layoutLastWorkdom         = `^lw$`
	layoutDowOfLastWeek       = `^(%value%)l$`
	layoutDowOfSpecificWeek   = `^(%value%)#([1-5])$`
	fieldFinder               = regexp.MustCompile(`\S+`)
	entryFinder               = regexp.MustCompile(`[^,]+`)
	layoutRegexp              = make(map[string]*regexp.Regexp)
)

/******************************************************************************/

var cronNormalizer = strings.NewReplacer(
	"@yearly", "0 0 0 1 1 * *",
	"@annually", "0 0 0 1 1 * *",
	"@monthly", "0 0 0 1 * * *",
	"@weekly", "0 0 0 * * 0 *",
	"@daily", "0 0 0 * * * *",
	"@hourly", "0 0 * * * * *")

/******************************************************************************/

func (expr *Expression) secondFieldHandler(s string) error {
	var err error
	expr.secondList, err = genericFieldHandler(s, secondDescriptor)
	return err
}

/******************************************************************************/

func (expr *Expression) minuteFieldHandler(s string) error {
	var err error
	expr.minuteList, err = genericFieldHandler(s, minuteDescriptor)
	return err
}

/******************************************************************************/

func (expr *Expression) hourFieldHandler(s string) error {
	var err error
	expr.hourList, err = genericFieldHandler(s, hourDescriptor)
	return err
}

/******************************************************************************/

func (expr *Expression) monthFieldHandler(s string) error {
	var err error
	expr.monthList, err = genericFieldHandler(s, monthDescriptor)
	return err
}

/******************************************************************************/

func (expr *Expression) yearFieldHandler(s string) error {
	var err error
	expr.yearList, err = genericFieldHandler(s, yearDescriptor)
	return err
}

/******************************************************************************/

const (
	none = 0
	one  = 1
	span = 2
	all  = 3
)

type cronDirective struct {
	kind  int
	first int
	last  int
	step  int
	sbeg  int
	send  int
}

func genericFieldHandler(s string, desc fieldDescriptor) ([]int, error) {
	directives, err := genericFieldParse(s, desc)
	if err != nil {
		return nil, err
	}
	values := make(map[int]bool)
	for _, directive := range directives {
		switch directive.kind {
		case none:
			return nil, fmt.Errorf("syntax error in %s field: '%s'", desc.name, s[directive.sbeg:directive.send])
		case one:
			populateOne(values, directive.first)
		case span:
			populateMany(values, directive.first, directive.last, directive.step)
		case all:
			return desc.defaultList, nil
		}
	}
	return toList(values), nil
}

func (expr *Expression) dowFieldHandler(s string) error {
	expr.daysOfWeekRestricted = true
	expr.daysOfWeek = make(map[int]bool)
	expr.lastWeekDaysOfWeek = make(map[int]bool)
	expr.specificWeekDaysOfWeek = make(map[int]bool)

	directives, err := genericFieldParse(s, dowDescriptor)
	if err != nil {
		return err
	}

	for _, directive := range directives {
		switch directive.kind {
		case none:
			sdirective := s[directive.sbeg:directive.send]
			snormal := strings.ToLower(sdirective)
			// `5L`
			pairs := makeLayoutRegexp(layoutDowOfLastWeek, dowDescriptor.valuePattern).FindStringSubmatchIndex(snormal)
			if len(pairs) > 0 {
				populateOne(expr.lastWeekDaysOfWeek, dowDescriptor.atoi(snormal[pairs[2]:pairs[3]]))
			} else {
				// `5#3`
				pairs := makeLayoutRegexp(layoutDowOfSpecificWeek, dowDescriptor.valuePattern).FindStringSubmatchIndex(snormal)
				if len(pairs) > 0 {
					populateOne(expr.specificWeekDaysOfWeek, (dowDescriptor.atoi(snormal[pairs[4]:pairs[5]])-1)*7+(dowDescriptor.atoi(snormal[pairs[2]:pairs[3]])%7))
				} else {
					return fmt.Errorf("syntax error in day-of-week field: '%s'", sdirective)
				}
			}
		case one:
			populateOne(expr.daysOfWeek, directive.first)
		case span:
			populateMany(expr.daysOfWeek, directive.first, directive.last, directive.step)
		case all:
			populateMany(expr.daysOfWeek, directive.first, directive.last, directive.step)
			expr.daysOfWeekRestricted = false
		}
	}
	return nil
}

func (expr *Expression) domFieldHandler(s string) error {
	expr.daysOfMonthRestricted = true
	expr.lastDayOfMonth = false
	expr.lastWorkdayOfMonth = false
	expr.daysOfMonth = make(map[int]bool)     // days of month map
	expr.workdaysOfMonth = make(map[int]bool) // work days of month map

	directives, err := genericFieldParse(s, domDescriptor)
	if err != nil {
		return err
	}

	for _, directive := range directives {
		switch directive.kind {
		case none:
			sdirective := s[directive.sbeg:directive.send]
			snormal := strings.ToLower(sdirective)
			// `L`
			if makeLayoutRegexp(layoutLastDom, domDescriptor.valuePattern).MatchString(snormal) {
				expr.lastDayOfMonth = true
			} else {
				// `LW`
				if makeLayoutRegexp(layoutLastWorkdom, domDescriptor.valuePattern).MatchString(snormal) {
					expr.lastWorkdayOfMonth = true
				} else {
					// `15W`
					pairs := makeLayoutRegexp(layoutWorkdom, domDescriptor.valuePattern).FindStringSubmatchIndex(snormal)
					if len(pairs) > 0 {
						populateOne(expr.workdaysOfMonth, domDescriptor.atoi(snormal[pairs[2]:pairs[3]]))
					} else {
						return fmt.Errorf("syntax error in day-of-month field: '%s'", sdirective)
					}
				}
			}
		case one:
			populateOne(expr.daysOfMonth, directive.first)
		case span:
			populateMany(expr.daysOfMonth, directive.first, directive.last, directive.step)
		case all:
			populateMany(expr.daysOfMonth, directive.first, directive.last, directive.step)
			expr.daysOfMonthRestricted = false
		}
	}
	return nil
}

/******************************************************************************/

func populateOne(values map[int]bool, v int) {
	values[v] = true
}

func populateMany(values map[int]bool, min, max, step int) {
	for i := min; i <= max; i += step {
		values[i] = true
	}
}

func toList(set map[int]bool) []int {
	list := make([]int, len(set))
	i := 0
	for k := range set {
		list[i] = k
		i += 1
	}
	sort.Ints(list)
	return list
}

/******************************************************************************/

func genericFieldParse(s string, desc fieldDescriptor) ([]*cronDirective, error) {
	// At least one entry must be present
	indices := entryFinder.FindAllStringIndex(s, -1)
	if len(indices) == 0 {
		return nil, fmt.Errorf("%s field: missing directive", desc.name)
	}

	directives := make([]*cronDirective, 0, len(indices))

	for i := range indices {
		directive := cronDirective{
			sbeg: indices[i][0],
			send: indices[i][1],
		}
		snormal := strings.ToLower(s[indices[i][0]:indices[i][1]])

		// `*`
		if makeLayoutRegexp(layoutWildcard, desc.valuePattern).MatchString(snormal) {
			directive.kind = all
			directive.first = desc.min
			directive.last = desc.max
			directive.step = 1
			directives = append(directives, &directive)
			continue
		}
		// `5`
		if makeLayoutRegexp(layoutValue, desc.valuePattern).MatchString(snormal) {
			directive.kind = one
			directive.first = desc.atoi(snormal)
			directives = append(directives, &directive)
			continue
		}
		// `5-20`
		pairs := makeLayoutRegexp(layoutRange, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			directive.kind = span
			directive.first = desc.atoi(snormal[pairs[2]:pairs[3]])
			directive.last = desc.atoi(snormal[pairs[4]:pairs[5]])
			directive.step = 1
			directives = append(directives, &directive)
			continue
		}
		// `*/2`
		pairs = makeLayoutRegexp(layoutWildcardAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			directive.kind = span
			directive.first = desc.min
			directive.last = desc.max
			directive.step = atoi(snormal[pairs[2]:pairs[3]])
			directives = append(directives, &directive)
			continue
		}
		// `5/2`
		pairs = makeLayoutRegexp(layoutValueAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			directive.kind = span
			directive.first = desc.atoi(snormal[pairs[2]:pairs[3]])
			directive.last = desc.max
			directive.step = atoi(snormal[pairs[4]:pairs[5]])
			directives = append(directives, &directive)
			continue
		}
		// `5-20/2`
		pairs = makeLayoutRegexp(layoutRangeAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			directive.kind = span
			directive.first = desc.atoi(snormal[pairs[2]:pairs[3]])
			directive.last = desc.atoi(snormal[pairs[4]:pairs[5]])
			directive.step = atoi(snormal[pairs[6]:pairs[7]])
			directives = append(directives, &directive)
			continue
		}
		// No behavior for this one, let caller deal with it
		directive.kind = none
		directives = append(directives, &directive)
	}
	return directives, nil
}

/******************************************************************************/

func makeLayoutRegexp(layout, value string) *regexp.Regexp {
	layout = strings.Replace(layout, `%value%`, value, -1)
	re := layoutRegexp[layout]
	if re == nil {
		re = regexp.MustCompile(layout)
		layoutRegexp[layout] = re
	}
	return re
}
