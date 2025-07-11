package lexer1

import (
	"testing"
)

// TestLexerDemo 运行词法分析器演示
func TestLexerDemo(t *testing.T) {
	t.Log("运行AQL词法分析器完整演示...")
	RunAllDemos()
}

// TestIndividualDemos 单独测试各个演示功能
func TestIndividualDemos(t *testing.T) {
	t.Run("BasicTokens", func(t *testing.T) {
		DemoBasicTokens()
	})

	t.Run("AsyncFeatures", func(t *testing.T) {
		DemoAsyncFeatures()
	})

	t.Run("AIServiceSyntax", func(t *testing.T) {
		DemoAIServiceSyntax()
	})

	t.Run("ComplexProgram", func(t *testing.T) {
		DemoComplexProgram()
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		DemoErrorHandling()
	})

	t.Run("TokenStats", func(t *testing.T) {
		DemoTokenStats()
	})
}
