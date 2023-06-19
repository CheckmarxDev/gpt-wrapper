package secrets

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	hexChars    = "1234567890abcdefABCDEF"
)

//go:embed regex_rules.json
var regexRules []byte

type entropy struct {
	Group int     `json:"group"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type Multiline struct {
	DetectLineGroup int `json:"detectLineGroup"`
}

type AllowRule struct {
	Description string `json:"description"`
	Regex       string `json:"regex"`
}

type SecretRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Regex       string      `json:"regex"`
	Entropies   []entropy   `json:"entropies"`
	Multiline   Multiline   `json:"multiline"`
	AllowRules  []AllowRule `json:"allowRules"`
	SpecialMask string      `json:"specialMask"`
}

type SecretRules struct {
	Rules      []SecretRule `json:"rules"`
	AllowRules []AllowRule  `json:"allowRules"`
}

type secretRegexp struct {
	QueryName   string
	Regex       *regexp.Regexp
	Multiline   bool
	Entropies   []entropy
	AllowRules  []*regexp.Regexp
	SpecialMask *regexp.Regexp
}

type Result struct {
	QueryName string `json:"queryName"`
	Line      int    `json:"line"`
	FileName  string `json:"fileName"`
	Severity  string `json:"severity"`
}

func loadRegexps() ([]secretRegexp, []*regexp.Regexp, error) {
	var allowRulesRegexes []*regexp.Regexp

	var secretRules SecretRules
	err := json.Unmarshal(regexRules, &secretRules)
	if err != nil {
		return nil, nil, err
	}

	var regexes []secretRegexp
	for _, regexStruct := range secretRules.Rules {
		regex := regexStruct.Regex
		specialMask := regexStruct.SpecialMask
		var allowRules []*regexp.Regexp

		for _, rule := range regexStruct.AllowRules {
			compiledRule, err := regexp.Compile(rule.Regex)
			if err == nil {
				allowRules = append(allowRules, compiledRule)
			}
		}

		var specialMaskCompiled *regexp.Regexp
		if specialMask == "" {
			specialMaskCompiled = nil
		} else {
			specialMaskCompiled, _ = regexp.Compile(specialMask)
		}
		regexCompiled, _ := regexp.Compile(regex)

		secretRegex := &secretRegexp{
			QueryName:   regexStruct.Name,
			Regex:       regexCompiled,
			AllowRules:  allowRules,
			Multiline:   regexStruct.Multiline.DetectLineGroup != 0,
			Entropies:   regexStruct.Entropies,
			SpecialMask: specialMaskCompiled,
		}
		regexes = append(regexes, *secretRegex)
	}

	for _, allowRegex := range secretRules.AllowRules {
		compiledRule, err := regexp.Compile(allowRegex.Regex)
		if err == nil {
			allowRulesRegexes = append(allowRulesRegexes, compiledRule)
		}
	}

	return regexes, allowRulesRegexes, nil
}

// getLineNumber calculates the line number based on the match index
func getLineNumber(str string, index int) int {
	lineNumber := 1
	for i := 0; i < index; i++ {
		if str[i] == '\n' && i != index-1 {
			lineNumber++
		}
	}
	return lineNumber
}

func getLines(str string, firstLine int, lastLine int) string {
	lineNumber := 1
	var returnLines []byte
	for i := 0; i < len(str) && lineNumber <= lastLine; i++ {
		if lineNumber >= firstLine && (str[i] != '\n' || lineNumber < lastLine) {
			returnLines = append(returnLines, str[i])
		}
		if str[i] == '\n' {
			lineNumber++
		}

	}
	return string(returnLines)
}

// checkEntropyInterval - verifies if a given token's entropy is within expected bounds
func checkEntropyInterval(entropy entropy, token string) (isEntropyInInterval bool, entropyLevel float64) {
	base64Entropy := calculateEntropy(token, base64Chars)
	hexEntropy := calculateEntropy(token, hexChars)
	highestEntropy := math.Max(base64Entropy, hexEntropy)
	if insideInterval(entropy, base64Entropy) || insideInterval(entropy, hexEntropy) {
		return true, highestEntropy
	}
	return false, highestEntropy
}

func insideInterval(entropy entropy, floatEntropy float64) bool {
	return floatEntropy >= entropy.Min && floatEntropy <= entropy.Max
}

// calculateEntropy - calculates the entropy of a string based on the Shannon formula
func calculateEntropy(token, charSet string) float64 {
	if token == "" {
		return 0
	}
	charMap := map[rune]float64{}
	for _, char := range token {
		if strings.Contains(charSet, string(char)) {
			charMap[char]++
		}
	}

	var freq float64
	length := float64(len(token))
	for _, count := range charMap {
		freq += count * math.Log2(count)
	}

	return math.Log2(length) - freq/length
}

func replaceMatches(result string, regexps []secretRegexp, allowRegexes []*regexp.Regexp) string {

	var multilineRegexes []secretRegexp

	lines := strings.Split(strings.ReplaceAll(result, "\r\n", "\n"), "\n")
	// Replace matches
	for _, re := range regexps {
		if re.Multiline {
			multilineRegexes = append(multilineRegexes, re)
		}
		for index, line := range lines {
			lines[index] = re.Regex.ReplaceAllStringFunc(line, func(match string) string {
				for _, allowRule := range append(re.AllowRules, allowRegexes...) {
					if allowRule.FindString(line) != "" {
						return match
					}
				}

				groups := re.Regex.FindAllStringSubmatch(result, -1)

				for _, entropy := range re.Entropies {
					if len(groups) < entropy.Group {
						if ok, _ := checkEntropyInterval(entropy, groups[0][entropy.Group]); !ok {
							return match
						}
					}
				}

				startOfMatch := ""
				if re.SpecialMask != nil {
					startOfMatch = re.SpecialMask.FindString(line)
				}
				maskedSecret := fmt.Sprintf("%s<masked>", startOfMatch)
				return maskedSecret
			})
		}
	}
	result = strings.Join(lines[:], "\n")
	for _, re := range multilineRegexes {
		// Find all matches of the regular expression in the string
		match := re.Regex.FindStringIndex(result)

		// Iterate over each match
		for match != nil {
			firstLine := getLineNumber(result, match[0])
			lastLine := getLineNumber(result, match[1])
			fullContext := getLines(result, firstLine, lastLine)
			allowed := false

			for _, allowRule := range append(re.AllowRules, allowRegexes...) {
				if allowRule.FindString(fullContext) != "" {
					allowed = true
					break
				}
			}
			if allowed { // Allowed by the allowRules of the regex
				match = nil
				continue
			}

			// Extract the matched substring
			matchString := result[match[0]:match[1]]

			startOfMatch := ""
			if re.SpecialMask != nil {
				partOfMatches := re.SpecialMask.FindAllStringIndex(matchString, -1)
				if len(partOfMatches) != 0 {
					partOfMatch := partOfMatches[len(partOfMatches)-1]
					startOfMatch = matchString[0:partOfMatch[1]]
				}
			}
			maskedSecret := fmt.Sprintf("%s<masked>", startOfMatch)
			result = strings.ReplaceAll(result, matchString, maskedSecret)
			match = re.Regex.FindStringIndex(result)
		}
	}
	return result
}

func MaskSecrets(fileContent string) (string, error) {
	rs, allowRs, err := loadRegexps()
	if err != nil {
		return "", err
	}

	maskedResult := replaceMatches(fileContent, rs, allowRs)

	return maskedResult, nil
}
