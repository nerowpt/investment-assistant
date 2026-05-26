package library

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"
)

// SimilarityResult 写入 candidate.similarity_json（03 §8.11）。
type SimilarityResult struct {
	MatchTier   string              `json:"match_tier"`
	Candidates  []SimilarityMatch   `json:"candidates,omitempty"`
	DiffSummary string              `json:"diff_summary,omitempty"`
	AnalyzedAt  string              `json:"analyzed_at"`
}

// SimilarityMatch 单个相似 library_item。
type SimilarityMatch struct {
	LibraryItemID string  `json:"library_item_id"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
	Title         string  `json:"title,omitempty"`
}

// ItemForSimilarity 参与相似度比较的已有素材摘要。
type ItemForSimilarity struct {
	ID              string
	Title           string
	RelatedStocks   []string
	Tags            []string
	DedupKey        string
}

// AnalyzeSimilarity 规则引擎相似度（MVP-1）。
func AnalyzeSimilarity(
	in DedupInput,
	stocks []string,
	existing []ItemForSimilarity,
	exactItemID string,
) SimilarityResult {
	now := time.Now().Format(time.RFC3339)
	if exactItemID != "" {
		return SimilarityResult{
			MatchTier:  "exact",
			AnalyzedAt: now,
			Candidates: []SimilarityMatch{{
				LibraryItemID: exactItemID,
				Score:         1.0,
				Reason:        "dedup_key_exact",
			}},
			DiffSummary: "与已有素材 dedup_key 完全相同，默认 dismiss",
		}
	}

	var matches []SimilarityMatch
	stockSet := map[string]struct{}{}
	for _, s := range stocks {
		stockSet[s] = struct{}{}
	}
	inTitle := tokenize(in.Title)

	for _, item := range existing {
		overlapStock := false
		for _, s := range item.RelatedStocks {
			if _, ok := stockSet[s]; ok {
				overlapStock = true
				break
			}
		}
		titleScore := jaccard(inTitle, tokenize(item.Title))
		tagScore := jaccard(sliceToSet(nil), sliceToSet(item.Tags))

		score := 0.0
		reason := ""
		switch {
		case overlapStock && titleScore >= 0.45:
			score = 0.55 + titleScore*0.4
			reason = "same_stock+title_overlap"
		case overlapStock && tagScore > 0:
			score = 0.35 + tagScore*0.3
			reason = "same_stock+tag_overlap"
		case titleScore >= 0.6:
			score = titleScore * 0.7
			reason = "title_overlap"
		default:
			continue
		}
		if score < 0.35 {
			continue
		}
		matches = append(matches, SimilarityMatch{
			LibraryItemID: item.ID,
			Score:         score,
			Reason:        reason,
			Title:         item.Title,
		})
	}

	matches = topN(matches, 3)
	tier := "none"
	diff := ""
	if len(matches) > 0 {
		best := matches[0]
		switch {
		case best.Score >= 0.75:
			tier = "near"
			diff = fmtDiff(in.Title, best.Title)
		case best.Score >= 0.35:
			tier = "theme"
			diff = "主题或标的可能相关，请确认 promote / supplement / dismiss"
		}
	}

	return SimilarityResult{
		MatchTier:   tier,
		Candidates:  matches,
		DiffSummary: diff,
		AnalyzedAt:  now,
	}
}

// TagsFromStocks 占位：ingest 阶段 tags 常为空，保留扩展点。
func TagsFromStocks(_ []string) []string {
	return nil
}

func (r SimilarityResult) JSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}

func tokenize(s string) map[string]struct{} {
	s = strings.ToLower(s)
	tokens := map[string]struct{}{}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			if b.Len() >= 2 {
				tokens[b.String()] = struct{}{}
			}
			b.Reset()
		}
	}
	if b.Len() >= 2 {
		tokens[b.String()] = struct{}{}
	}
	return tokens
}

func sliceToSet(list []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, s := range list {
		if s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func topN(in []SimilarityMatch, n int) []SimilarityMatch {
	if len(in) <= 1 {
		return in
	}
	// 简单选择排序（n 很小）
	out := append([]SimilarityMatch(nil), in...)
	for i := 0; i < len(out)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[maxIdx].Score {
				maxIdx = j
			}
		}
		out[i], out[maxIdx] = out[maxIdx], out[i]
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func fmtDiff(a, b string) string {
	if a == b {
		return "标题相同"
	}
	return "新素材与「" + b + "」标题部分重叠，请核对是否补充"
}
