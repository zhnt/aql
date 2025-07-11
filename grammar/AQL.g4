grammar AQL;

// ===== 程序入口 =====
program: statement* EOF;

// ===== 语句 =====
statement
    : functionDecl
    | agentDecl
    | variableDecl
    | assignment
    | expressionStmt
    | controlFlow
    | importStmt
    ;

// ===== 导入语句 =====
importStmt
    : IMPORT STRING (AS ID)?                                    # ImportModule
    | IMPORT '{' ID (',' ID)* '}' FROM STRING                  # ImportSelective
    ;

// ===== 函数声明 =====
functionDecl
    : ASYNC? FUNCTION ID '(' paramList? ')' block
    ;

paramList
    : ID (',' ID)*
    ;

// ===== Agent声明 (AQL特有) =====
agentDecl
    : AGENT ID '{' agentProperty* '}'
    ;

agentProperty
    : ID ':' expression ','?
    ;

// ===== 变量声明 =====
variableDecl
    : LOCAL ID ('=' expression)?
    ;

// ===== 赋值 =====
assignment
    : ID '=' expression
    ;

// ===== 表达式语句 =====
expressionStmt
    : expression
    ;

// ===== 控制流 =====
controlFlow
    : ifStatement
    | forStatement
    | whileStatement
    | matchStatement
    | block
    ;

// ===== 代码块 =====
block
    : DO statement* END
    | '{' statement* '}'
    ;

// ===== If语句 =====
ifStatement
    : IF expression THEN statement* (ELSEIF expression THEN statement*)* (ELSE statement*)? END
    ;

// ===== 循环语句 =====
forStatement
    : FOR PARALLEL? ID (',' ID)? IN expression DO statement* END    # ForInLoop
    | FOR AWAIT ID IN expression DO statement* END                  # ForAwaitLoop
    | FOR ID '=' expression ',' expression (',' expression)? DO statement* END  # ForNumericLoop
    ;

// ===== While循环 =====
whileStatement
    : WHILE expression DO statement* END
    ;

// ===== Match语句 =====
matchStatement
    : MATCH expression matchCase* (ELSE statement*)? END
    ;

matchCase
    : CASE expression THEN statement*
    ;

// ===== 表达式 =====
expression
    : primary                                    # PrimaryExpr
    | expression '.' ID                          # MemberAccess
    | expression '[' expression ']'             # IndexAccess
    | expression '(' argumentList? ')'          # FunctionCall
    | AWAIT expression                          # AwaitExpr
    | unaryOp expression                        # UnaryExpr
    | expression binaryOp expression            # BinaryExpr
    | expression ARROW expression               # FlowExpr
    | expression SPLIT expression               # SplitExpr
    | expression MERGE expression               # MergeExpr
    | expression PARALLEL_OP expression         # ParallelExpr
    ;

// ===== 参数列表 =====
argumentList
    : expression (',' expression)*
    ;

// ===== 一元操作符 =====
unaryOp
    : '!'
    | '-'
    | '+'
    ;

// ===== 二元操作符 =====
binaryOp
    : '*' | '/' | '%'                           # MultiplicativeOp
    | '+' | '-'                                 # AdditiveOp
    | '<' | '<=' | '>' | '>=' | '==' | '!='    # ComparisonOp
    | '&&'                                      # AndOp
    | '||'                                      # OrOp
    ;

// ===== 基础表达式 =====
primary
    : ID                                        # Identifier
    | NUMBER                                    # Number
    | STRING                                    # String
    | BOOLEAN                                   # Boolean
    | NIL                                       # Nil
    | arrayLiteral                              # Array
    | objectLiteral                             # Object
    | '(' expression ')'                        # Parenthesized
    ;

// ===== 数组字面量 =====
arrayLiteral
    : '[' (expression (',' expression)*)? ']'
    ;

// ===== 对象字面量 =====
objectLiteral
    : '{' (objectProperty (',' objectProperty)*)? '}'
    ;

objectProperty
    : (ID | STRING) ':' expression
    ;

// ===== 词法规则 =====

// 关键字
AGENT: 'agent';
ASYNC: 'async';
AWAIT: 'await';
FUNCTION: 'function';
LOCAL: 'local';
FOR: 'for';
PARALLEL: 'parallel';
IN: 'in';
DO: 'do';
END: 'end';
IF: 'if';
THEN: 'then';
ELSEIF: 'elseif';
ELSE: 'else';
WHILE: 'while';
MATCH: 'match';
CASE: 'case';
IMPORT: 'import';
FROM: 'from';
AS: 'as';

// AQL特有操作符
ARROW: '-->';
SPLIT: '-<';
MERGE: '->';
PARALLEL_OP: '-|';
QUANTITY: '-@' [0-9]+ | '-@*';
CONTROL: '-#' ('wait'|'auto');

// 基础常量
BOOLEAN: 'true' | 'false';
NIL: 'nil';

// 标识符和字面量
ID: [a-zA-Z_][a-zA-Z0-9_]*;
NUMBER: [0-9]+ ('.' [0-9]+)? ([eE] [+-]? [0-9]+)?;
STRING: '"' (~["\\\r\n] | '\\' .)* '"' | '\'' (~['\\\r\n] | '\\' .)* '\'';

// 注释和空白
WS: [ \t\r\n]+ -> skip;
LINE_COMMENT: '--' ~[\r\n]* -> skip;
BLOCK_COMMENT: '--[[' .*? ']]--' -> skip; 