package main

import (
	"fmt"
	"strings"
)

// 注曰「Token is a single lexical element of a 哲浩之谕 program.」
//
// 注曰「哲浩之谕不用空格：化生之名一律书于《》之内（mark），文句书于「」之内（str），」
// 注曰「其余令牌或为哲浩之谕之字（keyword，最长匹配），或为汉字数（numeral 连书）。」
// 注曰「顿号、逗号、句号等中文标点皆为分隔，亦可省略。」
type Token struct {
	Kind string // 注曰「"word" | "str" | "mark"」
	Text string // 注曰「for str: decoded content; for mark: the bare name」
	Line int
}

// 注曰「劫 is a compile/runtime error carrying a 哲浩之谕-style message.」
type 劫 struct{ msg string }

func (e *劫) Error() string { return e.msg }

func 劫f(format string, a ...interface{}) *劫 {
	return &劫{msg: fmt.Sprintf(format, a...)}
}

// 注曰「lexing」

// 注曰「keywordsSorted holds every reserved 哲浩之谕 word, longest first, so that」
// 注曰「maximal-munch matching prefers 逾等 over 逾, 虚谒 over 虚, and so on.」
var keywordsSorted = sortKeywords(strings.Fields(`
生 铭 取 受 双受 众受 加 减 乘 除 余 幂 反 绝 根 整
等 异 逾 逊 逾等 逊等 且 或 非 复 弃 换 凌 转
宣 语 聆 读简 书简 化数 化文 化阴阳 何类
聚 几何 取元 铭元 列附 列抽 剖 缀 天机
典 铭寓 取寓 有寓 除寓
若 否则 毕若 记 至 毕记 历 于 毕历 谒 虚谒 阳谒
术 毕术 归 归源 寂 承 宥 毕宥 哲浩御世 阳 阴 虚 标
`))

func sortKeywords(ws []string) []string {
	out := append([]string(nil), ws...)
	// 注曰「简单插入排序按字数降序（键字甚少，不必讲究）。」
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && len([]rune(out[j])) > len([]rune(out[j-1])); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// 注曰「isSep reports whether r is a punctuation separator (skipped by the lexer).」
func isSep(r rune) bool {
	switch r {
	case '、', '，', '。', '；', '：', '（', '）', '！', '？',
		' ', '\t', '\r', '\n', '\u3000', '\u00a0':
		return true
	}
	return false
}

// 注曰「numeralChars are the runes that may appear inside a Chinese-numeral run.」
const numeralChars = "〇零一二三四五六七八九十百千万亿两负点廿卅"

func isNumeralRune(r rune) bool {
	return strings.ContainsRune(numeralChars, r)
}

// 注曰「tokenize splits 哲浩之谕 source into tokens without requiring spaces.」
func tokenize(src string) ([]Token, error) {
	runes := []rune(src)
	var toks []Token
	i, n := 0, len(runes)
	line := 1
	for i < n {
		c := runes[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if isSep(c) {
			i++
			continue
		}
		// 注曰「注释之式：注曰「……」——以注曰起，继以「」括注文，至与之相合的「」而止」
		// 注曰「注中可复纳「」，以其相合为度；反斜杠可转义其后一字（如 \」），使其不作合璧之解」
		if c == '注' && i+1 < n && runes[i+1] == '曰' {
			i += 2
			if i >= n || runes[i] != '「' {
				return nil, 劫f("第%d行：注曰无「，注文无所附", line)
			}
			startLine := line
			i++
			depth := 1
			for i < n && depth > 0 {
				ch := runes[i]
				if ch == '\n' {
					line++
					i++
					continue
				}
				if ch == '\\' && i+1 < n {
					i += 2
					continue
				}
				if ch == '「' {
					depth++
				} else if ch == '」' {
					depth--
				}
				i++
			}
			if depth > 0 {
				return nil, 劫f("第%d行：注未合璧，「」缺其尾", startLine)
			}
			continue
		}
		if c == '「' {
			j := i + 1
			var buf strings.Builder
			for j < n && runes[j] != '」' {
				ch := runes[j]
				if ch == '\n' {
					line++
				}
				if ch == '\\' && j+1 < n {
					nxt := runes[j+1]
					var out rune
					switch nxt {
					case 'n':
						out = '\n'
					case 't':
						out = '\t'
					case '「':
						out = '「'
					case '」':
						out = '」'
					case '\\':
						out = '\\'
					default:
						out = nxt
					}
					buf.WriteRune(out)
					j += 2
				} else {
					buf.WriteRune(ch)
					j++
				}
			}
			if j >= n {
				return nil, 劫f("第%d行：文未合璧，「」缺其尾", line)
			}
			toks = append(toks, Token{"str", buf.String(), line})
			i = j + 1
			continue
		}
		if c == '《' {
			j := i + 1
			for j < n && runes[j] != '》' && runes[j] != '\n' {
				j++
			}
			if j >= n || runes[j] != '》' {
				return nil, 劫f("第%d行：名号未合，《》缺其尾", line)
			}
			name := string(runes[i+1 : j])
			if name == "" {
				return nil, 劫f("第%d行：《》中空空，无名可称", line)
			}
			toks = append(toks, Token{"mark", name, line})
			i = j + 1
			continue
		}
		// 注曰「哲浩之谕之字（最长匹配）」
		matched := false
		for _, kw := range keywordsSorted {
			kr := []rune(kw)
			if i+len(kr) <= n {
				ok := true
				for k, r := range kr {
					if runes[i+k] != r {
						ok = false
						break
					}
				}
				if ok {
					toks = append(toks, Token{"word", kw, line})
					i += len(kr)
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}
		// 注曰「汉字数（连书，遇非数字即止）」
		if isNumeralRune(c) {
			j := i
			for j < n && isNumeralRune(runes[j]) {
				j++
			}
			toks = append(toks, Token{"word", string(runes[i:j]), line})
			i = j
			continue
		}
		return nil, 劫f("第%d行：未识之字「%c」——名号当书于《》内", line, c)
	}
	return toks, nil
}

// 注曰「numbers」

var cnDigit = map[rune]int64{
	'〇': 0, '零': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4,
	'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
}

var cnUnit = map[rune]int64{
	'十': 10, '百': 100, '千': 1000, '万': 10000, '亿': 100000000,
}

// 注曰「cnInt parses a Chinese-numeral integer (no 负/点).」
func cnInt(s string) (int64, bool) {
	s = strings.ReplaceAll(s, "廿", "二十")
	s = strings.ReplaceAll(s, "卅", "三十")
	var total, section, cur int64
	seen := false
	for _, ch := range s {
		if d, ok := cnDigit[ch]; ok {
			cur = cur*10 + d
			seen = true
			continue
		}
		if u, ok := cnUnit[ch]; ok {
			if u >= 10000 {
				if section+cur > 0 {
					section = (section + cur) * u
				} else {
					section = u
				}
				total += section
				section, cur = 0, 0
			} else {
				if cur != 0 {
					section += cur * u
				} else {
					section += u
				}
				cur = 0
			}
			seen = true
			continue
		}
		return 0, false
	}
	if !seen {
		return 0, false
	}
	return total + section + cur, true
}

// 注曰「numberKind describes a parsed numeric literal.」
type numberKind struct {
	isInt bool
	i64   int64
	f64   float64
}

// 注曰「parseNumber attempts to interpret a word as a numeric literal.」
func parseNumber(w string) (numberKind, bool) {
	if w == "" {
		return numberKind{}, false
	}
	neg := strings.HasPrefix(w, "负")
	s := w
	if neg {
		s = w[len("负"):]
	}
	if idx := strings.IndexRune(s, '点'); idx >= 0 {
		ip, frac := s[:idx], s[idx+len("点"):]
		base, ok := cnInt(ip)
		if !ok {
			base = 0
		}
		if !fracPureDigits(frac) {
			return numberKind{}, false
		}
		var f float64 = float64(base)
		scale := 1.0
		for _, ch := range frac {
			d, _ := cnDigit[ch]
			scale /= 10
			f += float64(d) * scale
		}
		if neg {
			f = -f
		}
		return numberKind{f64: f}, true
	}
	v, ok := cnInt(s)
	if !ok {
		return numberKind{}, false
	}
	if neg {
		v = -v
	}
	return numberKind{isInt: true, i64: v}, true
}

func fracPureDigits(s string) bool {
	for _, ch := range s {
		if _, ok := cnDigit[ch]; !ok {
			return false
		}
	}
	return true
}
