package main

// 注曰「哲浩之谕运行时——AMD64（x86-64）版。」
//
// 注曰「值之表示（64 位带标记）：」
// 注曰「低三位：0=数 (n<<3)，1=虚，2=阴，3=阳，4=文 (指针)，5=列 (指针)，7=浮 (指针)。」
// 注曰「文：16 字节头 [长, 0] + 内联字节；列：32 字节头 [长, 容, 数据指针]。」
//
// 注曰「寄存器之约：」
// 注曰「r15 = 值栈指针（向下长，槽 8 字节）；r14 = 堆指针（向上长）；」
// 注曰「r13 = 循环栈指针；r12 = 宥栈指针；rsp/rbp = 机栈与帧；」
// 注曰「rax rcx rdx rsi rdi r8-r11 = 易逝（诸助理可毁）；rbx rbp r12-r15 = 恒存。」
//
// 注曰「助理之约：」
// 注曰「助理自值栈取 operands，算毕推回；rdi/rsi 常为入参，rax 为果；」
// 注曰「rt_alloc 只毁 rax 与 r11；rt_truthy 只毁 rax；」
// 注曰「助理可用机栈与 rbp 帧，不得久存于 rcx（syscall 毁之）。」
const runtimeText = `
# ---------------------------------------------------------------------------
# 哲浩之谕 runtime — AMD64
# ---------------------------------------------------------------------------
rt_mmap0:
    # esi = 长度 → rax = 匿名内存
    mov  eax, 9
    xor  edi, edi
    mov  edx, 3
    mov  r10d, 34
    mov  r8, -1
    xor  r9d, r9d
    syscall
    ret

rt_exit:
    # edi = 退出之码：先存之，刷清，再寂灭（rt_flush 毁 edi）
    push rdi
    call rt_flush
    pop  rdi
    mov  eax, 60
    syscall

rt_flush:
    lea  r10, [rip+g_outlen]
    mov  rax, [r10]
    test rax, rax
    jz   rt_flush_ret
    mov  edi, 1
    lea  rsi, [rip+g_outbuf]
    mov  rdx, rax
    mov  eax, 1
    syscall
    lea  r10, [rip+g_outlen]
    mov  qword ptr [r10], 0
rt_flush_ret:
    ret

rt_putc:
    # al = 一字节，入缓冲；满则刷清
    push rdi
    push rsi
    lea  r10, [rip+g_outlen]
    mov  rcx, [r10]
    lea  r11, [rip+g_outbuf]
    mov  [r11+rcx], al
    inc  rcx
    mov  [r10], rcx
    cmp  rcx, 4096
    jl   rt_putc_ret
    call rt_flush
rt_putc_ret:
    pop  rsi
    pop  rdi
    ret

rt_fail:
    # rax = 误由（带标记之文）
    lea  r10, [rip+g_errmsg]
    mov  [r10], rax
    jmp  rt_throw

rt_throw:
    mov  r10, [rip+g_trybase]
    cmp  r12, r10
    je   rt_uncaught
    mov  r15, [r12]
    mov  rsp, [r12+8]
    mov  rbp, [r12+16]
    mov  rax, [r12+24]
    add  r12, 32
    lea  r10, [rip+g_errmsg]
    mov  r10, [r10]
    sub  r15, 8
    mov  [r15], r10
    jmp  rax

rt_uncaught:
    call rt_flush
    mov  eax, 1
    mov  edi, 2
    lea  rsi, [rip+s_uncaught]
    mov  edx, 15
    syscall
    lea  r10, [rip+g_errmsg]
    mov  r10, [r10]
    shr  r10, 3
    mov  rsi, r10
    add  rsi, 16
    mov  edx, [r10]
    mov  eax, 1
    mov  edi, 2
    syscall
    mov  eax, 1
    mov  edi, 2
    lea  rsi, [rip+s_nl]
    mov  edx, 1
    syscall
    mov  eax, 60
    mov  edi, 1
    syscall

# ---- 真伪 ---------------------------------------------------------------
rt_truthy:
    # rax = 值 → rax = 一/零（只毁 rax）
    cmp  rax, 1
    je   rt_truthy_no
    cmp  rax, 2
    je   rt_truthy_no
    mov  eax, 1
    ret
rt_truthy_no:
    xor  eax, eax
    ret

# ---- 堆 -----------------------------------------------------------------
rt_alloc:
    # rdi = 字节数 → rax = 指针（十六对齐；只毁 rax 与 r11）
    lea  rax, [rdi+15]
    and  rax, -16
    mov  r11, r14
    add  r14, rax
    mov  rax, r11
    ret

rt_mkfloat:
    # xmm0 = 值 → rax = 带标记之浮
    mov  edi, 16
    call rt_alloc
    movsd qword ptr [rax], xmm0
    shl  rax, 3
    or   rax, 7
    ret

# ---- 数化 ----------------------------------------------------------------
rt_num1:
    # rdi = 带标记之数 → xmm0 = 双精度
    test dil, 7
    jz   rt_num1_int
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    ret
rt_num1_int:
    mov  rax, rdi
    sar  rax, 3
    cvtsi2sd xmm0, rax
    ret

rt_num2:
    # rsi = 带标记之数 → xmm1 = 双精度
    test sil, 7
    jz   rt_num2_int
    mov  rax, rsi
    shr  rax, 3
    movsd xmm1, qword ptr [rax]
    ret
rt_num2_int:
    mov  rax, rsi
    sar  rax, 3
    cvtsi2sd xmm1, rax
    ret

# ---- 算术 ----------------------------------------------------------------
rt_add:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    or   r8d, r9d
    jnz  rt_add_notint
    mov  rax, rsi
    add  rax, rdi
    jmp  rt_add_push
rt_add_notint:
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    cmp  r8d, 4
    jne  rt_add_chk1
    cmp  r9d, 4
    jne  rt_add_chk1
    mov  r10, rdi
    mov  rdi, rsi
    mov  rsi, r10
    call rt_strcat
    jmp  rt_add_push
rt_add_chk1:
    cmp  r8d, 5
    jne  rt_add_flt
    cmp  r9d, 5
    jne  rt_add_flt
    mov  r10, rdi
    mov  rdi, rsi
    mov  rsi, r10
    call rt_listcat
    jmp  rt_add_push
rt_add_flt:
    call rt_num1
    call rt_num2
    addsd xmm0, xmm1
    call rt_mkfloat
rt_add_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_sub:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    or   r8d, r9d
    jnz  rt_sub_flt
    mov  rax, rsi
    sub  rax, rdi
    jmp  rt_sub_push
rt_sub_flt:
    call rt_num1
    call rt_num2
    subsd xmm0, xmm1
    call rt_mkfloat
rt_sub_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_mul:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    cmp  r9d, 4
    jne  rt_mul_chk1
    test r8d, r8d
    jz   rt_mul_str
rt_mul_chk1:
    cmp  r9d, 5
    jne  rt_mul_chk2
    test r8d, r8d
    jz   rt_mul_list
rt_mul_chk2:
    or   r8d, r9d
    jnz  rt_mul_flt
    mov  rax, rsi
    imul rax, rdi
    sar  rax, 3
    jmp  rt_mul_push
rt_mul_str:
    mov  r10, rdi
    mov  rdi, rsi
    mov  rsi, r10
    call rt_strrepeat
    jmp  rt_mul_push
rt_mul_list:
    mov  r10, rdi
    mov  rdi, rsi
    mov  rsi, r10
    call rt_listrepeat
    jmp  rt_mul_push
rt_mul_flt:
    call rt_num1
    call rt_num2
    mulsd xmm0, xmm1
    call rt_mkfloat
rt_mul_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_div:
    # 除恒生浮
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    call rt_num1
    call rt_num2
    divsd xmm1, xmm0         # 首者除后者（后取者先出：甲乙除乃甲÷乙）
    movsd xmm0, xmm1
    call rt_mkfloat
    sub  r15, 8
    mov  [r15], rax
    ret

rt_rem:
    # 余：整数也（带标记相除，余自带符）
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    or   r8d, r9d
    jnz  rt_err_rem_j
    test rdi, rdi
    jz   rt_err_divzero_j
    mov  rax, rsi
    cqo
    idiv rdi
    mov  rax, rdx
    sub  r15, 8
    mov  [r15], rax
    ret

rt_pow:
    # 幂：底之指数皆为整，指数非负
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    test dil, 7
    jnz  rt_err_numpow_j
    test sil, 7
    jnz  rt_err_numpow_j
    mov  rax, rdi
    sar  rax, 3
    test rax, rax
    js   rt_err_numpow_j
    mov  r8, rax
    mov  r9, rsi
    sar  r9, 3
    mov  rax, 8
    xor  r10d, r10d
rt_pow_loop:
    cmp  r10, r8
    jge  rt_pow_done
    imul rax, r9
    inc  r10
    jmp  rt_pow_loop
rt_pow_done:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_neg:
    mov  rdi, [r15]
    add  r15, 8
    test dil, 7
    jnz  rt_neg_chk
    mov  rax, rdi
    neg  rax
    jmp  rt_neg_push
rt_neg_chk:
    movzx r8d, dil
    and  r8d, 7
    cmp  r8d, 7
    jne  rt_err_neg_j
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    pxor xmm1, xmm1
    subsd xmm1, xmm0
    movsd xmm0, xmm1
    call rt_mkfloat
rt_neg_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_abs:
    mov  rdi, [r15]
    add  r15, 8
    test dil, 7
    jnz  rt_abs_chk
    mov  rax, rdi
    test rax, rax
    jns  rt_abs_push
    neg  rax
    jmp  rt_abs_push
rt_abs_chk:
    movzx r8d, dil
    and  r8d, 7
    cmp  r8d, 7
    jne  rt_err_neg_j
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    movq r10, xmm0
    mov  r11, 0x7fffffffffffffff
    and  r10, r11
    movq xmm0, r10
    call rt_mkfloat
rt_abs_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_sqrt:
    mov  rdi, [r15]
    add  r15, 8
    test dil, 7
    jnz  rt_sqrt_f
    mov  rax, rdi
    sar  rax, 3
    cvtsi2sd xmm0, rax
    jmp  rt_sqrt_chk
rt_sqrt_f:
    movzx r8d, dil
    and  r8d, 7
    cmp  r8d, 7
    jne  rt_err_sqrt_j
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
rt_sqrt_chk:
    pxor xmm1, xmm1
    ucomisd xmm1, xmm0
    ja   rt_err_sqrt_j
    sqrtsd xmm0, xmm0
    call rt_mkfloat
    sub  r15, 8
    mov  [r15], rax
    ret

rt_floor:
    # 整：向下取整（floor）
    mov  rdi, [r15]
    add  r15, 8
    test dil, 7
    jz   rt_floor_push
    movzx r8d, dil
    and  r8d, 7
    cmp  r8d, 7
    jne  rt_err_num_j
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    cvttsd2si rax, xmm0
    mov  r10, rax
    cvtsi2sd xmm1, rax
    ucomisd xmm1, xmm0
    seta cl
    movzx ecx, cl
    mov  rax, r10
    sub  rax, rcx
    shl  rax, 3
rt_floor_push:
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 较（r11d：0等 1异 2逾 3逊 4逾等 5逊等）------------------------------
rt_cmp:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    cmp  r11d, 2
    jge  rt_cmp_ord
    cmp  r8d, r9d
    jne  rt_cmp_ne
    cmp  r8d, 1
    je   rt_cmp_eq1
    cmp  r8d, 2
    je   rt_cmp_eq1
    cmp  r8d, 3
    je   rt_cmp_eq1
    cmp  r8d, 4
    je   rt_cmp_streq
    test r8d, r8d
    jz   rt_cmp_inteq
    cmp  r8d, 7
    je   rt_cmp_floateq
    # 列：指针同否
    mov  rax, rsi
    xor  rax, rdi
    sete al
    movzx eax, al
    jmp  rt_cmp_eqfin
rt_cmp_ne:
    # 标异：唯整浮相杂可比，余者必不等
    test r8d, r8d
    jnz  rt_cmp_ne_chk
    cmp  r9d, 7
    je   rt_cmp_mixed_eq
    jmp  rt_cmp_zero
rt_cmp_ne_chk:
    cmp  r8d, 7
    jne  rt_cmp_zero
    test r9d, r9d
    jz   rt_cmp_mixed_eq
    jmp  rt_cmp_zero
rt_cmp_mixed_eq:
    call rt_num1
    call rt_num2
    ucomisd xmm0, xmm1
    sete al
    setnp cl
    and  al, cl
    movzx eax, al
    jmp  rt_cmp_eqfin
rt_cmp_eq1:
    mov  eax, 1
    jmp  rt_cmp_eqfin
rt_cmp_inteq:
    mov  rax, rsi
    xor  rax, rdi
    sete al
    movzx eax, al
    jmp  rt_cmp_eqfin
rt_cmp_streq:
    # 注：r11 载较码，不可毁——以 r9 代之
    mov  r10, rsi
    shr  r10, 3
    mov  r9, rdi
    shr  r9, 3
    mov  rax, [r10]
    cmp  rax, [r9]
    jne  rt_cmp_zero
    mov  r8, [r10]
    lea  r10, [r10+16]
    lea  r9, [r9+16]
    xor  rcx, rcx
rt_cmp_sb:
    cmp  rcx, r8
    jge  rt_cmp_eq1
    movzx edx, byte ptr [r10+rcx]
    movzx eax, byte ptr [r9+rcx]
    cmp  dl, al
    jne  rt_cmp_zero
    inc  rcx
    jmp  rt_cmp_sb
rt_cmp_floateq:
    call rt_num1
    call rt_num2
    ucomisd xmm0, xmm1
    sete al
    setnp cl
    and  al, cl
    movzx eax, al
    jmp  rt_cmp_eqfin
rt_cmp_zero:
    xor  eax, eax
rt_cmp_eqfin:
    # 码一（异）者反其果
    test r11d, 1
    jz   rt_cmp_eqgo
    xor  eax, 1
rt_cmp_eqgo:
    test eax, eax
    jz   rt_cmp_false
    mov  eax, 3
    jmp  rt_cmp_bool_push
rt_cmp_ord:
    cmp  r8d, 7
    je   rt_cmp_ord2
    test r8d, r8d
    jz   rt_cmp_ord2
    jmp  rt_err_cmp_j
rt_cmp_ord2:
    cmp  r9d, 7
    je   rt_cmp_ord3
    test r9d, r9d
    jz   rt_cmp_ord3
    jmp  rt_err_cmp_j
rt_cmp_ord3:
    test r8d, r8d
    jnz  rt_cmp_ord_f
    test r9d, r9d
    jnz  rt_cmp_ord_f
    # 皆整
    mov  rax, rsi
    sar  rax, 3
    mov  r10, rdi
    sar  r10, 3
    cmp  r11d, 2
    je   rt_cmp_i_gt
    cmp  r11d, 3
    je   rt_cmp_i_lt
    cmp  r11d, 4
    je   rt_cmp_i_ge
    cmp  rax, r10
    jle  rt_cmp_true
    jmp  rt_cmp_false
rt_cmp_i_gt:
    cmp  rax, r10
    jg   rt_cmp_true
    jmp  rt_cmp_false
rt_cmp_i_lt:
    cmp  rax, r10
    jl   rt_cmp_true
    jmp  rt_cmp_false
rt_cmp_i_ge:
    cmp  rax, r10
    jge  rt_cmp_true
    jmp  rt_cmp_false
rt_cmp_ord_f:
    call rt_num1
    call rt_num2
    cmp  r11d, 2
    je   rt_cmp_f_gt
    cmp  r11d, 3
    je   rt_cmp_f_lt
    cmp  r11d, 4
    je   rt_cmp_f_ge
    ucomisd xmm1, xmm0
    setbe al
    setnp cl
    and  al, cl
    movzx eax, al
    jmp  rt_cmp_bool_push
rt_cmp_f_gt:
    ucomisd xmm1, xmm0
    seta al
    movzx eax, al
    jmp  rt_cmp_bool_push
rt_cmp_f_lt:
    ucomisd xmm0, xmm1
    seta al
    movzx eax, al
    jmp  rt_cmp_bool_push
rt_cmp_f_ge:
    ucomisd xmm0, xmm1
    setbe al
    setnp cl
    and  al, cl
    movzx eax, al
    jmp  rt_cmp_bool_push
rt_cmp_true:
    mov  eax, 3
    jmp  rt_cmp_bool_push
rt_cmp_false:
    mov  eax, 2
rt_cmp_bool_push:
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 逻辑 ----------------------------------------------------------------
rt_and:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  rax, rdi
    call rt_truthy
    mov  r8, rax
    mov  rax, rsi
    call rt_truthy
    and  rax, r8
    jz   rt_and_false
    mov  eax, 3
    jmp  rt_and_push
rt_and_false:
    mov  eax, 2
rt_and_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_or:
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  rax, rdi
    call rt_truthy
    mov  r8, rax
    mov  rax, rsi
    call rt_truthy
    or   rax, r8
    jz   rt_or_false
    mov  eax, 3
    jmp  rt_or_push
rt_or_false:
    mov  eax, 2
rt_or_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_not:
    mov  rdi, [r15]
    add  r15, 8
    mov  rax, rdi
    call rt_truthy
    test rax, rax
    jnz  rt_not_false
    mov  eax, 3
    jmp  rt_not_push
rt_not_false:
    mov  eax, 2
rt_not_push:
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 化类 ----------------------------------------------------------------
rt_tobool:
    mov  rdi, [r15]
    add  r15, 8
    mov  rax, rdi
    call rt_truthy
    test rax, rax
    jz   rt_tobool_false
    mov  eax, 3
    jmp  rt_tobool_push
rt_tobool_false:
    mov  eax, 2
rt_tobool_push:
    sub  r15, 8
    mov  [r15], rax
    ret

rt_type:
    mov  rdi, [r15]
    add  r15, 8
    movzx eax, dil
    and  eax, 7
    cmp  eax, 1
    je   rt_type_nil
    cmp  eax, 2
    je   rt_type_bool
    cmp  eax, 3
    je   rt_type_bool
    test eax, eax
    jz   rt_type_num
    cmp  eax, 7
    je   rt_type_num
    cmp  eax, 4
    je   rt_type_str
    cmp  eax, 6
    je   rt_type_dict
    lea  rax, [rip+s_列]
    jmp  rt_type_tag
rt_type_nil:
    lea  rax, [rip+s_虚]
    jmp  rt_type_tag
rt_type_bool:
    lea  rax, [rip+s_阴阳]
    jmp  rt_type_tag
rt_type_num:
    lea  rax, [rip+s_数]
    jmp  rt_type_tag
rt_type_str:
    lea  rax, [rip+s_文]
    jmp  rt_type_tag
rt_type_dict:
    lea  rax, [rip+s_典]
rt_type_tag:
    shl  rax, 3
    or   rax, 4
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 宣聆 ----------------------------------------------------------------
rt_printstr:
    # rdi = 带标记之文，逐字入缓冲
    mov  rax, rdi
    shr  rax, 3
    mov  rsi, rax
    mov  rdx, [rsi]
    lea  rsi, [rsi+16]
    xor  rcx, rcx
rt_printstr_loop:
    cmp  rcx, rdx
    jge  rt_printstr_done
    push rsi
    push rdx
    push rcx
    movzx eax, byte ptr [rsi+rcx]
    call rt_putc
    pop  rcx
    pop  rdx
    pop  rsi
    inc  rcx
    jmp  rt_printstr_loop
rt_printstr_done:
    ret

rt_print:
    mov  rdi, [r15]
    add  r15, 8
    call rt_tostr
    mov  rdi, rax
    call rt_printstr
    mov  eax, 10
    call rt_putc
    ret

rt_printn:
    mov  rdi, [r15]
    add  r15, 8
    call rt_tostr
    mov  rdi, rax
    call rt_printstr
    ret

rt_read:
    # 聆一行：至换行或尽；空尽则虚
    call rt_flush
    xor  r10d, r10d
rt_read_loop:
    xor  eax, eax
    xor  edi, edi
    lea  rsi, [rip+g_ch]
    mov  edx, 1
    syscall
    cmp  rax, 1
    jl   rt_read_eof
    lea  r11, [rip+g_ch]
    movzx ecx, byte ptr [r11]
    cmp  cl, 10
    je   rt_read_build
    lea  r11, [rip+g_readbuf]
    mov  [r11+r10], cl
    inc  r10
    cmp  r10, 4095
    jl   rt_read_loop
rt_read_build:
    lea  rdi, [r10+16]
    call rt_alloc
    mov  [rax], r10
    mov  qword ptr [rax+8], 0
    lea  rsi, [rip+g_readbuf]
    xor  rdx, rdx
rt_read_copy:
    cmp  rdx, r10
    jge  rt_read_done
    movzx ecx, byte ptr [rsi+rdx]
    mov  [rax+rdx+16], cl
    inc  rdx
    jmp  rt_read_copy
rt_read_done:
    shl  rax, 3
    or   rax, 4
    sub  r15, 8
    mov  [r15], rax
    ret
rt_read_eof:
    test r10, r10
    jnz  rt_read_build
    mov  eax, 1
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 文之诸务 ------------------------------------------------------------
rt_strcat:
    # rdi = X，rsi = Y（带标记）→ rax = X+Y
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  [rbp-24], r8
    mov  rax, rsi
    shr  rax, 3
    mov  [rbp-16], rax
    mov  r8, [rax]
    mov  [rbp-32], r8
    mov  rdi, [rbp-24]
    add  rdi, [rbp-32]
    add  rdi, 16
    call rt_alloc
    mov  r9, rax
    mov  rax, [rbp-24]
    add  rax, [rbp-32]
    mov  [r9], rax
    mov  qword ptr [r9+8], 0
    lea  r10, [r9+16]
    mov  rax, [rbp-8]
    lea  rsi, [rax+16]
    mov  r8, [rbp-24]
    xor  rcx, rcx
rt_strcat_a:
    cmp  rcx, r8
    jge  rt_strcat_b
    movzx edx, byte ptr [rsi+rcx]
    mov  [r10+rcx], dl
    inc  rcx
    jmp  rt_strcat_a
rt_strcat_b:
    mov  rax, [rbp-16]
    lea  rsi, [rax+16]
    mov  r8, [rbp-24]
    mov  rdi, [rbp-32]
    xor  rcx, rcx
rt_strcat_b2:
    cmp  rcx, rdi
    jge  rt_strcat_done
    movzx edx, byte ptr [rsi+rcx]
    lea  r11, [r10+r8]
    mov  [r11+rcx], dl
    inc  rcx
    jmp  rt_strcat_b2
rt_strcat_done:
    mov  rax, r9
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

rt_strrepeat:
    # rdi = 文，rsi = 带标记之倍数 → rax = 文
    push rbp
    mov  rbp, rsp
    sub  rsp, 48
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  [rbp-16], r8
    mov  rax, rsi
    sar  rax, 3
    imul rax, r8
    test rax, rax
    jns  rt_strrep_ok
    xor  eax, eax
rt_strrep_ok:
    mov  [rbp-24], rax
    lea  rdi, [rax+16]
    call rt_alloc
    mov  [rbp-32], rax
    mov  rcx, [rbp-24]
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    lea  r10, [rax+16]
    mov  r9, [rbp-8]
    lea  rsi, [r9+16]
    xor  rcx, rcx
rt_strrepeat_loop:
    cmp  rcx, [rbp-24]
    jge  rt_strrepeat_done
    xor  edx, edx
    mov  rax, rcx
    div  qword ptr [rbp-16]
    movzx r8d, byte ptr [rsi+rdx]
    mov  [r10+rcx], r8b
    inc  rcx
    jmp  rt_strrepeat_loop
rt_strrepeat_done:
    mov  rax, [rbp-32]
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

# ---- 列之诸务 ------------------------------------------------------------
rt_makelist:
    # 值栈顶为数 n，其下 n 元 → 弹之，成列推回
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  rdi, [r15]
    add  r15, 8
    sar  rdi, 3
    test rdi, rdi
    js   rt_err_badlist_j
    mov  [rbp-8], rdi
    mov  edi, 32
    call rt_alloc
    mov  [rbp-16], rax
    mov  rdi, [rbp-8]
    shl  rdi, 3
    cmp  rdi, 16
    jge  rt_makelist_a1
    mov  edi, 16
rt_makelist_a1:
    call rt_alloc
    mov  [rbp-24], rax
    mov  r8, [rbp-16]
    mov  rcx, [rbp-8]
    mov  [r8], rcx
    mov  [r8+8], rcx
    mov  r9, [rbp-24]
    mov  [r8+16], r9
rt_makelist_loop:
    mov  rcx, [rbp-8]
    test rcx, rcx
    jz   rt_makelist_done
    dec  rcx
    mov  [rbp-8], rcx
    mov  rdx, [r15]
    add  r15, 8
    mov  [r9+rcx*8], rdx
    jmp  rt_makelist_loop
rt_makelist_done:
    mov  rax, [rbp-16]
    shl  rax, 3
    or   rax, 5
    sub  r15, 8
    mov  [r15], rax
    mov  rsp, rbp
    pop  rbp
    ret

rt_len:
    mov  rdi, [r15]
    add  r15, 8
    movzx eax, dil
    and  eax, 7
    cmp  eax, 4
    je   rt_len_ok
    cmp  eax, 5
    je   rt_len_ok
    cmp  eax, 6
    je   rt_len_ok
    lea  rax, [rip+err_len]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_len_ok:
    mov  rax, rdi
    shr  rax, 3
    mov  rax, [rax]
    shl  rax, 3
    sub  r15, 8
    mov  [r15], rax
    ret

rt_get:
    # 栈：[容器, 序]（序居顶）
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    sar  rdi, 3
    mov  [rbp-8], rdi
    movzx eax, sil
    and  eax, 7
    cmp  eax, 5
    je   rt_get_list
    cmp  eax, 4
    je   rt_get_str
    lea  rax, [rip+err_get]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_get_list:
    mov  rax, rsi
    shr  rax, 3
    mov  r8, [rax]
    mov  r9, [rax+16]
    mov  rcx, [rbp-8]
    test rcx, rcx
    js   rt_get_oob
    cmp  rcx, r8
    jge  rt_get_oob
    mov  rax, [r9+rcx*8]
    jmp  rt_get_push
rt_get_str:
    mov  rax, rsi
    shr  rax, 3
    mov  r8, [rax]
    mov  r9, rax
    mov  rcx, [rbp-8]
    test rcx, rcx
    js   rt_get_oob
    cmp  rcx, r8
    jge  rt_get_oob
    mov  rdi, 32
    call rt_alloc
    mov  qword ptr [rax], 1
    mov  qword ptr [rax+8], 0
    mov  r10, r9
    add  r10, 16
    add  r10, [rbp-8]
    movzx edx, byte ptr [r10]
    mov  [rax+16], dl
    shl  rax, 3
    or   rax, 4
rt_get_push:
    sub  r15, 8
    mov  [r15], rax
    mov  rsp, rbp
    pop  rbp
    ret
rt_get_oob:
    lea  rax, [rip+err_oob]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_set:
    # 栈：[值, 序, 容器]（容器居顶）
    push rbp
    mov  rbp, rsp
    sub  rsp, 16
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  rdx, [r15]
    add  r15, 8
    movzx eax, dil
    and  eax, 7
    cmp  eax, 5
    jne  rt_set_err
    mov  rax, rdi
    shr  rax, 3
    mov  r8, [rax]
    mov  r9, [rax+16]
    sar  rsi, 3
    test rsi, rsi
    js   rt_set_oob
    cmp  rsi, r8
    jge  rt_set_oob
    mov  [r9+rsi*8], rdx
    mov  rsp, rbp
    pop  rbp
    ret
rt_set_err:
    lea  rax, [rip+err_set]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_set_oob:
    lea  rax, [rip+err_oob]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_append:
    # 栈：[列, 值]（值居顶）
    push rbp
    mov  rbp, rsp
    sub  rsp, 48
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  [rbp-16], rdi
    movzx eax, sil
    and  eax, 7
    cmp  eax, 5
    jne  rt_append_err
    mov  rax, rsi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  r9, [rax+8]
    mov  r10, [rax+16]
    mov  [rbp-24], r8
    mov  [rbp-32], r9
    mov  [rbp-40], r10
    cmp  r8, r9
    jl   rt_append_place
    lea  rcx, [r9+r9]
    cmp  rcx, 4
    jge  rt_append_g2
    mov  ecx, 4
rt_append_g2:
    mov  [rbp-32], rcx
    mov  rdi, rcx
    shl  rdi, 3
    call rt_alloc
    mov  r11, rax
    xor  rcx, rcx
rt_append_cp:
    cmp  rcx, [rbp-24]
    jge  rt_append_cpd
    mov  rdx, [r10+rcx*8]
    mov  [r11+rcx*8], rdx
    inc  rcx
    jmp  rt_append_cp
rt_append_cpd:
    mov  rcx, [rbp-8]
    mov  rax, [rbp-32]
    mov  [rcx+8], rax
    mov  [rcx+16], r11
    mov  [rbp-40], r11
rt_append_place:
    mov  rcx, [rbp-24]
    mov  rdx, [rbp-40]
    mov  rax, [rbp-16]
    mov  [rdx+rcx*8], rax
    inc  rcx
    mov  rdx, [rbp-8]
    mov  [rdx], rcx
    mov  rsp, rbp
    pop  rbp
    ret
rt_append_err:
    lea  rax, [rip+err_append]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_popback:
    mov  rdi, [r15]
    add  r15, 8
    movzx eax, dil
    and  eax, 7
    cmp  eax, 5
    jne  rt_popback_err
    mov  rax, rdi
    shr  rax, 3
    mov  r8, [rax]
    test r8, r8
    jz   rt_popback_empty
    mov  r9, [rax+16]
    dec  r8
    mov  [rax], r8
    mov  rax, [r9+r8*8]
    sub  r15, 8
    mov  [r15], rax
    ret
rt_popback_err:
    lea  rax, [rip+err_append]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_popback_empty:
    lea  rax, [rip+err_empty]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_listcat:
    # rdi = X，rsi = Y（带标记之列）→ rax = X+Y
    push rbp
    mov  rbp, rsp
    sub  rsp, 80
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  [rbp-24], r8
    mov  r9, [rax+16]
    mov  [rbp-64], r9
    mov  rax, rsi
    shr  rax, 3
    mov  [rbp-16], rax
    mov  r10, [rax]
    mov  [rbp-32], r10
    mov  r11, [rax+16]
    mov  [rbp-72], r11
    mov  rax, r8
    add  rax, r10
    mov  [rbp-40], rax
    mov  edi, 32
    call rt_alloc
    mov  [rbp-48], rax
    mov  rdx, [rbp-40]
    mov  [rax], rdx
    mov  [rax+8], rdx
    mov  rdi, rdx
    shl  rdi, 3
    cmp  rdi, 16
    jge  rt_listcat_a1
    mov  edi, 16
rt_listcat_a1:
    call rt_alloc
    mov  [rbp-56], rax
    mov  rdx, [rbp-48]
    mov  rcx, [rbp-56]
    mov  [rdx+16], rcx
    xor  rcx, rcx
rt_listcat_ca:
    cmp  rcx, [rbp-24]
    jge  rt_listcat_cb
    mov  r9, [rbp-64]
    mov  rax, [r9+rcx*8]
    mov  r8, [rbp-56]
    mov  [r8+rcx*8], rax
    inc  rcx
    jmp  rt_listcat_ca
rt_listcat_cb:
    xor  rcx, rcx
rt_listcat_cb2:
    cmp  rcx, [rbp-32]
    jge  rt_listcat_done
    mov  r11, [rbp-72]
    mov  rax, [r11+rcx*8]
    mov  r8, [rbp-56]
    mov  rdx, [rbp-24]
    lea  rdi, [r8+rdx*8]
    mov  [rdi+rcx*8], rax
    inc  rcx
    jmp  rt_listcat_cb2
rt_listcat_done:
    mov  rax, [rbp-48]
    shl  rax, 3
    or   rax, 5
    mov  rsp, rbp
    pop  rbp
    ret

rt_listrepeat:
    # rdi = 列，rsi = 带标记之倍数 → rax = 列
    push rbp
    mov  rbp, rsp
    sub  rsp, 64
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  [rbp-24], r8
    mov  r9, [rax+16]
    mov  [rbp-32], r9
    mov  rax, rsi
    sar  rax, 3
    imul rax, r8
    test rax, rax
    jns  rt_listrep_ok
    xor  eax, eax
rt_listrep_ok:
    mov  [rbp-40], rax
    mov  edi, 32
    call rt_alloc
    mov  [rbp-48], rax
    mov  rdx, [rbp-40]
    mov  [rax], rdx
    mov  [rax+8], rdx
    mov  rdi, rdx
    shl  rdi, 3
    cmp  rdi, 16
    jge  rt_listrep_a1
    mov  edi, 16
rt_listrep_a1:
    call rt_alloc
    mov  [rbp-56], rax
    mov  rdx, [rbp-48]
    mov  rcx, [rbp-56]
    mov  [rdx+16], rcx
    xor  rcx, rcx
rt_listrep_loop:
    cmp  rcx, [rbp-40]
    jge  rt_listrep_done
    xor  edx, edx
    mov  rax, rcx
    div  qword ptr [rbp-24]
    mov  r8, [rbp-32]
    mov  rax, [r8+rdx*8]
    mov  r9, [rbp-56]
    mov  [r9+rcx*8], rax
    inc  rcx
    jmp  rt_listrep_loop
rt_listrep_done:
    mov  rax, [rbp-48]
    shl  rax, 3
    or   rax, 5
    mov  rsp, rbp
    pop  rbp
    ret

# ---- 天机 ----------------------------------------------------------------
rt_rand:
    mov  rdi, [r15]
    add  r15, 8
    sar  rdi, 3
    test rdi, rdi
    jg   rt_rand_1
    lea  rax, [rip+err_rand]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_rand_1:
    lea  r9, [rip+g_seed]
    mov  rax, [r9]
    imul rax, rax, 1103515245
    add  rax, 12345
    mov  [r9], rax
    xor  edx, edx
    div  rdi
    mov  rax, rdx
    shl  rax, 3
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 化文 ----------------------------------------------------------------
rt_tostr:
    # rdi = 值 → rax = 带标记之文
    movzx eax, dil
    and  eax, 7
    cmp  eax, 4
    jne  rt_tostr_1
    mov  rax, rdi
    ret
rt_tostr_1:
    test eax, eax
    jnz  rt_tostr_2
    jmp  rt_inttostr
rt_tostr_2:
    cmp  eax, 7
    jne  rt_tostr_3
    jmp  rt_floattostr
rt_tostr_3:
    cmp  eax, 5
    jne  rt_tostr_3b
    jmp  rt_listtostr
rt_tostr_3b:
    cmp  eax, 6
    jne  rt_tostr_4
    jmp  rt_dicttostr
rt_tostr_4:
    cmp  eax, 1
    jne  rt_tostr_5
    lea  rax, [rip+s_虚]
    jmp  rt_tostr_tag
rt_tostr_5:
    cmp  eax, 2
    jne  rt_tostr_6
    lea  rax, [rip+s_阴]
    jmp  rt_tostr_tag
rt_tostr_6:
    lea  rax, [rip+s_阳]
rt_tostr_tag:
    shl  rax, 3
    or   rax, 4
    ret

rt_inttostr:
    # rdi = 带标记之整 → rax = 带标记之文
    push rbp
    mov  rbp, rsp
    sub  rsp, 40
    mov  rax, rdi
    sar  rax, 3
    xor  r8d, r8d
    test rax, rax
    jns  rt_inttostr_p
    neg  rax
    mov  r8d, 1
rt_inttostr_p:
    mov  [rbp-8], rax
    mov  [rbp-16], r8
    lea  r9, [rip+g_itobuf]
    xor  rcx, rcx
    test rax, rax
    jnz  rt_inttostr_dig
    mov  byte ptr [r9], 48
    mov  ecx, 1
    jmp  rt_inttostr_digdone
rt_inttostr_dig:
    xor  edx, edx
    mov  r10, 10
    div  r10
    add  dl, 48
    mov  [r9+rcx], dl
    inc  rcx
    test rax, rax
    jnz  rt_inttostr_dig
rt_inttostr_digdone:
    mov  [rbp-24], rcx
    mov  rdi, rcx
    add  rdi, [rbp-16]
    add  rdi, 16
    call rt_alloc
    mov  [rbp-32], rax
    mov  rcx, [rbp-24]
    add  rcx, [rbp-16]
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    lea  r9, [rax+16]
    cmp  qword ptr [rbp-16], 0
    je   rt_inttostr_nosign
    mov  byte ptr [r9], 45
    inc  r9
rt_inttostr_nosign:
    lea  r10, [rip+g_itobuf]
    xor  rdx, rdx
rt_inttostr_copy:
    cmp  rdx, [rbp-24]
    jge  rt_inttostr_done
    mov  rcx, [rbp-24]
    sub  rcx, rdx
    dec  rcx
    movzx r8d, byte ptr [r10+rcx]
    mov  [r9+rdx], r8b
    inc  rdx
    jmp  rt_inttostr_copy
rt_inttostr_done:
    mov  rax, [rbp-32]
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

rt_floattostr:
    # rdi = 带标记之浮 → rax = 带标记之文
    # 浮恒有点（整者如「2.0」）；小数取十五位，以第十六位合其近，后去尾零
    push rbp
    mov  rbp, rsp
    sub  rsp, 80
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    movq r10, xmm0
    mov  r8, r10
    shr  r8, 63
    mov  [rbp-8], r8          # 符
    mov  qword ptr [rbp-24], 0
    cmp  r8, 0
    je   rt_ftostr_noi
    mov  qword ptr [rbp-24], 1 # 有符：整部之字自一位始
rt_ftostr_noi:
    mov  r11, 0x7fffffffffffffff
    and  r10, r11
    movq xmm0, r10             # |x|
    cvttsd2si rax, xmm0
    mov  [rbp-48], rax         # 整部
    cvtsi2sd xmm1, rax
    subsd xmm0, xmm1
    movq rax, xmm0
    mov  [rbp-56], rax         # 小数（位样）
    # 小数合近至十五位（一举乘 10^15 而加半，无累乘之漂）
    movq xmm0, [rbp-56]
    mulsd xmm0, [rip+f_1e15]
    addsd xmm0, [rip+f_half]
    cvttsd2si rax, xmm0
    mov  r9, 1000000000000000
    cmp  rax, r9
    jl   rt_ftostr_nok
    sub  rax, r9
    inc  qword ptr [rbp-48]    # 诸九皆进，整部进一
rt_ftostr_nok:
    mov  [rbp-64], rax         # N（十五位之整）
rt_ftostr_restart:
    lea  rsi, [rip+g_ftobuf]
    xor  rcx, rcx
    cmp  qword ptr [rbp-8], 0
    je   rt_ftostr_ns
    mov  byte ptr [rsi], 45
    mov  rcx, 1
rt_ftostr_ns:
    mov  rax, [rbp-48]
    xor  r8d, r8d
    test rax, rax
    jnz  rt_ftostr_dig
    push 48
    mov  r8d, 1
    jmp  rt_ftostr_digdone
rt_ftostr_dig:
    xor  edx, edx
    mov  r9, 10
    div  r9
    add  dl, 48
    push rdx
    inc  r8d
    test rax, rax
    jnz  rt_ftostr_dig
rt_ftostr_digdone:
rt_ftostr_ipw:
    test r8d, r8d
    jz   rt_ftostr_dot
    pop  rax
    mov  [rsi+rcx], al
    inc  rcx
    dec  r8d
    jmp  rt_ftostr_ipw
rt_ftostr_dot:
    mov  byte ptr [rsi+rcx], 46
    inc  rcx
    mov  [rbp-16], rcx         # 点之位
    # N 之十五位，前补〇（逆取而顺写）
    mov  rax, [rbp-64]
    xor  r8d, r8d
    mov  r9d, 15
rt_ftostr_n15:
    xor  edx, edx
    mov  r10, 10
    div  r10
    add  dl, 48
    push rdx
    inc  r8d
    dec  r9d
    jnz  rt_ftostr_n15
rt_ftostr_w15:
    pop  rax
    mov  [rsi+rcx], al
    inc  rcx
    dec  r8d
    jnz  rt_ftostr_w15
rt_ftostr_trim:
    mov  r9, [rbp-16]
    inc  r9
rt_ftostr_trim2:
    cmp  rcx, r9
    jle  rt_ftostr_fin
    cmp  byte ptr [rsi+rcx-1], 48
    jne  rt_ftostr_fin
    dec  rcx
    jmp  rt_ftostr_trim2
rt_ftostr_fin:
    mov  rdi, rcx
    add  rdi, 16
    call rt_alloc
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    lea  rsi, [rip+g_ftobuf]
    lea  rdi, [rax+16]
    xor  rdx, rdx
rt_ftostr_cp:
    cmp  rdx, rcx
    jge  rt_ftostr_ret
    movzx r8d, byte ptr [rsi+rdx]
    mov  [rdi+rdx], r8b
    inc  rdx
    jmp  rt_ftostr_cp
rt_ftostr_ret:
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

rt_listtostr:
    # rdi = 带标记之列 → rax = 〔…〕（文中之元得「」之饰）
    push rbp
    mov  rbp, rsp
    sub  rsp, 8192
    mov  rax, rdi
    shr  rax, 3
    mov  r8, [rax]
    mov  [rbp-8], r8
    mov  r9, [rax+16]
    mov  [rbp-16], r9
    lea  r10, [rbp-8192]
    mov  byte ptr [r10], 0xE3
    mov  byte ptr [r10+1], 0x80
    mov  byte ptr [r10+2], 0x94
    mov  qword ptr [rbp-24], 3
    mov  qword ptr [rbp-32], 0
rt_ltostr_loop:
    mov  rax, [rbp-32]
    cmp  rax, [rbp-8]
    jge  rt_ltostr_close
    test rax, rax
    jz   rt_ltostr_elem
    lea  r10, [rbp-8192]
    mov  rcx, [rbp-24]
    mov  byte ptr [r10+rcx], 0xEF
    mov  byte ptr [r10+rcx+1], 0xBC
    mov  byte ptr [r10+rcx+2], 0x8C
    add  rcx, 3
    mov  [rbp-24], rcx
rt_ltostr_elem:
    mov  r9, [rbp-16]
    mov  rax, [rbp-32]
    mov  r11, [r9+rax*8]
    mov  [rbp-48], r11
    movzx eax, r11b
    and  eax, 7
    cmp  eax, 4
    jne  rt_ltostr_noq
    mov  qword ptr [rbp-40], 1
    lea  r10, [rbp-8192]
    mov  rcx, [rbp-24]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x8C
    add  rcx, 3
    mov  [rbp-24], rcx
    jmp  rt_ltostr_conv
rt_ltostr_noq:
    mov  qword ptr [rbp-40], 0
rt_ltostr_conv:
    mov  rdi, [rbp-48]
    call rt_tostr
    mov  rsi, rax
    shr  rsi, 3
    mov  r8, [rsi]
    lea  rsi, [rsi+16]
    lea  r10, [rbp-8192]
    mov  r9, [rbp-24]
    lea  rdi, [r10+r9]
    xor  rcx, rcx
rt_ltostr_cp:
    cmp  rcx, r8
    jge  rt_ltostr_cpdone
    movzx edx, byte ptr [rsi+rcx]
    mov  [rdi+rcx], dl
    inc  rcx
    jmp  rt_ltostr_cp
rt_ltostr_cpdone:
    add  r9, r8
    mov  [rbp-24], r9
    cmp  qword ptr [rbp-40], 0
    je   rt_ltostr_next
    lea  r10, [rbp-8192]
    mov  rcx, [rbp-24]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x8D
    add  rcx, 3
    mov  [rbp-24], rcx
rt_ltostr_next:
    inc  qword ptr [rbp-32]
    jmp  rt_ltostr_loop
rt_ltostr_close:
    lea  r10, [rbp-8192]
    mov  rcx, [rbp-24]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x95
    add  rcx, 3
    mov  [rbp-24], rcx
    lea  rdi, [rcx+16]
    call rt_alloc
    mov  r9, rax
    mov  rcx, [rbp-24]
    mov  [r9], rcx
    mov  qword ptr [r9+8], 0
    lea  r10, [rbp-8192]
    lea  r11, [r9+16]
    xor  rdx, rdx
rt_ltostr_fcopy:
    cmp  rdx, rcx
    jge  rt_ltostr_ret
    movzx r8d, byte ptr [r10+rdx]
    mov  [r11+rdx], r8b
    inc  rdx
    jmp  rt_ltostr_fcopy
rt_ltostr_ret:
    mov  rax, r9
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

# ---- 化数 ----------------------------------------------------------------
rt_tonum:
    mov  rdi, [r15]
    add  r15, 8
    movzx eax, dil
    and  eax, 7
    test eax, eax
    jz   rt_tonum_push
    cmp  eax, 7
    je   rt_tonum_push
    cmp  eax, 4
    jne  rt_tonum_err
    call rt_strtonum
    jmp  rt_tonum_push
rt_tonum_push:
    sub  r15, 8
    mov  [r15], rax
    ret
rt_tonum_err:
    lea  rax, [rip+err_num]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_strtonum:
    # rdi = 带标记之文 → rax = 带标记之数（整或浮）
    # 兼识十进数字与汉数：可冠负（负/-），可带点（点/.），兼识两廿卅
    push rbp
    mov  rbp, rsp
    sub  rsp, 112
    mov  rax, rdi
    shr  rax, 3
    mov  r10, [rax]          # 长
    lea  rdx, [rax+16]       # 字节基
    xor  r9, r9
rt_stn_t0:
    cmp  r9, r10
    jge  rt_stn_t0d
    movzx eax, byte ptr [rdx+r9]
    cmp  al, 32
    je   rt_stn_t0n
    cmp  al, 9
    je   rt_stn_t0n
    cmp  al, 10
    je   rt_stn_t0n
    cmp  al, 13
    jne  rt_stn_t0d
rt_stn_t0n:
    inc  r9
    jmp  rt_stn_t0
rt_stn_t0d:
    mov  r8, r10
rt_stn_t2:
    cmp  r8, r9
    jle  rt_stn_t2d
    movzx eax, byte ptr [rdx+r8-1]
    cmp  al, 32
    je   rt_stn_t2n
    cmp  al, 9
    je   rt_stn_t2n
    cmp  al, 10
    je   rt_stn_t2n
    cmp  al, 13
    jne  rt_stn_t2d
rt_stn_t2n:
    dec  r8
    jmp  rt_stn_t2
rt_stn_t2d:
    mov  [rbp-8], rdx        # 基
    mov  [rbp-16], r9        # 首
    mov  [rbp-24], r8        # 末
    mov  qword ptr [rbp-32], 0   # total
    mov  qword ptr [rbp-40], 0   # section
    mov  qword ptr [rbp-48], 0   # cur
    mov  qword ptr [rbp-56], 0   # （fracv 寄于 r11）
    mov  qword ptr [rbp-64], 0   # fracn
    mov  qword ptr [rbp-72], 0   # infrac（曾见点乎）
    mov  qword ptr [rbp-80], 0   # seen
    xor  r8d, r8d            # 负
    xor  r11d, r11d          # fracv
    mov  r9, [rbp-16]
    cmp  r9, [rbp-24]
    jge  rt_stn_err
    movzx eax, byte ptr [rdx+r9]
    cmp  al, 45              # '-'
    jne  rt_stn_neg2
    inc  r8d
    inc  r9
    jmp  rt_stn_neg3
rt_stn_neg2:
    cmp  byte ptr [rdx+r9], 0xE8
    jne  rt_stn_neg3
    cmp  byte ptr [rdx+r9+1], 0xB4
    jne  rt_stn_neg3
    cmp  byte ptr [rdx+r9+2], 0x9F
    jne  rt_stn_neg3
    inc  r8d
    add  r9, 3
rt_stn_neg3:
    cmp  r9, [rbp-24]
    jge  rt_stn_err
rt_stn_loop:
    cmp  r9, [rbp-24]
    jge  rt_stn_fin
    mov  qword ptr [rbp-88], 3
    mov  qword ptr [rbp-96], 0
    mov  rdx, [rbp-8]
    movzx eax, byte ptr [rdx+r9]
    cmp  al, 48
    jl   rt_stn_dot
    cmp  al, 57
    jg   rt_stn_dot
    sub  al, 48
    movzx edx, al
    mov  qword ptr [rbp-88], 1
    jmp  rt_stn_dig
rt_stn_dot:
    cmp  al, 46              # '.'
    jne  rt_stn_dian
    mov  qword ptr [rbp-72], 1
    inc  r9
    jmp  rt_stn_loop
rt_stn_dian:
    cmp  byte ptr [rdx+r9], 0xE7
    jne  rt_stn_tab
    cmp  byte ptr [rdx+r9+1], 0x82
    jne  rt_stn_tab
    cmp  byte ptr [rdx+r9+2], 0xB9
    jne  rt_stn_tab
    mov  qword ptr [rbp-72], 1
    add  r9, 3
    jmp  rt_stn_loop
rt_stn_tab:
    movzx eax, byte ptr [rdx+r9]
    shl  eax, 16
    movzx ecx, byte ptr [rdx+r9+1]
    shl  ecx, 8
    or   eax, ecx
    movzx ecx, byte ptr [rdx+r9+2]
    or   eax, ecx
    lea  rcx, [rip+stn_tab]
rt_stn_look:
    mov  r10d, [rcx]
    test r10d, r10d
    jz   rt_stn_err
    cmp  r10d, eax
    je   rt_stn_found
    add  rcx, 8
    jmp  rt_stn_look
rt_stn_found:
    mov  r10d, [rcx+4]
    mov  edx, r10d
    shr  edx, 3             # 值
    and  r10d, 7            # 类
    cmp  r10d, 0
    je   rt_stn_dig
    cmp  r10d, 1
    je   rt_stn_units
    cmp  r10d, 2
    je   rt_stn_unitb
    cmp  r10d, 3
    je   rt_stn_ptk
    cmp  r10d, 4
    je   rt_stn_nian
    mov  edx, 3             # 卅：三
    mov  qword ptr [rbp-96], 10
    jmp  rt_stn_dig
rt_stn_nian:
    mov  edx, 2             # 廿：二
    mov  qword ptr [rbp-96], 10
    jmp  rt_stn_dig
rt_stn_ptk:
    mov  qword ptr [rbp-72], 1
    jmp  rt_stn_next
rt_stn_dig:
    cmp  qword ptr [rbp-72], 0
    jne  rt_stn_digf
    imul rcx, qword ptr [rbp-48], 10
    add  rcx, rdx
    mov  [rbp-48], rcx
    inc  qword ptr [rbp-80]
    jmp  rt_stn_afterd
rt_stn_digf:
    imul r11, r11, 10
    add  r11, rdx
    inc  qword ptr [rbp-64]
    inc  qword ptr [rbp-80]
    jmp  rt_stn_afterd
rt_stn_afterd:
    cmp  qword ptr [rbp-96], 0
    je   rt_stn_next
    mov  rdx, [rbp-96]
    mov  qword ptr [rbp-96], 0
    jmp  rt_stn_units
rt_stn_units:
    cmp  qword ptr [rbp-72], 0
    jne  rt_stn_err         # 小数之中遇单位，非数
    cmp  qword ptr [rbp-48], 0
    je   rt_stn_us1
    mov  rcx, [rbp-48]
    imul rcx, rdx
    add  [rbp-40], rcx
    mov  qword ptr [rbp-48], 0
    jmp  rt_stn_next
rt_stn_us1:
    add  [rbp-40], rdx
    inc  qword ptr [rbp-80]
    jmp  rt_stn_next
rt_stn_unitb:
    cmp  qword ptr [rbp-72], 0
    jne  rt_stn_err
    mov  rcx, [rbp-40]
    add  rcx, [rbp-48]
    test rcx, rcx
    jz   rt_stn_ub1
    imul rcx, rdx
    mov  [rbp-40], rcx
    jmp  rt_stn_ub2
rt_stn_ub1:
    mov  [rbp-40], rdx
rt_stn_ub2:
    mov  rax, [rbp-32]
    add  rax, [rbp-40]
    mov  [rbp-32], rax
    mov  qword ptr [rbp-40], 0
    mov  qword ptr [rbp-48], 0
    inc  qword ptr [rbp-80]
    jmp  rt_stn_next
rt_stn_next:
    add  r9, [rbp-88]
    jmp  rt_stn_loop
rt_stn_fin:
    cmp  qword ptr [rbp-80], 0
    je   rt_stn_err
    mov  rax, [rbp-32]
    add  rax, [rbp-40]
    add  rax, [rbp-48]
    cmp  qword ptr [rbp-72], 0
    je   rt_stn_int
    cvtsi2sd xmm0, rax
    cvtsi2sd xmm1, r11
    mov  rcx, [rbp-64]
rt_stn_p10:
    test rcx, rcx
    jz   rt_stn_pd
    mov  r10d, 10
    cvtsi2sd xmm2, r10
    divsd xmm1, xmm2
    dec  rcx
    jmp  rt_stn_p10
rt_stn_pd:
    addsd xmm0, xmm1
    test r8d, r8d
    jz   rt_stn_fp
    pxor xmm2, xmm2
    subsd xmm2, xmm0
    movsd xmm0, xmm2
rt_stn_fp:
    call rt_mkfloat
    jmp  rt_stn_ret
rt_stn_int:
    test r8d, r8d
    jz   rt_stn_ip
    neg  rax
rt_stn_ip:
    shl  rax, 3
rt_stn_ret:
    mov  rsp, rbp
    pop  rbp
    ret
rt_stn_err:
    lea  rax, [rip+err_num]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

# ---- 缀 ------------------------------------------------------------------
rt_concat:
    # 栈：[甲, 乙]（乙居顶）→ 化文相连
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    push rdi
    mov  rdi, rsi
    call rt_tostr
    pop  rdi
    push rax
    call rt_tostr
    pop  rdi
    mov  rsi, rax
    call rt_strcat
    sub  r15, 8
    mov  [r15], rax
    ret

# ---- 误由之所 ------------------------------------------------------------
rt_err_divzero_j:
    lea  rax, [rip+err_divzero]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_numpow_j:
    lea  rax, [rip+err_pow]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_neg_j:
    lea  rax, [rip+err_neg]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_num_j:
    lea  rax, [rip+err_num]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_sqrt_j:
    lea  rax, [rip+err_sqrt]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_badlist_j:
    lea  rax, [rip+err_list]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_rem_j:
    lea  rax, [rip+err_rem]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_err_cmp_j:
    lea  rax, [rip+err_cmp]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

# ---- 文之铸 --------------------------------------------------------------
rt_mkstr:
    # rdi = 字节之址，rsi = 长 → rax = 带标记之文
    push rbp
    mov  rbp, rsp
    sub  rsp, 16
    mov  [rbp-8], rdi
    mov  [rbp-16], rsi
    lea  rdi, [rsi+16]
    call rt_alloc
    mov  rcx, [rbp-16]
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    lea  rdi, [rax+16]
    mov  rsi, [rbp-8]
    xor  rdx, rdx
rt_mkstr_cp:
    cmp  rdx, rcx
    jge  rt_mkstr_done
    movzx r8d, byte ptr [rsi+rdx]
    mov  [rdi+rdx], r8b
    inc  rdx
    jmp  rt_mkstr_cp
rt_mkstr_done:
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret

# ---- 简牍（读简 · 书简） --------------------------------------------------
rt_readfile:
    # rdi = 带标记之径 → rax = 带标记之文（简牍全文；不得则劫）
    push rbp
    mov  rbp, rsp
    sub  rsp, 64
    mov  [rbp-8], rdi
    movzx eax, dil
    and  eax, 7
    cmp  eax, 4
    jne  rt_rf_errp
    mov  rax, rdi
    shr  rax, 3
    mov  rcx, [rax]
    mov  [rbp-16], rax
    mov  [rbp-24], rcx
    lea  rdi, [rcx+1]
    call rt_alloc
    mov  [rbp-32], rax
    mov  rsi, [rbp-16]
    lea  rsi, [rsi+16]
    xor  rdx, rdx
rt_rf_pc:
    cmp  rdx, [rbp-24]
    jge  rt_rf_pcd
    movzx r8d, byte ptr [rsi+rdx]
    mov  [rax+rdx], r8b
    inc  rdx
    jmp  rt_rf_pc
rt_rf_pcd:
    mov  byte ptr [rax+rdx], 0
    mov  rax, 2               # open
    mov  rdi, [rbp-32]
    xor  esi, esi             # O_RDONLY
    xor  edx, edx
    syscall
    test rax, rax
    js   rt_rf_err
    mov  [rbp-40], rax        # fd
    mov  edi, 160
    call rt_alloc
    mov  [rbp-48], rax
    mov  eax, 5               # fstat
    mov  rdi, [rbp-40]
    mov  rsi, [rbp-48]
    syscall
    test rax, rax
    js   rt_rf_cerr
    mov  rcx, [rbp-48]
    mov  r8, [rcx+48]         # st_size
    mov  [rbp-56], r8
    lea  rdi, [r8+16]
    call rt_alloc
    mov  [rbp-64], rax
    mov  rcx, [rbp-56]
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    xor  r9, r9
rt_rf_rl:
    cmp  r9, [rbp-56]
    jge  rt_rf_done
    xor  eax, eax             # read
    mov  rdi, [rbp-40]
    mov  rsi, [rbp-64]
    add  rsi, 16
    add  rsi, r9
    mov  rdx, [rbp-56]
    sub  rdx, r9
    syscall
    test rax, rax
    js   rt_rf_cerr
    jz   rt_rf_done
    add  r9, rax
    jmp  rt_rf_rl
rt_rf_done:
    mov  rax, 3               # close
    mov  rdi, [rbp-40]
    syscall
    mov  rcx, [rbp-64]
    mov  [rcx], r9
    mov  rax, rcx
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret
rt_rf_cerr:
    mov  rax, 3               # close
    mov  rdi, [rbp-40]
    syscall
rt_rf_err:
    lea  rax, [rip+err_read]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_rf_errp:
    lea  rax, [rip+err_path]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_writefile:
    # 栈：[实, 径]（径居顶）——实化文而书于简牍，覆其旧，无果
    push rbp
    mov  rbp, rsp
    sub  rsp, 64
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  [rbp-8], rdi         # 径（带标记）
    movzx eax, dil
    and  eax, 7
    cmp  eax, 4
    jne  rt_wf_errp
    mov  rdi, rsi
    call rt_tostr
    mov  rsi, rax
    shr  rsi, 3
    mov  rcx, [rsi]
    mov  [rbp-16], rsi        # 文（裸）
    mov  [rbp-24], rcx        # 文长
    mov  rax, [rbp-8]
    shr  rax, 3
    mov  rcx, [rax]
    mov  [rbp-32], rcx        # 径长
    lea  rdi, [rcx+1]
    call rt_alloc
    mov  [rbp-40], rax
    mov  r8, [rbp-8]
    shr  r8, 3
    lea  r8, [r8+16]
    xor  rdx, rdx
rt_wf_pc:
    cmp  rdx, [rbp-32]
    jge  rt_wf_pcd
    movzx r9d, byte ptr [r8+rdx]
    mov  [rax+rdx], r9b
    inc  rdx
    jmp  rt_wf_pc
rt_wf_pcd:
    mov  byte ptr [rax+rdx], 0
    mov  rax, 2               # open
    mov  rdi, [rbp-40]
    mov  esi, 0x241           # O_WRONLY|O_CREAT|O_TRUNC
    mov  edx, 420             # 0644
    syscall
    test rax, rax
    js   rt_wf_err
    mov  [rbp-48], rax        # fd
    xor  r9, r9
rt_wf_wl:
    cmp  r9, [rbp-24]
    jge  rt_wf_done
    mov  eax, 1               # write
    mov  rdi, [rbp-48]
    mov  rsi, [rbp-16]
    lea  rsi, [rsi+16]
    add  rsi, r9
    mov  rdx, [rbp-24]
    sub  rdx, r9
    syscall
    test rax, rax
    js   rt_wf_cerr
    jz   rt_wf_done
    add  r9, rax
    jmp  rt_wf_wl
rt_wf_done:
    mov  eax, 3               # close
    mov  rdi, [rbp-48]
    syscall
    mov  rsp, rbp
    pop  rbp
    ret
rt_wf_cerr:
    mov  eax, 3               # close
    mov  rdi, [rbp-48]
    syscall
rt_wf_err:
    lea  rax, [rip+err_write]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_wf_errp:
    lea  rax, [rip+err_path]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

# ---- 剖 ------------------------------------------------------------------
rt_split:
    # 栈：[文, 隔]（隔居顶）→ 列（空元不遗；隔空则劫）
    push rbp
    mov  rbp, rsp
    sub  rsp, 80
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  [rbp-8], rdi         # 隔（带标记）
    mov  [rbp-16], rsi        # 文（带标记）
    movzx eax, dil
    and  eax, 7
    cmp  eax, 4
    jne  rt_sp_errn
    movzx eax, sil
    and  eax, 7
    cmp  eax, 4
    jne  rt_sp_errn
    mov  rax, rdi
    shr  rax, 3
    mov  rcx, [rax]
    test rcx, rcx
    jz   rt_sp_err
    mov  [rbp-24], rcx        # 隔长
    lea  r8, [rax+16]
    mov  [rbp-32], r8         # 隔字节
    mov  rax, rsi
    shr  rax, 3
    mov  rcx, [rax]
    lea  r9, [rax+16]
    mov  [rbp-40], rcx        # 文长
    mov  [rbp-48], r9         # 文字节
    xor  r10, r10             # i
    xor  r11, r11             # 现
rt_sp_cnt:
    mov  rcx, [rbp-40]
    sub  rcx, [rbp-24]
    inc  rcx
    cmp  r10, rcx
    jge  rt_sp_cntd
    xor  rdx, rdx
rt_sp_cmp1:
    cmp  rdx, [rbp-24]
    jge  rt_sp_hit
    mov  rcx, [rbp-48]
    lea  rax, [rcx+r10]
    movzx eax, byte ptr [rax+rdx]
    mov  rcx, [rbp-32]
    cmp  al, [rcx+rdx]
    jne  rt_sp_nohit
    inc  rdx
    jmp  rt_sp_cmp1
rt_sp_hit:
    inc  r11
    add  r10, [rbp-24]
    jmp  rt_sp_cnt
rt_sp_nohit:
    inc  r10
    jmp  rt_sp_cnt
rt_sp_cntd:
    inc  r11
    mov  [rbp-56], r11        # 元数
    mov  rdi, r11
    shl  rdi, 3
    cmp  rdi, 16
    jge  rt_sp_a1
    mov  edi, 16
rt_sp_a1:
    call rt_alloc
    mov  [rbp-64], rax        # 阵
    mov  rdi, 32
    call rt_alloc
    mov  rcx, [rbp-56]
    mov  [rax], rcx
    mov  [rax+8], rcx
    mov  rcx, [rbp-64]
    mov  [rax+16], rcx
    mov  [rbp-72], rax        # 列
    xor  r10, r10             # i
    xor  r9, r9               # 元下标
    xor  r11, r11             # 元起
rt_sp_scan:
    mov  rcx, [rbp-40]
    sub  rcx, [rbp-24]
    inc  rcx
    cmp  r10, rcx
    jge  rt_sp_last
    xor  rdx, rdx
rt_sp_cmp2:
    cmp  rdx, [rbp-24]
    jge  rt_sp_take
    mov  rcx, [rbp-48]
    lea  rax, [rcx+r10]
    movzx eax, byte ptr [rax+rdx]
    mov  rcx, [rbp-32]
    cmp  al, [rcx+rdx]
    jne  rt_sp_step
    inc  rdx
    jmp  rt_sp_cmp2
rt_sp_take:
    mov  rsi, r10
    sub  rsi, r11             # 元长
    mov  rdi, [rbp-48]
    add  rdi, r11             # 元之字节
    call rt_mkstr
    mov  rcx, [rbp-64]
    mov  [rcx+r9*8], rax
    inc  r9
    add  r10, [rbp-24]
    mov  r11, r10
    jmp  rt_sp_scan
rt_sp_step:
    inc  r10
    jmp  rt_sp_scan
rt_sp_last:
    mov  rsi, [rbp-40]
    sub  rsi, r11             # 末元之长
    mov  rdi, [rbp-48]
    add  rdi, r11             # 末元之字节
    call rt_mkstr
    mov  rcx, [rbp-64]
    mov  [rcx+r9*8], rax
    mov  rax, [rbp-72]
    shl  rax, 3
    or   rax, 5
    sub  r15, 8
    mov  [r15], rax
    mov  rsp, rbp
    pop  rbp
    ret
rt_sp_err:
    lea  rax, [rip+err_split]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail
rt_sp_errn:
    lea  rax, [rip+err_splitn]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

# ---- 典之诸务 ------------------------------------------------------------
# 典：[计, 掩, 桶阵]；桶阵为链之目：[哈希, 键, 实, 次]（三十二字节）
rt_mkdict:
    push rbp
    mov  rbp, rsp
    sub  rsp, 16
    mov  edi, 32
    call rt_alloc
    mov  [rbp-8], rax
    mov  qword ptr [rax], 0
    mov  qword ptr [rax+8], 7
    mov  edi, 64
    call rt_alloc
    mov  rcx, [rbp-8]
    mov  [rcx+16], rax
    xor  rdx, rdx
rt_mkdict_z:
    mov  qword ptr [rax+rdx*8], 0
    inc  rdx
    cmp  rdx, 8
    jl   rt_mkdict_z
    mov  rax, rcx
    shl  rax, 3
    or   rax, 6
    mov  rsp, rbp
    pop  rbp
    ret

rt_dict_hash:
    # rdi = 键（带标记）→ rax = 哈希（既 avalanche）
    mov  rax, rdi
    and  eax, 7
    jz   rt_dh_int
    cmp  eax, 1
    je   rt_dh_nil
    cmp  eax, 2
    je   rt_dh_tag
    cmp  eax, 3
    je   rt_dh_tag
    cmp  eax, 4
    je   rt_dh_str
    cmp  eax, 7
    je   rt_dh_flt
    mov  rax, rdi             # 列/典：以指为质
    jmp  rt_dh_mix
rt_dh_nil:
    mov  rax, 1
    jmp  rt_dh_mix
rt_dh_tag:
    mov  rax, rdi
    and  rax, 7
    jmp  rt_dh_mix
rt_dh_int:
    mov  rax, rdi
    sar  rax, 3
    jmp  rt_dh_mix
rt_dh_flt:
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    cvttsd2si rcx, xmm0
    cvtsi2sd xmm1, rcx
    ucomisd xmm0, xmm1
    jne  rt_dh_fltb
    jp   rt_dh_fltb
    mov  rax, rcx             # 整值之浮，与整同哈希
    jmp  rt_dh_mix
rt_dh_fltb:
    mov  rax, rdi
    shr  rax, 3
    mov  rax, [rax]
    jmp  rt_dh_mix
rt_dh_str:
    mov  rax, rdi
    shr  rax, 3
    mov  rcx, [rax]
    lea  rax, [rax+16]
    movabs rdx, 0xcbf29ce484222325
rt_dh_sloop:
    test rcx, rcx
    jz   rt_dh_mixs
    movzx r8d, byte ptr [rax]
    xor  rdx, r8
    movabs r9, 0x100000001b3
    imul rdx, r9
    inc  rax
    dec  rcx
    jmp  rt_dh_sloop
rt_dh_mixs:
    mov  rax, rdx
rt_dh_mix:
    mov  r10, rax
    shr  r10, 33
    xor  rax, r10
    movabs r10, 0xff51afd7ed558ccd
    imul rax, r10
    mov  r10, rax
    shr  r10, 33
    xor  rax, r10
    movabs r10, 0xc4ceb9fe1a85ec53
    imul rax, r10
    mov  r10, rax
    shr  r10, 33
    xor  rax, r10
    ret

rt_dict_keyeq:
    # rdi, rsi = 键（带标记）→ eax = 一/〇（数整数浮以值相映，文比其字节，列典以指为质）
    movzx r8d, dil
    and  r8d, 7
    movzx r9d, sil
    and  r9d, 7
    mov  eax, r8d
    or   eax, r9d
    cmp  eax, 7               # 整浮相杂（〇|七＝七）
    jne  rt_dke_tag
    cmp  r8d, 7
    je   rt_dke_n1
    mov  rax, rdi
    sar  rax, 3
    cvtsi2sd xmm0, rax
    jmp  rt_dke_n2
rt_dke_n1:
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
rt_dke_n2:
    cmp  r9d, 7
    je   rt_dke_n3
    mov  rax, rsi
    sar  rax, 3
    cvtsi2sd xmm1, rax
    jmp  rt_dke_n4
rt_dke_n3:
    mov  rax, rsi
    shr  rax, 3
    movsd xmm1, qword ptr [rax]
rt_dke_n4:
    ucomisd xmm0, xmm1
    jne  rt_dke_no
    jp   rt_dke_no
    jmp  rt_dke_yes
rt_dke_tag:
    cmp  r8d, r9d
    jne  rt_dke_no
    cmp  rdi, rsi
    je   rt_dke_yes
    cmp  r8d, 4
    je   rt_dke_str
    cmp  r8d, 7
    je   rt_dke_fltf
    jmp  rt_dke_no
rt_dke_fltf:
    mov  rax, rdi
    shr  rax, 3
    movsd xmm0, qword ptr [rax]
    mov  rax, rsi
    shr  rax, 3
    movsd xmm1, qword ptr [rax]
    ucomisd xmm0, xmm1
    jne  rt_dke_no
    jp   rt_dke_no
    jmp  rt_dke_yes
rt_dke_str:
    mov  rax, rdi
    shr  rax, 3
    mov  rcx, [rax]
    mov  r9, rax
    mov  rax, rsi
    shr  rax, 3
    cmp  rcx, [rax]
    jne  rt_dke_no
    lea  r9, [r9+16]
    lea  rax, [rax+16]
    xor  rdx, rdx
rt_dke_sc:
    cmp  rdx, rcx
    jge  rt_dke_yes
    movzx r8d, byte ptr [r9+rdx]
    cmp  r8b, [rax+rdx]
    jne  rt_dke_no
    inc  rdx
    jmp  rt_dke_sc
rt_dke_yes:
    mov  eax, 1
    ret
rt_dke_no:
    xor  eax, eax
    ret

rt_dict_find:
    # rdi = 典（裸），rsi = 键（带标记），rdx = 全哈希
    # → rax = 目（〇若无），rdx = 桶下标
    push rbp
    mov  rbp, rsp
    sub  rsp, 40
    mov  [rbp-8], rsi
    mov  [rbp-16], rdx
    mov  r8, rdi
    and  rdx, [r8+8]
    mov  [rbp-24], rdx
    mov  rcx, [r8+16]
    mov  rax, [rcx+rdx*8]
rt_df_loop:
    test rax, rax
    jz   rt_df_done
    mov  rcx, [rbp-16]
    cmp  [rax], rcx
    jne  rt_df_next
    mov  [rbp-32], rax
    mov  rdi, [rax+8]
    mov  rsi, [rbp-8]
    call rt_dict_keyeq
    test eax, eax
    jz   rt_df_next0
    mov  rax, [rbp-32]
    jmp  rt_df_done
rt_df_next0:
    mov  rax, [rbp-32]
rt_df_next:
    mov  rax, [rax+24]
    jmp  rt_df_loop
rt_df_done:
    mov  rdx, [rbp-24]
    mov  rsp, rbp
    pop  rbp
    ret

rt_dict_rehash:
    # rdi = 典（裸）——桶倍其数，诸目重链
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  [rbp-8], rdi
    mov  rcx, [rdi+8]
    inc  rcx
    add  rcx, rcx
    mov  rdx, rcx
    dec  rdx
    mov  [rbp-16], rdx        # 新掩
    mov  rdi, rcx
    shl  rdi, 3
    call rt_alloc
    mov  [rbp-24], rax
    xor  rdx, rdx
rt_dr_z:
    mov  qword ptr [rax+rdx*8], 0
    inc  rdx
    mov  rcx, [rbp-16]
    inc  rcx
    cmp  rdx, rcx
    jl   rt_dr_z
    mov  r8, [rbp-8]
    mov  r9, [r8+16]          # 旧桶阵
    xor  r10, r10
rt_dr_b:
    mov  rcx, [rbp-8]
    mov  rcx, [rcx+8]
    inc  rcx
    cmp  r10, rcx
    jge  rt_dr_done
    mov  r11, [r9+r10*8]
rt_dr_c:
    test r11, r11
    jz   rt_dr_nb
    mov  rax, [r11+24]
    push rax
    mov  rax, [r11]
    and  rax, [rbp-16]
    mov  rcx, [rbp-24]
    mov  rdx, [rcx+rax*8]
    mov  [r11+24], rdx
    mov  [rcx+rax*8], r11
    pop  r11
    jmp  rt_dr_c
rt_dr_nb:
    inc  r10
    jmp  rt_dr_b
rt_dr_done:
    mov  rcx, [rbp-8]
    mov  rax, [rbp-16]
    mov  [rcx+8], rax
    mov  rax, [rbp-24]
    mov  [rcx+16], rax
    mov  rsp, rbp
    pop  rbp
    ret

rt_dictset:
    # 栈：[实, 键, 典]（典居顶）——铭实于键，有则代之
    push rbp
    mov  rbp, rsp
    sub  rsp, 48
    mov  rdi, [r15]
    add  r15, 8
    mov  rsi, [r15]
    add  r15, 8
    mov  rax, [r15]
    add  r15, 8
    mov  [rbp-8], rax         # 实
    mov  [rbp-16], rsi        # 键
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_ds_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-24], rax
    mov  rdi, rsi
    call rt_dict_hash
    mov  rdi, [rbp-24]
    mov  rsi, [rbp-16]
    mov  rdx, rax
    mov  [rbp-32], rax
    call rt_dict_find
    mov  [rbp-40], rdx
    test rax, rax
    jz   rt_ds_ins
    mov  rcx, [rbp-8]
    mov  [rax+16], rcx
    jmp  rt_ds_ret
rt_ds_ins:
    mov  rdi, 32
    call rt_alloc
    mov  rcx, [rbp-32]
    mov  [rax], rcx
    mov  rcx, [rbp-16]
    mov  [rax+8], rcx
    mov  rcx, [rbp-8]
    mov  [rax+16], rcx
    mov  r8, [rbp-24]
    mov  r9, [r8+16]
    mov  rcx, [rbp-40]
    mov  rdx, [r9+rcx*8]
    mov  [rax+24], rdx
    mov  [r9+rcx*8], rax
    inc  qword ptr [r8]
    mov  rcx, [r8]
    mov  rdx, [r8+8]
    inc  rdx
    add  rdx, rdx
    cmp  rcx, rdx
    jl   rt_ds_ret
    mov  rdi, r8
    call rt_dict_rehash
rt_ds_ret:
    mov  rsp, rbp
    pop  rbp
    ret
rt_ds_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dictget:
    # 栈：[典, 键]（键居顶）→ 实；无键则虚
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  rsi, [r15]
    add  r15, 8
    mov  rdi, [r15]
    add  r15, 8
    mov  [rbp-8], rsi
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_dg_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-16], rax
    mov  rdi, rsi
    call rt_dict_hash
    mov  rdi, [rbp-16]
    mov  rsi, [rbp-8]
    mov  rdx, rax
    call rt_dict_find
    test rax, rax
    jz   rt_dg_no
    mov  rax, [rax+16]
    jmp  rt_dg_push
rt_dg_no:
    mov  eax, 1
rt_dg_push:
    sub  r15, 8
    mov  [r15], rax
    mov  rsp, rbp
    pop  rbp
    ret
rt_dg_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dicthas:
    # 栈：[典, 键]（键居顶）→ 阳/阴
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    mov  rsi, [r15]
    add  r15, 8
    mov  rdi, [r15]
    add  r15, 8
    mov  [rbp-8], rsi
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_dha_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-16], rax
    mov  rdi, rsi
    call rt_dict_hash
    mov  rdi, [rbp-16]
    mov  rsi, [rbp-8]
    mov  rdx, rax
    call rt_dict_find
    test rax, rax
    jz   rt_dha_no
    mov  eax, 3
    jmp  rt_dha_push
rt_dha_no:
    mov  eax, 2
rt_dha_push:
    sub  r15, 8
    mov  [r15], rax
    mov  rsp, rbp
    pop  rbp
    ret
rt_dha_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dictdel:
    # 栈：[典, 键]（键居顶）——除其寓，无键则默然
    push rbp
    mov  rbp, rsp
    sub  rsp, 48
    mov  rsi, [r15]
    add  r15, 8
    mov  rdi, [r15]
    add  r15, 8
    mov  [rbp-8], rsi
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_dd_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-16], rax
    mov  rdi, rsi
    call rt_dict_hash
    mov  [rbp-24], rax
    mov  r8, [rbp-16]
    mov  rcx, rax
    and  rcx, [r8+8]
    mov  [rbp-32], rcx
    mov  r9, [r8+16]
    mov  rax, [r9+rcx*8]
    xor  r10, r10             # 前目
rt_dd_loop:
    test rax, rax
    jz   rt_dd_ret
    mov  rcx, [rbp-24]
    cmp  [rax], rcx
    jne  rt_dd_next
    mov  [rbp-40], rax
    mov  rdi, [rax+8]
    mov  rsi, [rbp-8]
    call rt_dict_keyeq
    test eax, eax
    jz   rt_dd_next0
    mov  rax, [rbp-40]
    jmp  rt_dd_unlink
rt_dd_next0:
    mov  rax, [rbp-40]
rt_dd_next:
    mov  r10, rax
    mov  rax, [rax+24]
    jmp  rt_dd_loop
rt_dd_unlink:
    mov  rcx, [rax+24]
    test r10, r10
    jz   rt_dd_head
    mov  [r10+24], rcx
    jmp  rt_dd_dec
rt_dd_head:
    mov  r8, [rbp-16]
    mov  r9, [r8+16]
    mov  rdx, [rbp-32]
    mov  [r9+rdx*8], rcx
rt_dd_dec:
    mov  r8, [rbp-16]
    dec  qword ptr [r8]
rt_dd_ret:
    mov  rsp, rbp
    pop  rbp
    ret
rt_dd_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dictkeys:
    # rdi = 带标记之典 → rax = 键之列（快照，序无定）
    push rbp
    mov  rbp, rsp
    sub  rsp, 32
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_dks_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    mov  r8, [rax]
    mov  [rbp-16], r8
    mov  rdi, r8
    shl  rdi, 3
    cmp  rdi, 16
    jge  rt_dks_a1
    mov  edi, 16
rt_dks_a1:
    call rt_alloc
    mov  [rbp-24], rax
    mov  rdi, 32
    call rt_alloc
    mov  rcx, [rbp-16]
    mov  [rax], rcx
    mov  [rax+8], rcx
    mov  rcx, [rbp-24]
    mov  [rax+16], rcx
    mov  [rbp-32], rax
    xor  r9, r9
    xor  r10, r10
rt_dks_b:
    mov  rcx, [rbp-8]
    mov  rcx, [rcx+8]
    inc  rcx
    cmp  r10, rcx
    jge  rt_dks_done
    mov  r8, [rbp-8]
    mov  r8, [r8+16]
    mov  r11, [r8+r10*8]
rt_dks_c:
    test r11, r11
    jz   rt_dks_nb
    mov  rax, [rbp-24]
    mov  rdx, [r11+8]
    mov  [rax+r9*8], rdx
    inc  r9
    mov  r11, [r11+24]
    jmp  rt_dks_c
rt_dks_nb:
    inc  r10
    jmp  rt_dks_b
rt_dks_done:
    mov  rax, [rbp-32]
    shl  rax, 3
    or   rax, 5
    mov  rsp, rbp
    pop  rbp
    ret
rt_dks_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dicttostr:
    # rdi = 带标记之典 → rax = 带标记之文：〔键曰实，键曰实〕
    push rbp
    mov  rbp, rsp
    sub  rsp, 8192
    movzx eax, dil
    and  eax, 7
    cmp  eax, 6
    jne  rt_dts_err
    mov  rax, rdi
    shr  rax, 3
    mov  [rbp-8], rax
    lea  r10, [rbp-8192]
    mov  [rbp-16], r10
    mov  byte ptr [r10], 0xE3
    mov  byte ptr [r10+1], 0x80
    mov  byte ptr [r10+2], 0x94
    mov  rcx, 3
    mov  [rbp-24], rcx         # 写位
    mov  qword ptr [rbp-32], 0 # 当前目
    mov  qword ptr [rbp-40], 0 # 桶下标
    mov  qword ptr [rbp-48], 0 # 已书对数
rt_dts_b:
    mov  rax, [rbp-8]
    mov  rcx, [rax+8]
    inc  rcx
    cmp  [rbp-40], rcx
    jge  rt_dts_close
    mov  rax, [rbp-8]
    mov  r8, [rax+16]
    mov  rcx, [rbp-40]
    mov  rax, [r8+rcx*8]
    mov  [rbp-32], rax
rt_dts_c:
    mov  rax, [rbp-32]
    test rax, rax
    jz   rt_dts_nb
    cmp  qword ptr [rbp-48], 0
    je   rt_dts_pair
    cmp  qword ptr [rbp-24], 8000
    jae  rt_dts_skip
    mov  rcx, [rbp-24]
    mov  r10, [rbp-16]
    mov  byte ptr [r10+rcx], 0xEF
    mov  byte ptr [r10+rcx+1], 0xBC
    mov  byte ptr [r10+rcx+2], 0x8C
    add  rcx, 3
    mov  [rbp-24], rcx
rt_dts_pair:
    inc  qword ptr [rbp-48]
    mov  rcx, [rbp-24]
    mov  rax, [rbp-32]
    mov  rsi, [rax+8]
    call rt_dts_elem
    mov  [rbp-24], rcx
    cmp  qword ptr [rbp-24], 8000
    jae  rt_dts_skip
    mov  rcx, [rbp-24]
    mov  r10, [rbp-16]
    mov  byte ptr [r10+rcx], 0xE6
    mov  byte ptr [r10+rcx+1], 0x9B
    mov  byte ptr [r10+rcx+2], 0xB0
    add  rcx, 3
    mov  [rbp-24], rcx
    mov  rcx, [rbp-24]
    mov  rax, [rbp-32]
    mov  rsi, [rax+16]
    call rt_dts_elem
    mov  [rbp-24], rcx
rt_dts_skip:
    mov  rax, [rbp-32]
    mov  rax, [rax+24]
    mov  [rbp-32], rax
    jmp  rt_dts_c
rt_dts_nb:
    inc  qword ptr [rbp-40]
    jmp  rt_dts_b
rt_dts_close:
    mov  rcx, [rbp-24]
    mov  r10, [rbp-16]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x95
    add  rcx, 3
    mov  [rbp-24], rcx
    mov  rdi, rcx
    add  rdi, 16
    call rt_alloc
    mov  [rbp-56], rax
    mov  rcx, [rbp-24]
    mov  [rax], rcx
    mov  qword ptr [rax+8], 0
    lea  rdi, [rax+16]
    mov  r10, [rbp-16]
    xor  rdx, rdx
rt_dts_cp:
    cmp  rdx, rcx
    jge  rt_dts_done
    movzx r8d, byte ptr [r10+rdx]
    mov  [rdi+rdx], r8b
    inc  rdx
    jmp  rt_dts_cp
rt_dts_done:
    mov  rax, [rbp-56]
    shl  rax, 3
    or   rax, 4
    mov  rsp, rbp
    pop  rbp
    ret
rt_dts_err:
    lea  rax, [rip+err_dict]
    shl  rax, 3
    or   rax, 4
    jmp  rt_fail

rt_dts_elem:
    # rsi = 实（带标记），r10 = 缓冲基，rcx = 写位 → rcx 前进而返（实化文而书，文则被「」）
    sub  rsp, 24
    mov  [rsp], rcx
    mov  [rsp+8], r10
    mov  qword ptr [rsp+16], 0
    mov  rdi, rsi
    movzx eax, sil
    and  eax, 7
    cmp  eax, 4
    jne  rt_dts_el_go
    mov  qword ptr [rsp+16], 1
    mov  r10, [rsp+8]
    mov  rcx, [rsp]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x8C
    add  rcx, 3
    mov  [rsp], rcx
rt_dts_el_go:
    mov  rdi, rsi
    call rt_tostr
    mov  rsi, rax
    mov  r10, [rsp+8]
    mov  rcx, [rsp]
    call rt_dts_el_cp
    mov  [rsp], rcx
    cmp  qword ptr [rsp+16], 0
    je   rt_dts_el_ret
    mov  r10, [rsp+8]
    mov  rcx, [rsp]
    mov  byte ptr [r10+rcx], 0xE3
    mov  byte ptr [r10+rcx+1], 0x80
    mov  byte ptr [r10+rcx+2], 0x8D
    add  rcx, 3
    mov  [rsp], rcx
rt_dts_el_ret:
    mov  rcx, [rsp]
    add  rsp, 24
    ret

rt_dts_el_cp:
    # rsi = 带标记之文，r10 = 缓冲基，rcx = 写位 → 前进（逾八千则止）
    mov  rax, rsi
    shr  rax, 3
    mov  r8, [rax]
    lea  rax, [rax+16]
    xor  rdx, rdx
rt_dts_el_cpl:
    cmp  rdx, r8
    jge  rt_dts_el_cpd
    cmp  rcx, 8000
    jae  rt_dts_el_cpd
    movzx edi, byte ptr [rax+rdx]
    mov  [r10+rcx], dil
    inc  rcx
    inc  rdx
    jmp  rt_dts_el_cpl
rt_dts_el_cpd:
    ret
`
