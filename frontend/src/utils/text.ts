/** 按 Unicode 字符计数字符串长度（与后端 len([]rune) 一致） */
export function charCount(text: string): number {
  return [...text].length
}
