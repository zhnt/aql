package lexer1

import (
	"fmt"
	"strings"
)

// DemoBasicTokens 演示基础token识别
func DemoBasicTokens() {
	fmt.Println("=== AQL词法分析器 - 基础Token演示 ===")

	input := `let x = 42;
let name = "Hello AQL";
let pi = 3.14;`

	fmt.Printf("输入代码:\n%s\n\n", input)
	fmt.Println("词法分析结果:")

	l := New(input)
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			fmt.Printf("%-12s %-15s Line:%d Col:%d\n", "EOF", `""`, tok.Line, tok.Column)
			break
		}
		fmt.Printf("%-12s %-15s Line:%d Col:%d\n", tok.Type, fmt.Sprintf(`"%s"`, tok.Literal), tok.Line, tok.Column)
	}
}

// DemoAsyncFeatures 演示异步特性
func DemoAsyncFeatures() {
	fmt.Println("\n=== AQL词法分析器 - 异步特性演示 ===")

	input := `async function fetchData() {
    let result = await @api.getData();
    yield result;
}`

	fmt.Printf("输入代码:\n%s\n\n", input)
	fmt.Println("词法分析结果:")

	l := New(input)
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			fmt.Printf("%-12s %-15s Line:%d Col:%d\n", "EOF", `""`, tok.Line, tok.Column)
			break
		}
		fmt.Printf("%-12s %-15s Line:%d Col:%d\n", tok.Type, fmt.Sprintf(`"%s"`, tok.Literal), tok.Line, tok.Column)
	}
}

// DemoAIServiceSyntax 演示AI服务语法
func DemoAIServiceSyntax() {
	fmt.Println("\n=== AQL词法分析器 - AI服务语法演示 ===")

	input := `let response = await @openai.chat("Hello");
data |> @preprocess.clean() |> @ai.analyze();`

	fmt.Printf("输入代码:\n%s\n\n", input)
	fmt.Println("词法分析结果:")

	l := New(input)
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			fmt.Printf("%-12s %-15s Line:%d Col:%d\n", "EOF", `""`, tok.Line, tok.Column)
			break
		}
		fmt.Printf("%-12s %-15s Line:%d Col:%d\n", tok.Type, fmt.Sprintf(`"%s"`, tok.Literal), tok.Line, tok.Column)
	}
}

// DemoComplexProgram 演示复杂程序
func DemoComplexProgram() {
	fmt.Println("\n=== AQL词法分析器 - 复杂程序演示 ===")

	input := `// AQL AI服务编排程序
import { openai } from "@ai/openai";

async function processText(text: string) {
    try {
        // 数据预处理管道
        let cleaned = text |> @preprocess.clean() |> @preprocess.normalize();
        
        // AI分析
        let analysis = await @openai.chat({
            model: "gpt-4",
            prompt: cleaned,
            temperature: 0.7
        });
        
        // 后处理
        return analysis |> @postprocess.format();
    } catch (error) {
        throw new Error("Processing failed: " + error.message);
    }
}

export { processText };`

	fmt.Printf("输入代码:\n%s\n\n", input)
	fmt.Println("词法分析结果:")

	l := New(input)
	tokenCount := 0
	for {
		tok := l.NextToken()
		tokenCount++
		if tok.Type == EOF {
			fmt.Printf("%3d: %-12s %-20s Line:%d Col:%d\n", tokenCount, "EOF", `""`, tok.Line, tok.Column)
			break
		}

		// 突出显示重要的token类型
		tokenDisplay := tok.Type.String()
		if tok.Type == ASYNC || tok.Type == AWAIT || tok.Type == AT_SYMBOL || tok.Type == PIPE {
			tokenDisplay = "🔥" + tokenDisplay
		}

		literal := tok.Literal
		if len(literal) > 15 {
			literal = literal[:12] + "..."
		}

		fmt.Printf("%3d: %-15s %-20s Line:%d Col:%d\n",
			tokenCount, tokenDisplay, fmt.Sprintf(`"%s"`, literal), tok.Line, tok.Column)
	}

	fmt.Printf("\n总计: %d个token\n", tokenCount)
}

// DemoErrorHandling 演示错误处理
func DemoErrorHandling() {
	fmt.Println("\n=== AQL词法分析器 - 错误处理演示 ===")

	input := `let x = 5;
& | # @ valid_identifier`

	fmt.Printf("输入代码:\n%s\n\n", input)
	fmt.Println("词法分析结果:")

	l := New(input)
	illegalCount := 0
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			fmt.Printf("%-12s %-15s Line:%d Col:%d\n", "EOF", `""`, tok.Line, tok.Column)
			break
		}

		if tok.Type == ILLEGAL {
			illegalCount++
			fmt.Printf("❌%-11s %-15s Line:%d Col:%d\n", tok.Type, fmt.Sprintf(`"%s"`, tok.Literal), tok.Line, tok.Column)
		} else {
			fmt.Printf("✅%-11s %-15s Line:%d Col:%d\n", tok.Type, fmt.Sprintf(`"%s"`, tok.Literal), tok.Line, tok.Column)
		}
	}

	fmt.Printf("\n发现 %d 个非法token\n", illegalCount)
}

// DemoTokenStats 演示token统计
func DemoTokenStats() {
	fmt.Println("\n=== AQL词法分析器 - Token统计演示 ===")

	input := `async function analyzeData(input: string) {
    let processed = input |> @preprocess.clean();
    let result = await @ai.analyze(processed);
    return result;
}`

	fmt.Printf("输入代码:\n%s\n\n", input)

	l := New(input)
	tokenStats := make(map[TokenType]int)
	totalTokens := 0

	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			break
		}
		tokenStats[tok.Type]++
		totalTokens++
	}

	fmt.Println("Token统计:")
	fmt.Printf("%-15s %s\n", "Token类型", "数量")
	fmt.Println(strings.Repeat("-", 25))

	for tokenType, count := range tokenStats {
		fmt.Printf("%-15s %d\n", tokenType, count)
	}

	fmt.Printf("\n总计: %d个token\n", totalTokens)
}

// RunAllDemos 运行所有演示
func RunAllDemos() {
	DemoBasicTokens()
	DemoAsyncFeatures()
	DemoAIServiceSyntax()
	DemoComplexProgram()
	DemoErrorHandling()
	DemoTokenStats()

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("🎉 AQL词法分析器MVP版本已完成!")
	fmt.Println("✅ 支持基础语法token")
	fmt.Println("✅ 支持异步关键字(async/await/yield)")
	fmt.Println("✅ 支持AI服务语法(@symbol, |>)")
	fmt.Println("✅ 支持注释和错误处理")
	fmt.Println("✅ 准确的行号和列号跟踪")
	fmt.Println("\n下一步: 实现语法分析器(Parser)")
}
