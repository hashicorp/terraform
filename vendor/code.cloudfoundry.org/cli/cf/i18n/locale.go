package i18n

import (
	"path"
	"strings"

	"code.cloudfoundry.org/cli/cf/resources"
	"github.com/nicksnyder/go-i18n/i18n/language"
)

func SupportedLocales() []string {
	languages := supportedLanguages()
	localeNames := make([]string, len(languages))

	for i, l := range languages {
		localeParts := strings.Split(l.String(), "-")
		lang := localeParts[0]
		regionOrScript := localeParts[1]

		switch len(regionOrScript) {
		case 2: // Region
			localeNames[i] = lang + "-" + strings.ToUpper(regionOrScript)
		case 4: // Script
			localeNames[i] = lang + "-" + strings.Title(regionOrScript)
		default:
			localeNames[i] = l.String()
		}
	}

	return localeNames
}

func IsSupportedLocale(locale string) bool {
	for _, supportedLanguage := range supportedLanguages() {
		for _, l := range language.Parse(locale) {
			if supportedLanguage.String() == l.String() {
				return true
			}
		}
	}

	return false
}

func supportedLanguages() []*language.Language {
	assetNames := resources.AssetNames()
	languages := []*language.Language{}

	for _, assetName := range assetNames {
		assetLocale := strings.TrimSuffix(path.Base(assetName), resourceSuffix)
		languages = append(languages, language.Parse(assetLocale)...)
	}

	return languages
}
