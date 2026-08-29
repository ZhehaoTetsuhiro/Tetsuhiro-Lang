package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 注曰「reserved words of 哲浩之谕 (may not be used as 术/化生 names).」
var reserved = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
生 铭 取 受 双受 众受 加 减 乘 除 余 幂 反 绝 根 整
等 异 逾 逊 逾等 逊等 且 或 非 复 弃 换 凌 转
宣 语 聆 读简 书简 化数 化文 化阴阳 何类
聚 几何 取元 铭元 列附 列抽 剖 缀 天机
典 铭寓 取寓 有寓 除寓
若 否则 毕若 记 至 毕记 历 于 毕历 谒 虚谒 阳谒
术 毕术 归 归源 寂 承 宥 毕宥 哲浩御世 阳 阴 虚 标`) {
		reserved[w] = true
	}
}

// 注曰「Unit is a compilable token range with its control-structure matches.」
type Unit struct {
	Name     string // 注曰「unique label prefix」
	Kind     string // 注曰「"main" | "func" | "init"」
	Toks     []Token
	If       map[int][2]int // 注曰「若 idx -> {否则 idx (-1), 毕若 idx}」
	IfEnd    map[int]int    // 注曰「否则 idx -> 毕若 idx」
	Loop     map[int]int    // 注曰「记/历 idx -> 毕记/毕历 idx」
	Try      map[int]int    // 注曰「宥 idx -> 毕宥 idx」
	Labels   map[string]int // 注曰「标名 -> idx」
	Locals   []string       // 注曰「ordered local names (func/init units)」
	LocalIdx map[string]int
	Includes map[int]string // 注曰「承 token idx -> init unit name（承典册则不载，无启元可演）」
}

func newUnit(name, kind string, toks []Token) *Unit {
	return &Unit{
		Name:     name,
		Kind:     kind,
		Toks:     toks,
		If:       map[int][2]int{},
		IfEnd:    map[int]int{},
		Loop:     map[int]int{},
		Try:      map[int]int{},
		Labels:   map[string]int{},
		LocalIdx: map[string]int{},
		Includes: map[int]string{},
	}
}

// 注曰「prescan matches control structures within a flat token list.」
func prescan(u *Unit) error {
	toks := u.Toks
	type frame struct {
		kind string // 注曰「术/若/记/历/宥」
		idx  int
		aux  int
	}
	var st []frame
	for idx, tk := range toks {
		if tk.Kind != "word" {
			continue
		}
		w := tk.Text
		switch w {
		case "标":
			// 注曰「标《名》：标记之声明；所标之地居声明之后（idx+2）」
			if idx+1 >= len(toks) || toks[idx+1].Kind != "mark" {
				return 劫f("第%d行：标后无标名（当书 标《名》）", tk.Line)
			}
			u.Labels[toks[idx+1].Text] = idx + 2
		case "若":
			st = append(st, frame{"若", idx, -1})
		case "否则":
			if len(st) == 0 || st[len(st)-1].kind != "若" || st[len(st)-1].aux != -1 {
				return 劫f("第%d行：否则无所否", tk.Line)
			}
			st[len(st)-1].aux = idx
		case "毕若":
			if len(st) == 0 || st[len(st)-1].kind != "若" {
				return 劫f("第%d行：毕若无所毕", tk.Line)
			}
			f := st[len(st)-1]
			st = st[:len(st)-1]
			u.If[f.idx] = [2]int{f.aux, idx}
			if f.aux >= 0 {
				u.IfEnd[f.aux] = idx
			}
		case "记":
			if idx+3 >= len(toks) {
				return 劫f("第%d行：记法不全：记《名》至 限", tk.Line)
			}
			if toks[idx+1].Kind != "mark" {
				return 劫f("第%d行：记后无名（当书 记《名》）", tk.Line)
			}
			if toks[idx+2].Text != "至" {
				return 劫f("第%d行：记法缺「至」", tk.Line)
			}
			st = append(st, frame{"记", idx, -1})
		case "历":
			if idx+3 >= len(toks) {
				return 劫f("第%d行：历法不全：历《名》于《列》", tk.Line)
			}
			if toks[idx+1].Kind != "mark" {
				return 劫f("第%d行：历后无名（当书 历《名》）", tk.Line)
			}
			if toks[idx+2].Text != "于" {
				return 劫f("第%d行：历法缺「于」", tk.Line)
			}
			st = append(st, frame{"历", idx, -1})
		case "毕记", "毕历":
			tag := "记"
			if w == "毕历" {
				tag = "历"
			}
			if len(st) == 0 || st[len(st)-1].kind != tag {
				return 劫f("第%d行：%s无所毕", tk.Line, w)
			}
			f := st[len(st)-1]
			st = st[:len(st)-1]
			u.Loop[f.idx] = idx
		case "宥":
			st = append(st, frame{"宥", idx, -1})
		case "毕宥":
			if len(st) == 0 || st[len(st)-1].kind != "宥" {
				return 劫f("第%d行：毕宥无所毕", tk.Line)
			}
			f := st[len(st)-1]
			st = st[:len(st)-1]
			u.Try[f.idx] = idx
		}
	}
	if len(st) > 0 {
		names := make([]string, 0, len(st))
		for _, f := range st {
			names = append(names, f.kind)
		}
		return 劫f("有敕未毕：%s", strings.Join(names, "、"))
	}
	return nil
}

// 注曰「collectAssigned returns the ordered unique names assigned within toks」
// 注曰「(via 生/《名》铭/受/双受/众受/记/历), for scope resolution.」
func collectAssigned(toks []Token) []string {
	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if n != "" && !reserved[n] && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for i := 0; i < len(toks); i++ {
		tk := toks[i]
		// 注曰「新铭法：《名》铭 —— 名号居前，铭字居后收之」
		if tk.Kind == "mark" && i+1 < len(toks) &&
			toks[i+1].Kind == "word" && toks[i+1].Text == "铭" {
			add(tk.Text)
			continue
		}
		if tk.Kind != "word" {
			continue
		}
		w := tk.Text
		switch w {
		case "生", "受":
			if i+1 < len(toks) && toks[i+1].Kind == "mark" {
				add(toks[i+1].Text)
			}
		case "双受":
			if i+2 < len(toks) {
				if toks[i+1].Kind == "mark" {
					add(toks[i+1].Text)
				}
				if toks[i+2].Kind == "mark" {
					add(toks[i+2].Text)
				}
			}
		case "众受":
			if i+1 < len(toks) {
				cnt, _ := parseNumber(toks[i+1].Text)
				if cnt.isInt {
					for k := 0; k < int(cnt.i64) && i+2+k < len(toks); k++ {
						if toks[i+2+k].Kind == "mark" {
							add(toks[i+2+k].Text)
						}
					}
				}
			}
		case "记", "历":
			if i+1 < len(toks) && toks[i+1].Kind == "mark" {
				add(toks[i+1].Text)
			}
		}
	}
	return out
}

// 注曰「splitTop separates top-level tokens from 术 definitions.」
// 注曰「术 may only appear at top level (outside any block and outside any 术).」
func splitTop(toks []Token) (top []Token, funcs []*Func, err error) {
	depth := 0
	block := map[string]int{"若": 1, "记": 1, "历": 1, "宥": 1}
	closeBlock := map[string]int{"毕若": -1, "毕记": -1, "毕历": -1, "毕宥": -1}
	i := 0
	n := len(toks)
	for i < n {
		tk := toks[i]
		if tk.Kind == "str" {
			top = append(top, tk)
			i++
			continue
		}
		w := tk.Text
		if w == "术" {
			if depth != 0 {
				return nil, nil, 劫f("第%d行：术不可立于他敕之内，须当界之坦处", tk.Line)
			}
			if i+1 >= n || toks[i+1].Kind != "mark" {
				return nil, nil, 劫f("第%d行：术后无术名（当书 术《名》）", tk.Line)
			}
			name := toks[i+1].Text
			if reserved[name] {
				return nil, nil, 劫f("第%d行：「%s」乃哲浩之谕之字，不得为术名", tk.Line, name)
			}
			// 注曰「find matching 毕术」
			j := i + 2
			for j < n && toks[j].Text != "毕术" {
				j++
			}
			if j >= n {
				return nil, nil, 劫f("第%d行：术「%s」无所毕", tk.Line, name)
			}
			body := toks[i+2 : j]
			f := &Func{Name: name, Body: body, Line: tk.Line}
			funcs = append(funcs, f)
			i = j + 1
			continue
		}
		if w == "毕术" {
			return nil, nil, 劫f("第%d行：毕术无所毕", tk.Line)
		}
		top = append(top, tk)
		if block[w] > 0 {
			depth++
		} else if closeBlock[w] < 0 {
			depth--
		}
		i++
	}
	if depth != 0 {
		return nil, nil, 劫f("有敕未毕")
	}
	return top, funcs, nil
}

// 注曰「Func is a 术 definition before unit construction.」
type Func struct {
	Name string
	Body []Token
	Line int
}

// 注曰「program」

// 注曰「Program holds everything needed to emit assembly.」
type Program struct {
	Main       *Unit
	Funcs      map[string]*Unit // 注曰「keyed by 术 name」
	FuncOrder  []string
	Inits      map[string]*Unit // 注曰「keyed by init unit name」
	InitOrder  []string
	InitByMod  map[string]string // 注曰「module name -> init unit name」
	Globals    []string
	GlobalIdx  map[string]int
	mainDir    string
	loading    map[string]bool
	initSeq    int
	modInits   map[string]string // 注曰「absolute file path -> init unit name（典册则为空）」
	includeErr error
}

func newProgram(mainDir string) *Program {
	return &Program{
		Funcs:     map[string]*Unit{},
		Inits:     map[string]*Unit{},
		InitByMod: map[string]string{},
		GlobalIdx: map[string]int{},
		mainDir:   mainDir,
		loading:   map[string]bool{},
		modInits:  map[string]string{},
	}
}

// 注曰「loadProgram reads and analyses the main file plus all 承 modules.」
func loadProgram(path string) (*Program, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	p := newProgram(filepath.Dir(abs))
	top, err := p.loadFile(abs)
	if err != nil {
		return nil, err
	}
	// 注曰「main unit」
	main := newUnit("main", "main", top)
	if err := prescan(main); err != nil {
		return nil, err
	}
	p.Main = main
	// 注曰「resolve 承 in main (recursively loads modules)」
	if err := p.resolveIncludes(main); err != nil {
		return nil, err
	}
	// 注曰「globals from main top-level + 外众」
	for _, nm := range collectAssigned(top) {
		p.addGlobal(nm)
	}
	p.addGlobal("外众")
	return p, nil
}

func (p *Program) addGlobal(nm string) {
	if _, ok := p.GlobalIdx[nm]; !ok {
		p.GlobalIdx[nm] = len(p.Globals)
		p.Globals = append(p.Globals, nm)
	}
}

// 注曰「loadFile tokenizes + splits one file, registers its 术, and processes 承」
// 注曰「sites (recursively) that appear in its top-level token stream. It returns」
// 注曰「the top-level tokens with 承 sites left in place (resolved at codegen).」
func (p *Program) loadFile(abs string) ([]Token, error) {
	if p.loading[abs] {
		return nil, 劫f("经籍循环相承：「%s」", filepath.Base(abs))
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, 劫f("界牍无载：%s 不可得", abs)
	}
	toks, err := tokenize(string(data))
	if err != nil {
		return nil, err
	}
	if len(toks) == 0 || toks[0].Text != "哲浩御世" {
		if strings.HasSuffix(abs, ".哲") {
			return nil, 劫f("典册非界：「%s」以「.哲」名之，唯堪承纳，不可为界牍之主", filepath.Base(abs))
		}
		return nil, 劫f("界无本源：凡哲浩之谕之界，必以「哲浩御世」开篇")
	}
	top, funcs, err := splitTop(toks)
	if err != nil {
		return nil, err
	}
	// 注曰「register 术」
	for _, f := range funcs {
		if reserved[f.Name] {
			return nil, 劫f("第%d行：「%s」乃哲浩之谕之字，不得为术名", f.Line, f.Name)
		}
		if _, exists := p.Funcs[f.Name]; exists {
			return nil, 劫f("二术同名：「%s」", f.Name)
		}
		u := newUnit("fn_"+f.Name, "func", f.Body)
		if err := prescan(u); err != nil {
			return nil, err
		}
		for _, nm := range collectAssigned(f.Body) {
			u.Locals = append(u.Locals, nm)
			u.LocalIdx[nm] = len(u.Locals) - 1
		}
		p.Funcs[f.Name] = u
		p.FuncOrder = append(p.FuncOrder, f.Name)
	}
	return top, nil
}

// 注曰「resolveIncludes walks top-level tokens of a unit and, for each 承, loads the」
// 注曰「named file and records its init unit (keyed by token index). Headers carry」
// 注曰「no init unit, so a header 承 records nothing — its 术 are already registered.」
func (p *Program) resolveIncludes(unit *Unit) error {
	for i := 0; i < len(unit.Toks); i++ {
		tk := unit.Toks[i]
		if tk.Kind != "word" || tk.Text != "承" {
			continue
		}
		if i+1 >= len(unit.Toks) || unit.Toks[i+1].Kind != "mark" {
			return 劫f("第%d行：承后无籍名（当书 承《名》）", tk.Line)
		}
		mod := unit.Toks[i+1].Text
		initName, err := p.loadModule(mod)
		if err != nil {
			return err
		}
		if initName != "" {
			unit.Includes[i] = initName
		}
		i++ // 注曰「skip the module name token」
	}
	return nil
}

// 注曰「loadModule locates and loads a 承'd file by name. 承《名》 first seeks」
// 注曰「《名》.哲（典册）, then 《名》.浩（经卷）. It returns the init unit name to」
// 注曰「be called at the 承 site — empty for headers, which declare 术 but perform」
// 注曰「no deeds.」
func (p *Program) loadModule(name string) (string, error) {
	if initName, ok := p.InitByMod[name]; ok {
		return initName, nil
	}
	var path string
	for _, d := range []string{p.mainDir, "."} {
		for _, ext := range []string{".哲", ".浩"} {
			cand := filepath.Join(d, name+ext)
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				path = cand
				break
			}
		}
		if path != "" {
			break
		}
	}
	if path == "" {
		return "", 劫f("典籍无载：「%s.哲」与「%s.浩」皆不可得", name, name)
	}
	abs, _ := filepath.Abs(path)
	if p.loading[abs] {
		return "", 劫f("经籍循环相承：「%s」", name)
	}
	if initName, ok := p.modInits[abs]; ok {
		p.InitByMod[name] = initName
		return initName, nil
	}
	p.loading[abs] = true
	defer delete(p.loading, abs)
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", 劫f("界牍无载：%s 不可得", abs)
	}
	toks, err := tokenize(string(data))
	if err != nil {
		return "", err
	}
	header := strings.HasSuffix(path, ".哲")
	if !header && (len(toks) == 0 || toks[0].Text != "哲浩御世") {
		return "", 劫f("界无本源：凡哲浩之谕之界，必以「哲浩御世」开篇")
	}
	top, funcs, err := splitTop(toks)
	if err != nil {
		return "", err
	}
	for _, f := range funcs {
		if _, exists := p.Funcs[f.Name]; exists {
			return "", 劫f("二术同名：「%s」", f.Name)
		}
		u := newUnit("fn_"+f.Name, "func", f.Body)
		if err := prescan(u); err != nil {
			return "", err
		}
		for _, nm := range collectAssigned(f.Body) {
			u.Locals = append(u.Locals, nm)
			u.LocalIdx[nm] = len(u.Locals) - 1
		}
		p.Funcs[f.Name] = u
		p.FuncOrder = append(p.FuncOrder, f.Name)
	}
	if header {
		// 注曰「典册非界：唯立术式，兼承他典册，不演敕令，故无启元可演。」
		if err := p.resolveHeaderIncludes(top); err != nil {
			return "", err
		}
		p.InitByMod[name] = ""
		p.modInits[abs] = ""
		return "", nil
	}
	p.initSeq++
	initName := fmt.Sprintf("init_%d", p.initSeq)
	init := newUnit(initName, "init", top)
	if err := prescan(init); err != nil {
		return "", err
	}
	for _, nm := range collectAssigned(top) {
		init.Locals = append(init.Locals, nm)
		init.LocalIdx[nm] = len(init.Locals) - 1
	}
	p.Inits[initName] = init
	p.InitOrder = append(p.InitOrder, initName)
	p.modInits[abs] = initName
	p.InitByMod[name] = initName
	// 注曰「resolve nested 承 inside this module's top-level」
	if err := p.resolveIncludes(init); err != nil {
		return "", err
	}
	return initName, nil
}

// 注曰「resolveHeaderIncludes validates a header's top-level tokens. A header is」
// 注曰「not a world: it may only declare 术 (already extracted by splitTop) and」
// 注曰「承 other headers; any other deed is refused. 承'd names must resolve to」
// 注曰「headers — a world's 顶敕 may only be performed in a world.」
func (p *Program) resolveHeaderIncludes(top []Token) error {
	for i := 0; i < len(top); i++ {
		tk := top[i]
		if tk.Kind == "word" && tk.Text == "哲浩御世" && i == 0 {
			continue // 注曰「典册不须御世，然御之亦宥」
		}
		if tk.Kind == "word" && tk.Text == "承" {
			if i+1 >= len(top) || top[i+1].Kind != "mark" {
				return 劫f("第%d行：承后无籍名（当书 承《名》）", tk.Line)
			}
			sub := top[i+1].Text
			initName, err := p.loadModule(sub)
			if err != nil {
				return err
			}
			if initName != "" {
				return 劫f("第%d行：典册唯承典册，「%s」乃经卷（.浩），请承于界中", tk.Line, sub)
			}
			i++
			continue
		}
		return 劫f("第%d行：典册非界，不演敕令——「%s」非典册所容（唯术与承）", tk.Line, tk.Text)
	}
	return nil
}

// 注曰「sortedGlobalNames is unused but kept for determinism utilities.」
func sortedGlobalNames(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
