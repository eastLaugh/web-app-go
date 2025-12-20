package document

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Chunk 文档块
type Chunk struct {
	File      string
	ChunkIndex int
	Content   string
	Title     string
}

// ProcessMarkdown 处理 Markdown 文件，按段落切分
func ProcessMarkdown(content string, file string) []Chunk {
	// 提取标题（第一行 # 开头的）
	lines := strings.Split(content, "\n")
	title := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	// 按双换行符切分段落
	paragraphs := strings.Split(content, "\n\n")
	chunks := make([]Chunk, 0)

	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 跳过标题行
		if strings.HasPrefix(para, "#") {
			continue
		}

		// 如果段落太长（超过 1000 字符），进一步切分
		if len(para) > 1000 {
			// 按句子切分
			sentences := splitBySentences(para)
			currentChunk := ""
			chunkIndex := i

			for _, sentence := range sentences {
				if len(currentChunk)+len(sentence) > 1000 && currentChunk != "" {
					chunks = append(chunks, Chunk{
						File:       file,
						ChunkIndex: chunkIndex,
						Content:    currentChunk,
						Title:      title,
					})
					currentChunk = sentence
					chunkIndex++
				} else {
					if currentChunk != "" {
						currentChunk += "\n"
					}
					currentChunk += sentence
				}
			}

			if currentChunk != "" {
				chunks = append(chunks, Chunk{
					File:       file,
					ChunkIndex: chunkIndex,
					Content:    currentChunk,
					Title:      title,
				})
			}
		} else {
			chunks = append(chunks, Chunk{
				File:       file,
				ChunkIndex: i,
				Content:    para,
				Title:      title,
			})
		}
	}

	return chunks
}

// splitBySentences 按句子切分文本
func splitBySentences(text string) []string {
	// 简单的按句号、问号、感叹号切分
	text = strings.ReplaceAll(text, "。", "。\n")
	text = strings.ReplaceAll(text, "！", "！\n")
	text = strings.ReplaceAll(text, "？", "？\n")
	text = strings.ReplaceAll(text, ". ", ".\n")
	text = strings.ReplaceAll(text, "! ", "!\n")
	text = strings.ReplaceAll(text, "? ", "?\n")

	sentences := strings.Split(text, "\n")
	result := make([]string, 0, len(sentences))
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// LoadBlogsFromFS 从文件系统加载所有博客文件
func LoadBlogsFromFS(fsys fs.FS, blogDir string) (map[string]string, error) {
	blogs := make(map[string]string)

	err := fs.WalkDir(fsys, blogDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		// 使用相对路径作为 key（如 "blogs/xxx.md"）
		relPath := strings.TrimPrefix(path, blogDir+"/")
		blogs[relPath] = string(content)

		return nil
	})

	return blogs, err
}


// GetBlogFilePath 获取博客文件的完整路径（用于从 dist 目录读取）
func GetBlogFilePath(relPath string) string {
	// 如果已经是完整路径，直接返回
	if strings.HasPrefix(relPath, "blogs/") {
		return relPath
	}
	// 否则添加 blogs/ 前缀
	return filepath.Join("blogs", relPath)
}

