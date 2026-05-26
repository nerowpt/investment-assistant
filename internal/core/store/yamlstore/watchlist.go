package yamlstore

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const WatchlistSchemaVersion = 1

// Watchlist 对应 03 §10B.3 watchlist.yaml 根结构（观察池 SoT；watching 不进 portfolio）。
type Watchlist struct {
	SchemaVersion int            `yaml:"schema_version"` // 文件 schema 版本，当前为 1
	Meta          WatchlistMeta  `yaml:"meta"`           // 元信息
	Items         []WatchlistItem `yaml:"items"`         // 观察项列表；无观察时可 []
}

// WatchlistMeta watchlist.yaml 的 meta 块。
type WatchlistMeta struct {
	UpdatedAt string `yaml:"updated_at"` // 最后一次合法写入时间 ISO8601
}

// WatchlistItem 单条观察池记录（对齐 02 §16.1）。
type WatchlistItem struct {
	ID                  string   `yaml:"id"`                             // 主键 w_{YYYYMMDD}_{seq}
	Code                string   `yaml:"code,omitempty"`                 // 标的代码；主题级观察可空
	Name                string   `yaml:"name"`                           // 名称
	WatchType           string   `yaml:"watch_type"`                     // stock | sector | theme | person | event
	State               string   `yaml:"state"`                          // watching | removed
	SourceEntry         string   `yaml:"source_entry"`                   // manual | passive_discovery | from_candidate | from_inspection
	SourceCandidateID   string   `yaml:"source_candidate_id,omitempty"`  // from_candidate 时 cand_*
	WatchReason         string   `yaml:"watch_reason"`                   // 为什么值得观察
	Hypothesis          string   `yaml:"hypothesis"`                     // 待验证假设
	KeyMetricsToWatch   []string `yaml:"key_metrics_to_watch"`           // 跟踪指标，至少 1 项
	ExpectedTrigger     string   `yaml:"expected_trigger"`               // 考虑建仓的触发条件
	InvalidCondition    string   `yaml:"invalid_condition"`              // 移出观察池条件
	ReviewDate          string   `yaml:"review_date"`                    // 下次复查 YYYY-MM-DD
	Priority            string   `yaml:"priority,omitempty"`             // low | medium | high
	RelatedLibraryIDs   []string `yaml:"related_library_ids,omitempty"`  // 关联 L1 id
	RelatedPositions    []string `yaml:"related_positions,omitempty"`    // 关联持仓 code
	CreatedAt           string   `yaml:"created_at"`                     // 创建时间 ISO8601
	UpdatedAt           string   `yaml:"updated_at,omitempty"`           // 最后更新时间
	RemovedAt           string   `yaml:"removed_at,omitempty"`           // state=removed 时必填
	RemovedReason       string   `yaml:"removed_reason,omitempty"`       // 见 ValidateWatchlist 枚举
	RemovedDetail       string   `yaml:"removed_detail,omitempty"`       // removed_reason=other 时说明
	PromotedJournalID   string   `yaml:"promoted_journal_id,omitempty"`  // promoted_to_holding 时 j_*
	Notes               string   `yaml:"notes,omitempty"`                // 自由备注
}

// LoadWatchlist 读取 watchlist.yaml。
func LoadWatchlist(path string) (*Watchlist, error) {
	var w Watchlist
	if err := readYAML(path, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// SaveWatchlist 原子写入 watchlist.yaml。
func SaveWatchlist(path string, w *Watchlist) error {
	if w.SchemaVersion == 0 {
		w.SchemaVersion = WatchlistSchemaVersion
	}
	return writeYAML(path, w)
}

// cloneWatchlist 深拷贝（Memory store 隔离用）。
func cloneWatchlist(w *Watchlist) *Watchlist {
	if w == nil {
		return nil
	}
	raw, err := yaml.Marshal(w)
	if err != nil {
		return w
	}
	var out Watchlist
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return w
	}
	return &out
}

var validWatchTypes = map[string]struct{}{
	"stock": {}, "sector": {}, "theme": {}, "person": {}, "event": {},
}

var validWatchStates = map[string]struct{}{
	"watching": {}, "removed": {},
}

var validSourceEntries = map[string]struct{}{
	"manual": {}, "passive_discovery": {}, "from_candidate": {}, "from_inspection": {},
}

var validRemovedReasons = map[string]struct{}{
	"invalidated": {}, "no_longer_relevant": {}, "promoted_to_holding": {},
	"merged": {}, "other": {},
}

var validPriorities = map[string]struct{}{
	"low": {}, "medium": {}, "high": {},
}

// ValidateWatchlist 校验 schema 与基本约束（不含 SQLite / portfolio 交叉检查）。
func ValidateWatchlist(w *Watchlist) []string {
	var issues []string
	if w.SchemaVersion != WatchlistSchemaVersion {
		issues = append(issues, fmt.Sprintf("schema_version 期望 %d，实际 %d", WatchlistSchemaVersion, w.SchemaVersion))
	}
	if w.Meta.UpdatedAt == "" {
		issues = append(issues, "meta.updated_at 不能为空")
	}
	ids := map[string]int{}
	for i, item := range w.Items {
		prefix := fmt.Sprintf("items[%d]", i)
		if item.ID == "" {
			issues = append(issues, prefix+": id 不能为空")
			continue
		}
		ids[item.ID]++
		if ids[item.ID] > 1 {
			issues = append(issues, fmt.Sprintf("重复 watchlist id: %s", item.ID))
		}
		if item.Name == "" {
			issues = append(issues, prefix+": name 不能为空")
		}
		if _, ok := validWatchStates[item.State]; !ok {
			issues = append(issues, fmt.Sprintf("%s: state 非法 %q（仅 watching|removed）", item.ID, item.State))
		}
		if item.State == "holding" {
			issues = append(issues, fmt.Sprintf("%s: watchlist 不应含 state=holding", item.ID))
		}
		if _, ok := validWatchTypes[item.WatchType]; !ok {
			issues = append(issues, fmt.Sprintf("%s: watch_type 非法 %q", item.ID, item.WatchType))
		}
		if _, ok := validSourceEntries[item.SourceEntry]; !ok {
			issues = append(issues, fmt.Sprintf("%s: source_entry 非法 %q", item.ID, item.SourceEntry))
		}
		if item.SourceEntry == "from_candidate" && item.SourceCandidateID == "" {
			issues = append(issues, fmt.Sprintf("%s: from_candidate 须填 source_candidate_id", item.ID))
		}
		if item.WatchReason == "" {
			issues = append(issues, fmt.Sprintf("%s: watch_reason 不能为空", item.ID))
		}
		if item.Hypothesis == "" {
			issues = append(issues, fmt.Sprintf("%s: hypothesis 不能为空", item.ID))
		}
		if len(item.KeyMetricsToWatch) == 0 {
			issues = append(issues, fmt.Sprintf("%s: key_metrics_to_watch 至少 1 项", item.ID))
		}
		if item.ExpectedTrigger == "" {
			issues = append(issues, fmt.Sprintf("%s: expected_trigger 不能为空", item.ID))
		}
		if item.InvalidCondition == "" {
			issues = append(issues, fmt.Sprintf("%s: invalid_condition 不能为空", item.ID))
		}
		if item.ReviewDate == "" {
			issues = append(issues, fmt.Sprintf("%s: review_date 不能为空", item.ID))
		}
		if item.CreatedAt == "" {
			issues = append(issues, fmt.Sprintf("%s: created_at 不能为空", item.ID))
		}
		if item.Priority != "" {
			if _, ok := validPriorities[item.Priority]; !ok {
				issues = append(issues, fmt.Sprintf("%s: priority 非法 %q", item.ID, item.Priority))
			}
		}
		if item.State == "removed" {
			if item.RemovedAt == "" {
				issues = append(issues, fmt.Sprintf("%s: removed 须填 removed_at", item.ID))
			}
			if item.RemovedReason == "" {
				issues = append(issues, fmt.Sprintf("%s: removed 须填 removed_reason", item.ID))
			} else if _, ok := validRemovedReasons[item.RemovedReason]; !ok {
				issues = append(issues, fmt.Sprintf("%s: removed_reason 非法 %q", item.ID, item.RemovedReason))
			}
			if item.RemovedReason == "other" && item.RemovedDetail == "" {
				issues = append(issues, fmt.Sprintf("%s: removed_reason=other 须填 removed_detail", item.ID))
			}
			if item.RemovedReason == "promoted_to_holding" && item.PromotedJournalID == "" {
				issues = append(issues, fmt.Sprintf("%s: promoted_to_holding 须填 promoted_journal_id（03 §10B.8）", item.ID))
			}
		}
		if item.State == "watching" && item.PromotedJournalID != "" {
			issues = append(issues, fmt.Sprintf("%s: watching 状态不应有 promoted_journal_id", item.ID))
		}
	}
	return issues
}
