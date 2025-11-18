package mcpclient

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CollectText 从 content 中提取所有文本块
func CollectText(content []mcp.Content) []string {
	if len(content) == 0 {
		return nil
	}
	var texts []string
	for _, item := range content {
		if block, ok := item.(*mcp.TextContent); ok && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return texts
}

// TextContent 将内容拼接成一个文本字符串，便于日志或报错输出
func TextContent(content []mcp.Content) string {
	return strings.Join(CollectText(content), "\n")
}
