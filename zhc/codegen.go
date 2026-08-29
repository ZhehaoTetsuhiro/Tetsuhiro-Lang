package main

import (
	"fmt"
	"math"
	"strings"
)

// 注曰「emitter 累积所生之 AMD64 汇编（GNU as，Intel 句法）。」
type emitter struct {
	prog    *Program
	text    strings.Builder
	data    strings.Builder
	slits   map[string]string // 注曰「文之内容 → 数据标签」
	fconst  map[uint64]string // 注曰「浮之位样 → 数据标签」
	prefix  map[*Unit]string  // 注曰「单元 → 标签前缀」
	funcLbl map[string]string // 注曰「术名 → 标签（避汉字符号）」
	uid     int
	lseq    int
}

func newEmitter(p *Program) *emitter {
	e := &emitter{
		prog:   p,
		slits:  map[string]string{},
		fconst: map[uint64]string{},
		prefix: map[*Unit]string{},
	}
	// 注曰「数据节先立，此后凡入 e.data 者皆属 .data」
	e.data.WriteString("    .data\n")
	return e
}

func (e *emitter) tmpLbl() string {
	e.lseq++
	return fmt.Sprintf("tmp_%d", e.lseq)
}

func (e *emitter) emit(format string, a ...interface{}) {
	e.text.WriteString(fmt.Sprintf(format, a...))
	e.text.WriteString("\n")
}

func (e *emitter) unitPrefix(u *Unit) string {
	if p, ok := e.prefix[u]; ok {
		return p
	}
	e.uid++
	p := fmt.Sprintf("u%d", e.uid)
	e.prefix[u] = p
	return p
}

func (e *emitter) lbl(u *Unit, idx int) string {
	return fmt.Sprintf("%s_%d", e.unitPrefix(u), idx)
}

// 注曰「li 发射立数之令（合于三十二位者用 mov，否则 movabs）。」
func (e *emitter) li(reg string, v int64) {
	if v >= math.MinInt32 && v <= math.MaxInt32 {
		e.emit("    mov  %s, %d", reg, v)
	} else {
		e.emit("    movabs %s, %d", reg, v)
	}
}

func (e *emitter) push(reg string) {
	e.emit("    sub  r15, 8")
	e.emit("    mov  [r15], %s", reg)
}

func (e *emitter) pop(reg string) {
	e.emit("    mov  %s, [r15]", reg)
	e.emit("    add  r15, 8")
}

// 注曰「化生 ----------------------------------------------------------------」

func (e *emitter) storeVar(u *Unit, name string) {
	if u.Kind != "main" {
		if idx, ok := u.LocalIdx[name]; ok {
			e.emit("    mov  [rbp-%d], rax", 8*(idx+1))
			return
		}
	}
	gi, ok := e.prog.GlobalIdx[name]
	if !ok {
		panic("storeVar: unknown global " + name)
	}
	e.emit("    lea  r10, [rip+gvar_%d]", gi)
	e.emit("    mov  [r10], rax")
}

func (e *emitter) loadVar(u *Unit, name string) {
	if u.Kind != "main" {
		if idx, ok := u.LocalIdx[name]; ok {
			e.emit("    mov  rax, [rbp-%d]", 8*(idx+1))
			return
		}
	}
	gi, ok := e.prog.GlobalIdx[name]
	if !ok {
		panic(fmt.Sprintf("未名之化：%s", name))
	}
	e.emit("    lea  r10, [rip+gvar_%d]", gi)
	e.emit("    mov  rax, [r10]")
}

// 注曰「实数 ----------------------------------------------------------------」

func (e *emitter) strLit(content string) string {
	if lbl, ok := e.slits[content]; ok {
		return lbl
	}
	lbl := fmt.Sprintf("slit_%d", len(e.slits))
	e.slits[content] = lbl
	e.emitStrAt(lbl, content)
	return lbl
}

// 注曰「emitStrAt 立十六字节头之文实：[长, 0] + 内联字节。」
func (e *emitter) emitStrAt(lbl, content string) {
	b := []byte(content)
	e.data.WriteString("    .align 8\n")
	e.data.WriteString(lbl + ":\n")
	e.data.WriteString(fmt.Sprintf("    .quad %d\n", len(b)))
	e.data.WriteString("    .quad 0\n")
	for i := 0; i < len(b); i += 8 {
		end := i + 8
		if end > len(b) {
			end = len(b)
		}
		parts := make([]string, 0, 8)
		for _, x := range b[i:end] {
			parts = append(parts, fmt.Sprintf("%d", x))
		}
		e.data.WriteString("    .byte " + strings.Join(parts, ", ") + "\n")
	}
}

func (e *emitter) floatConst(f float64) string {
	bits := math.Float64bits(f)
	if lbl, ok := e.fconst[bits]; ok {
		return lbl
	}
	lbl := fmt.Sprintf("fconst_%d", len(e.fconst))
	e.fconst[bits] = lbl
	e.data.WriteString("    .align 8\n")
	e.data.WriteString(lbl + ":\n")
	e.data.WriteString(fmt.Sprintf("    .quad 0x%x\n", bits))
	return lbl
}

// 注曰「定数与全域 -----------------------------------------------------------」

func (e *emitter) emitFixedData() {
	e.data.WriteString("g_errmsg:\n    .quad 0\n")
	e.emitStrAt("s_阳", "阳")
	e.emitStrAt("s_阴", "阴")
	e.emitStrAt("s_虚", "虚")
	e.emitStrAt("s_数", "数")
	e.emitStrAt("s_文", "文")
	e.emitStrAt("s_列", "列")
	e.emitStrAt("s_阴阳", "阴阳")
	e.emitStrAt("s_典", "典")
	e.emitStrAt("err_add", "不可相加")
	e.emitStrAt("err_divzero", "除以虚")
	e.emitStrAt("err_len", "无几何")
	e.emitStrAt("err_get", "无元可取")
	e.emitStrAt("err_oob", "越界")
	e.emitStrAt("err_set", "不可铭元")
	e.emitStrAt("err_append", "不可附")
	e.emitStrAt("err_empty", "列空")
	e.emitStrAt("err_rand", "天机限")
	e.emitStrAt("err_num", "非数之化")
	e.emitStrAt("err_pow", "幂未行")
	e.emitStrAt("err_neg", "反之非数")
	e.emitStrAt("err_sqrt", "负数无根")
	e.emitStrAt("err_rem", "余未行")
	e.emitStrAt("err_list", "聚数非正")
	e.emitStrAt("err_cmp", "所较非数")
	e.emitStrAt("err_dict", "非典之化")
	e.emitStrAt("err_path", "径非文")
	e.emitStrAt("err_read", "简牍难读")
	e.emitStrAt("err_write", "简牍难书")
	e.emitStrAt("err_split", "剖隔空空")
	e.emitStrAt("err_splitn", "剖者唯文")
	// 注曰「化数识汉数之表：码、类（〇数字 一小单位 二大单位 三点 四廿 五卅）、值」
	stnTab := []struct {
		code int
		kind int
		val  int
	}{
		{0xE38087, 0, 0},         // 〇
		{0xE99BB6, 0, 0},         // 零
		{0xE4B880, 0, 1},         // 一
		{0xE4BA8C, 0, 2},         // 二
		{0xE4B8A4, 0, 2},         // 两
		{0xE4B889, 0, 3},         // 三
		{0xE59B9B, 0, 4},         // 四
		{0xE4BA94, 0, 5},         // 五
		{0xE585AD, 0, 6},         // 六
		{0xE4B883, 0, 7},         // 七
		{0xE585AB, 0, 8},         // 八
		{0xE4B99D, 0, 9},         // 九
		{0xE58D81, 1, 10},        // 十
		{0xE799BE, 1, 100},       // 百
		{0xE58D83, 1, 1000},      // 千
		{0xE4B887, 2, 10000},     // 万
		{0xE4BABF, 2, 100000000}, // 亿
		{0xE782B9, 3, 0},         // 点
		{0xE5BBBF, 4, 10},        // 廿（二而十）
		{0xE58D85, 5, 10},        // 卅（三而十）
	}
	e.data.WriteString("    .align 8\nstn_tab:\n")
	for _, r := range stnTab {
		e.data.WriteString(fmt.Sprintf("    .long 0x%X\n    .long %d\n", r.code, (r.val<<3)|r.kind))
	}
	e.data.WriteString("    .long 0\n    .long 0\n")
	// 注曰「浮印之常数」
	e.data.WriteString("    .align 8\nf_ten:\n    .quad 0x4024000000000000\n")
	e.data.WriteString("    .align 8\nf_half:\n    .quad 0x3FE0000000000000\n")
	e.data.WriteString("    .align 8\nf_1e15:\n    .quad 0x430C6BF526340000\n")
	// 注曰「未捕之劫，径书于误途（stderr）」
	e.data.WriteString("s_uncaught:\n")
	ub := []byte("哲浩垂示：")
	for i := 0; i < len(ub); i += 8 {
		end := i + 8
		if end > len(ub) {
			end = len(ub)
		}
		parts := make([]string, 0, 8)
		for _, x := range ub[i:end] {
			parts = append(parts, fmt.Sprintf("%d", x))
		}
		e.data.WriteString("    .byte " + strings.Join(parts, ", ") + "\n")
	}
	e.data.WriteString("s_nl:\n    .byte 10\n")
}

// 注曰「令牌之译 ------------------------------------------------------------」

// 注曰「genUnit 译一单元之全令牌，每令牌之前立一标签。」
func (e *emitter) genUnit(u *Unit) {
	toks := u.Toks
	i := 0
	for i < len(toks) {
		e.emit("%s:", e.lbl(u, i))
		i = e.genToken(u, i)
	}
	// 注曰「单元末标签：供 否则/毕记/毕历 等跳至「敕块之后」。」
	e.emit("%s:", e.lbl(u, len(toks)))
}

// 注曰「genToken 译第 i 令牌，返下一令牌之位。」
func (e *emitter) genToken(u *Unit, i int) int {
	tok := u.Toks[i]
	if tok.Kind == "str" {
		lbl := e.strLit(tok.Text)
		e.emit("    lea  rax, [rip+%s]", lbl)
		e.emit("    shl  rax, 3")
		e.emit("    or   rax, 4")
		e.push("rax")
		return i + 1
	}
	if tok.Kind == "mark" {
		// 注曰「名号：术之调用，或化生之读；若铭字居后，则为《名》铭之新法」
		w := tok.Text
		if i+1 < len(u.Toks) && u.Toks[i+1].Kind == "word" && u.Toks[i+1].Text == "铭" {
			if _, isFn := e.prog.Funcs[w]; isFn {
				panic(fmt.Sprintf("第%d行：「%s」乃术名，不可铭之", tok.Line, w))
			}
			e.pop("rax")
			e.storeVar(u, w)
			return i + 2
		}
		if _, ok := e.prog.Funcs[w]; ok {
			e.emit("    call %s", e.funcLbl[w])
			return i + 1
		}
		e.loadVar(u, w)
		e.push("rax")
		return i + 1
	}
	w := tok.Text
	switch w {
	case "阳":
		e.li("rax", 3)
		e.push("rax")
		return i + 1
	case "阴":
		e.li("rax", 2)
		e.push("rax")
		return i + 1
	case "虚":
		e.li("rax", 1)
		e.push("rax")
		return i + 1
	case "标":
		// 注曰「标之定义：genUnit 已于此位排出标签，令牌与其名皆不生码。」
		return i + 2
	}
	// 注曰「数实？」
	if nk, ok := parseNumber(w); ok {
		if nk.isInt {
			e.li("rax", nk.i64<<3)
			e.push("rax")
		} else {
			lbl := e.floatConst(nk.f64)
			e.emit("    lea  rax, [rip+%s]", lbl)
			e.emit("    shl  rax, 3")
			e.emit("    or   rax, 7")
			e.push("rax")
		}
		return i + 1
	}
	// 注曰「化生」
	switch w {
	case "生":
		name := u.Toks[i+1].Text
		e.li("rax", 1)
		e.storeVar(u, name)
		return i + 2
	case "铭":
		// 注曰「新铭法：《名》铭 —— 铭字必承名号之后；孤铭无所铭，乃旧法之误」
		panic(fmt.Sprintf("第%d行：铭无所铭——新法：实后书《名》铭，如 三《甲》铭", tok.Line))
	case "取":
		name := u.Toks[i+1].Text
		e.loadVar(u, name)
		e.push("rax")
		return i + 2
	case "受":
		name := u.Toks[i+1].Text
		e.pop("rax")
		e.storeVar(u, name)
		return i + 2
	case "双受":
		a, b := u.Toks[i+1].Text, u.Toks[i+2].Text
		e.pop("rax")
		e.storeVar(u, b)
		e.pop("rax")
		e.storeVar(u, a)
		return i + 3
	case "众受":
		nk, _ := parseNumber(u.Toks[i+1].Text)
		n := int(nk.i64)
		names := make([]string, n)
		for k := 0; k < n; k++ {
			names[k] = u.Toks[i+2+k].Text
		}
		for k := n - 1; k >= 0; k-- {
			e.pop("rax")
			e.storeVar(u, names[k])
		}
		return i + 2 + n
	}
	// 注曰「算术」
	switch w {
	case "加":
		e.emit("    call rt_add")
		return i + 1
	case "减":
		e.emit("    call rt_sub")
		return i + 1
	case "乘":
		e.emit("    call rt_mul")
		return i + 1
	case "除":
		e.emit("    call rt_div")
		return i + 1
	case "余":
		e.emit("    call rt_rem")
		return i + 1
	case "幂":
		e.emit("    call rt_pow")
		return i + 1
	case "反":
		e.emit("    call rt_neg")
		return i + 1
	case "绝":
		e.emit("    call rt_abs")
		return i + 1
	case "根":
		e.emit("    call rt_sqrt")
		return i + 1
	case "整":
		e.emit("    call rt_floor")
		return i + 1
	}
	// 注曰「较」
	if op, ok := map[string]int{"等": 0, "异": 1, "逾": 2, "逊": 3, "逾等": 4, "逊等": 5}[w]; ok {
		e.emit("    mov  r11d, %d", op)
		e.emit("    call rt_cmp")
		return i + 1
	}
	// 注曰「逻辑」
	switch w {
	case "且":
		e.emit("    call rt_and")
		return i + 1
	case "或":
		e.emit("    call rt_or")
		return i + 1
	case "非":
		e.emit("    call rt_not")
		return i + 1
	}
	// 注曰「栈礼」
	switch w {
	case "复":
		e.emit("    mov  rax, [r15]")
		e.emit("    sub  r15, 8")
		e.emit("    mov  [r15], rax")
		return i + 1
	case "弃":
		e.emit("    add  r15, 8")
		return i + 1
	case "换":
		e.emit("    mov  rax, [r15]")
		e.emit("    mov  r10, [r15+8]")
		e.emit("    mov  [r15], r10")
		e.emit("    mov  [r15+8], rax")
		return i + 1
	case "凌":
		e.emit("    mov  rax, [r15+8]")
		e.emit("    sub  r15, 8")
		e.emit("    mov  [r15], rax")
		return i + 1
	case "转":
		e.emit("    mov  rax, [r15]")
		e.emit("    mov  r10, [r15+8]")
		e.emit("    mov  r11, [r15+16]")
		e.emit("    mov  [r15+16], r10")
		e.emit("    mov  [r15+8], rax")
		e.emit("    mov  [r15], r11")
		return i + 1
	}
	// 注曰「类」
	switch w {
	case "化数":
		e.emit("    call rt_tonum")
		return i + 1
	case "化文":
		// 注曰「化文：出栈一实，化为文（rt_tostr 取参于 rdi）」
		e.pop("rdi")
		e.emit("    call rt_tostr")
		e.push("rax")
		return i + 1
	case "化阴阳":
		e.emit("    call rt_tobool")
		return i + 1
	case "何类":
		e.emit("    call rt_type")
		return i + 1
	}
	// 注曰「宣聆」
	switch w {
	case "宣":
		e.emit("    call rt_print")
		return i + 1
	case "语":
		e.emit("    call rt_printn")
		return i + 1
	case "聆":
		e.emit("    call rt_read")
		return i + 1
	case "读简":
		// 注曰「出栈一径（文），读简牍全文为文」
		e.pop("rdi")
		e.emit("    call rt_readfile")
		e.push("rax")
		return i + 1
	case "书简":
		// 注曰「栈顶为径，其下为实——实化文而书于简牍，覆其旧」
		e.emit("    call rt_writefile")
		return i + 1
	}
	// 注曰「列典」
	switch w {
	case "聚":
		e.emit("    call rt_makelist")
		return i + 1
	case "几何":
		e.emit("    call rt_len")
		return i + 1
	case "取元":
		e.emit("    call rt_get")
		return i + 1
	case "铭元":
		e.emit("    call rt_set")
		return i + 1
	case "列附":
		e.emit("    call rt_append")
		return i + 1
	case "列抽":
		e.emit("    call rt_popback")
		return i + 1
	case "剖":
		// 注曰「「文」 「隔」 剖——以隔剖文为列，空元不遗」
		e.emit("    call rt_split")
		return i + 1
	case "典":
		// 注曰「典字自生一空典」
		e.emit("    call rt_mkdict")
		e.push("rax")
		return i + 1
	case "铭寓":
		// 注曰「实 键 《典》 铭寓——铭实寓于键，有则代之」
		e.emit("    call rt_dictset")
		return i + 1
	case "取寓":
		// 注曰「《典》《键》 取寓——取其实，无键则虚」
		e.emit("    call rt_dictget")
		return i + 1
	case "有寓":
		// 注曰「《典》《键》 有寓——问其有无，得阴阳」
		e.emit("    call rt_dicthas")
		return i + 1
	case "除寓":
		// 注曰「《典》《键》 除寓——除其寓，无键则默然」
		e.emit("    call rt_dictdel")
		return i + 1
	case "缀":
		e.emit("    call rt_concat")
		return i + 1
	case "天机":
		e.emit("    call rt_rand")
		return i + 1
	}
	// 注曰「流转」
	switch w {
	case "若":
		info := u.If[i]
		var target string
		if info[0] >= 0 {
			target = e.lbl(u, info[0]+1)
		} else {
			target = e.lbl(u, info[1]+1)
		}
		e.pop("rax")
		e.emit("    call rt_truthy")
		e.emit("    test rax, rax")
		e.emit("    jz   %s", target)
		return i + 1
	case "否则":
		e.emit("    jmp  %s", e.lbl(u, u.IfEnd[i]+1))
		return i + 1
	case "毕若":
		return i + 1
	case "毕记", "毕历":
		e.emitLoopEnd(u, i)
		return i + 1
	case "毕宥":
		e.emit("    add  r12, 32")
		return i + 1
	case "记":
		return e.genFor(u, i)
	case "历":
		return e.genForEach(u, i)
	case "谒":
		name := u.Toks[i+1].Text
		e.emit("    jmp  %s", e.lbl(u, u.Labels[name]))
		return i + 2
	case "虚谒":
		name := u.Toks[i+1].Text
		e.pop("rax")
		e.emit("    call rt_truthy")
		e.emit("    test rax, rax")
		e.emit("    jz   %s", e.lbl(u, u.Labels[name]))
		return i + 2
	case "阳谒":
		name := u.Toks[i+1].Text
		e.pop("rax")
		e.emit("    call rt_truthy")
		e.emit("    test rax, rax")
		e.emit("    jnz  %s", e.lbl(u, u.Labels[name]))
		return i + 2
	case "宥":
		e.emit("    lea  rax, [rip+%s]", e.lbl(u, u.Try[i]+1))
		e.emit("    sub  r12, 32")
		e.emit("    mov  [r12], r15")
		e.emit("    mov  [r12+8], rsp")
		e.emit("    mov  [r12+16], rbp")
		e.emit("    mov  [r12+24], rax")
		return i + 1
	}
	// 注曰「界 / 术」
	switch w {
	case "承":
		// 注曰「经卷之承，演其启元；典册唯纳其术，无启元可演」
		if init := u.Includes[i]; init != "" {
			e.emit("    call %s", init)
		}
		return i + 2
	case "寂":
		e.emit("    xor  edi, edi")
		e.emit("    call rt_exit")
		return i + 1
	case "归源", "归":
		if u.Kind == "main" {
			e.emit("    xor  edi, edi")
			e.emit("    call rt_exit")
		} else {
			e.emitReturn(u)
		}
		return i + 1
	case "毕术":
		e.emitReturn(u)
		return i + 1
	case "哲浩御世", "至", "于":
		return i + 1
	case "术":
		panic("内部错误：术 出现于令牌流")
	}
	panic(fmt.Sprintf("第%d行：未识之令「%s」", tok.Line, w))
}

// 注曰「genFor 译 记《名》至 限 … 毕记。」
func (e *emitter) genFor(u *Unit, i int) int {
	name := u.Toks[i+1].Text
	end := u.Loop[i]
	body := i + 4
	// 注曰「限：数或化生之名」
	lim := u.Toks[i+3]
	if nk, ok := parseNumber(lim.Text); ok {
		if !nk.isInt {
			panic(fmt.Sprintf("第%d行：至后之限须整数", lim.Line))
		}
		e.li("rax", nk.i64)
	} else {
		e.loadVar(u, lim.Text)
		e.emit("    sar  rax, 3")
	}
	e.emit("    sub  r13, 8")
	e.emit("    mov  [r13], rax")
	// 注曰「循环之元始于零」
	e.emit("    xor  eax, eax")
	e.storeVar(u, name)
	// 注曰「较名与限」
	e.loadVar(u, name)
	e.emit("    mov  r10, [r13]")
	e.emit("    sar  rax, 3")
	e.emit("    cmp  rax, r10")
	e.emit("    jl   %s", e.lbl(u, body))
	e.emit("    add  r13, 8")
	e.emit("    jmp  %s", e.lbl(u, end+1))
	return body
}

// 注曰「genForEach 译 历《名》于《列》… 毕历。」
func (e *emitter) genForEach(u *Unit, i int) int {
	name := u.Toks[i+1].Text
	src := u.Toks[i+3].Text
	end := u.Loop[i]
	// 注曰「载列；典亦堪历：先取其键之快照（一列），后循列之旧礼」
	e.loadVar(u, src)
	e.emit("    mov  r10, rax")
	e.emit("    and  r10d, 7")
	e.emit("    cmp  r10d, 6")
	skip := e.tmpLbl()
	e.emit("    jne  %s", skip)
	e.emit("    mov  rdi, rax")
	e.emit("    call rt_dictkeys")
	e.emit("%s:", skip)
	e.emit("    mov  r10, rax")
	e.emit("    shr  r10, 3")
	e.emit("    sub  r13, 16")
	e.emit("    mov  [r13], r10")
	e.emit("    mov  qword ptr [r13+8], 0")
	// 注曰「空列则跳过」
	e.emit("    mov  r11, [r10]")
	e.emit("    test r11, r11")
	first := e.tmpLbl()
	e.emit("    jnz  %s", first)
	e.emit("    add  r13, 16")
	e.emit("    jmp  %s", e.lbl(u, end+1))
	e.emit("%s:", first)
	// 注曰「载首元」
	e.emitForEachLoad(u, name)
	return i + 4
}

func (e *emitter) emitForEachLoad(u *Unit, name string) {
	e.emit("    mov  r10, [r13]")
	e.emit("    mov  rax, [r13+8]")
	e.emit("    mov  r11, [r10+16]")
	e.emit("    mov  rax, [r11+rax*8]")
	e.storeVar(u, name)
}

// 注曰「emitLoopEnd 应 毕记/毕历 之译。」
func (e *emitter) emitLoopEnd(u *Unit, i int) {
	// 注曰「反索匹配之记/历」
	var start int
	var isFor bool
	for s, en := range u.Loop {
		if en == i {
			start = s
			isFor = u.Toks[s].Text == "记"
			break
		}
	}
	if isFor {
		name := u.Toks[start+1].Text
		e.loadVar(u, name)
		e.emit("    add  rax, 8")
		e.storeVar(u, name)
		e.emit("    mov  r10, [r13]")
		e.emit("    sar  rax, 3")
		e.emit("    cmp  rax, r10")
		e.emit("    jl   %s", e.lbl(u, start+4))
		e.emit("    add  r13, 8")
	} else {
		name := u.Toks[start+1].Text
		e.emit("    mov  rax, [r13+8]")
		e.emit("    inc  rax")
		e.emit("    mov  [r13+8], rax")
		e.emit("    mov  r10, [r13]")
		e.emit("    mov  r11, [r10]")
		e.emit("    cmp  rax, r11")
		rep := e.tmpLbl()
		e.emit("    jl   %s", rep)
		e.emit("    add  r13, 16")
		e.emit("    jmp  %s", e.lbl(u, i+1))
		e.emit("%s:", rep)
		e.emitForEachLoad(u, name)
		e.emit("    jmp  %s", e.lbl(u, start+4))
	}
}

func (e *emitter) emitReturn(u *Unit) {
	e.emit("    mov  rsp, rbp")
	e.emit("    pop  rbp")
	e.emit("    ret")
}

// 注曰「emitFuncPrologue 立函数式单元之帧。」
func (e *emitter) emitFuncPrologue(u *Unit) {
	e.emit("    push rbp")
	e.emit("    mov  rbp, rsp")
	if n := len(u.Locals); n > 0 {
		bytes := 8 * n
		if bytes%16 != 0 {
			bytes += 8
		}
		e.emit("    sub  rsp, %d", bytes)
	}
}

// 注曰「compileProgram 生全份汇编之源。」
func compileProgram(p *Program) string {
	e := newEmitter(p)
	// 注曰「术之标签以序号为名（避汉字符号之疑）」
	e.funcLbl = map[string]string{}
	for idx, name := range p.FuncOrder {
		e.funcLbl[name] = fmt.Sprintf("fn_%d", idx)
	}

	e.emit(".intel_syntax noprefix")
	e.emit(".text")
	e.emit(".globl _start")
	e.emit("_start:")
	e.emit("    mov  rbp, rsp")
	// 注曰「值栈（一兆）」
	e.emit("    mov  esi, %d", 0x100000)
	e.emit("    call rt_mmap0")
	e.emit("    lea  r15, [rax+%d]", 0x100000)
	// 注曰「堆（十六兆）」
	e.emit("    mov  esi, %d", 0x1000000)
	e.emit("    call rt_mmap0")
	e.emit("    mov  r14, rax")
	// 注曰「循环栈（六万四千）」
	e.emit("    mov  esi, %d", 0x10000)
	e.emit("    call rt_mmap0")
	e.emit("    lea  r13, [rax+%d]", 0x10000)
	// 注曰「宥栈（六万四千）；g_trybase 记其空栈之顶（r12 初值），以辨无主之劫」
	e.emit("    mov  esi, %d", 0x10000)
	e.emit("    call rt_mmap0")
	e.emit("    lea  r12, [rax+%d]", 0x10000)
	e.emit("    lea  r10, [rip+g_trybase]")
	e.emit("    mov  [r10], r12")
	// 注曰「外众 = 空列」
	e.emit("    xor  eax, eax")
	e.push("rax")
	e.emit("    call rt_makelist")
	e.pop("rax")
	gi := p.GlobalIdx["外众"]
	e.emit("    lea  r10, [rip+gvar_%d]", gi)
	e.emit("    mov  [r10], rax")
	e.emit("    call main_entry")
	e.emit("    xor  edi, edi")
	e.emit("    call rt_exit")

	// 注曰「运行时诸助理」
	e.text.WriteString(runtimeText)

	// 注曰「主界」
	e.emit("main_entry:")
	e.emit("    push rbp")
	e.emit("    mov  rbp, rsp")
	e.genUnit(p.Main)
	// 注曰「主界若无 归源，行至末尾亦当安然归寂。」
	e.emit("    xor  edi, edi")
	e.emit("    call rt_exit")

	// 注曰「诸经之启元」
	for _, name := range p.InitOrder {
		u := p.Inits[name]
		e.emit("%s:", name)
		e.emitFuncPrologue(u)
		e.genUnit(u)
		e.emitReturn(u)
	}
	// 注曰「诸术」
	for _, name := range p.FuncOrder {
		u := p.Funcs[name]
		e.emit("%s:", e.funcLbl[name])
		e.emitFuncPrologue(u)
		e.genUnit(u)
		e.emitReturn(u)
	}

	// 注曰「定数、文实、浮实、全域之槽（e.data 已以 .data 开节）」
	e.emitFixedData()
	e.data.WriteString("    .align 8\n")
	for i := range p.Globals {
		e.data.WriteString(fmt.Sprintf("gvar_%d:\n    .quad 0\n", i))
	}
	e.data.WriteString("g_seed:\n    .quad 305419896\n")
	// 注曰「缓冲诸器（本归于零）」
	e.data.WriteString("\n    .bss\n")
	e.data.WriteString("    .align 8\n")
	e.data.WriteString("g_outbuf:\n    .space 4096\n")
	e.data.WriteString("g_outlen:\n    .space 8\n")
	e.data.WriteString("g_readbuf:\n    .space 4096\n")
	e.data.WriteString("g_ch:\n    .space 8\n")
	e.data.WriteString("g_itobuf:\n    .space 64\n")
	e.data.WriteString("g_ftobuf:\n    .space 64\n")
	e.data.WriteString("g_trybase:\n    .space 8\n")
	e.data.WriteString("\n    .section .note.GNU-stack,\"\",@progbits\n")

	return e.text.String() + e.data.String()
}
