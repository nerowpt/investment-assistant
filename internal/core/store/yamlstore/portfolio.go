package yamlstore

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

const PortfolioSchemaVersion = 1

// Portfolio 对应 03 §10B.2 portfolio.yaml 根结构。
type Portfolio struct {
	SchemaVersion int                 `yaml:"schema_version"` // 文件 schema 版本，当前为 1
	Meta          PortfolioMeta       `yaml:"meta"`           // 元信息（最后更新时间等）
	Positions     []PortfolioPosition `yaml:"positions"`      // 持仓列表；无持仓时为 []
}

// PortfolioMeta portfolio.yaml 的 meta 块。
type PortfolioMeta struct {
	UpdatedAt      string           `yaml:"updated_at"`       // 最后一次合法写入时间（ISO8601）
	TotalEquityRef *decimal.Decimal `yaml:"total_equity_ref"` // 可选总资产参考，用于推算 position_pct
	Currency       string           `yaml:"currency"`         // 计价货币，默认 CNY
}

// PortfolioPosition 单标的持仓视图（Layer A SoT；历史流水在 SQLite）。
type PortfolioPosition struct {
	Code                     string               `yaml:"code"`                          // 标的代码，文件内唯一
	Name                     string               `yaml:"name"`                          // 标的名称
	State                    string               `yaml:"state"`                         // holding / closed；watching 仅属于 watchlist（Q2A）
	PositionType             string               `yaml:"position_type"`                 // core 主仓 / swing 波段
	PositionPct              decimal.Decimal      `yaml:"position_pct"`                  // 当前占总资产 %，= open/partial lot 的 current_pct 之和
	CostBasis                decimal.Decimal      `yaml:"cost_basis"`                    // 加权平均成本价；可由 open lots 重算
	Shares                   *decimal.Decimal     `yaml:"shares,omitempty"`              // 可选维护的持股数量
	SectorID                 string               `yaml:"sector_id,omitempty"`           // 板块受控标签 id，供 M7 集中度
	ThesisID                 string               `yaml:"thesis_id,omitempty"`           // 投资逻辑分组 id，同 thesis 多标的共用
	EntryDate                string               `yaml:"entry_date"`                    // 首次进入 holding 的日期
	ClosedAt                 string               `yaml:"closed_at,omitempty"`           // state=closed 时必填清仓日
	ThesisVersion            int                  `yaml:"thesis_version"`                // 持有逻辑版本，用户更新 thesis 时递增
	InvestmentThesis         string               `yaml:"investment_thesis"`             // 当前持有逻辑全文（非历史快照）
	TargetPrice              yaml.Node            `yaml:"target_price"`                  // 目标价：数值或 {lower, upper} 区间
	StopLoss                 decimal.Decimal      `yaml:"stop_loss"`                     // 止损/减仓线
	ReversalConditions       []string             `yaml:"reversal_conditions"`           // 逻辑反转条件，至少 1 项
	OpportunityCostBenchmark string               `yaml:"opportunity_cost_benchmark"`    // 机会成本基准：HS300 / CSI_TECH 等
	Confidence               string               `yaml:"confidence,omitempty"`          // 最近一次 buy/add 时的置信度缓存
	RelatedLibraryIDs        []string             `yaml:"related_library_ids,omitempty"` // 当前有效依据的 L1 素材 id
	LotIDs                   []string             `yaml:"lot_ids"`                       // 关联 lot id（含已关闭，便于归因）
	JournalIDs               []string             `yaml:"journal_ids"`                   // 关联 journal id，按时间追加
	WatchlistOriginID        string               `yaml:"watchlist_origin_id,omitempty"` // 若从观察池升级，原 w_* id
	LegacyFlags              []string             `yaml:"legacy_flags,omitempty"`        // 存量导入标记，如 legacy_over_limit
	Monitoring               *PositionMonitoring  `yaml:"monitoring,omitempty"`          // 最近一次巡检摘要（与 SQLite inspection_records 对齐，docs/06 §D12）
	Notes                    string               `yaml:"notes,omitempty"`               // 用户自由备注
}

// PositionMonitoring 巡检摘要的可选缓存（SoT 仍在 SQLite inspection_records；此处用于 CLI 快速查看）。
// docs/06 §D12：与 portfolio.yaml.example 中 monitoring 子结构一一对应，
// 缺该字段会导致 load → save 时 monitoring 块被序列化丢失。
type PositionMonitoring struct {
	LastInspectionID  string `yaml:"last_inspection_id,omitempty"`  // 最近一次巡检 record id
	LastInspectionAt  string `yaml:"last_inspection_at,omitempty"`  // 最近一次巡检时间 ISO8601
	NextInspectionDue string `yaml:"next_inspection_due,omitempty"` // 下次巡检到期日（YYYY-MM-DD）
	Classification    string `yaml:"classification,omitempty"`      // 巡检 Checklist 分类结论快照（02 §16）
	PlannedAction     string `yaml:"planned_action,omitempty"`      // 巡检后计划动作：hold / add / reduce / exit
}

// LoadPortfolio 读取 portfolio.yaml。
func LoadPortfolio(path string) (*Portfolio, error) {
	var p Portfolio
	if err := readYAML(path, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SavePortfolio 原子写入 portfolio.yaml。
func SavePortfolio(path string, p *Portfolio) error {
	if p.SchemaVersion == 0 {
		p.SchemaVersion = PortfolioSchemaVersion
	}
	return writeYAML(path, p)
}

// clonePortfolio 通过 YAML round-trip 做深拷贝（用于 Memory store 隔离读写双方引用）。
func clonePortfolio(p *Portfolio) *Portfolio {
	if p == nil {
		return nil
	}
	raw, err := yaml.Marshal(p)
	if err != nil {
		// schema 内部错误极少发生；保守返回原指针避免 panic
		return p
	}
	var out Portfolio
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return p
	}
	return &out
}

// ValidatePortfolio 校验 schema 与基本约束（不含 SQLite 交叉检查）。
func ValidatePortfolio(p *Portfolio) []string {
	var issues []string
	if p.SchemaVersion != PortfolioSchemaVersion {
		issues = append(issues, fmt.Sprintf("schema_version 期望 %d，实际 %d", PortfolioSchemaVersion, p.SchemaVersion))
	}
	if p.Meta.UpdatedAt == "" {
		issues = append(issues, "meta.updated_at 不能为空")
	}
	codes := map[string]int{}
	for i, pos := range p.Positions {
		if pos.Code == "" {
			issues = append(issues, fmt.Sprintf("positions[%d].code 不能为空", i))
			continue
		}
		codes[pos.Code]++
		if codes[pos.Code] > 1 {
			issues = append(issues, fmt.Sprintf("重复 code: %s", pos.Code))
		}
		if pos.State == "watching" {
			issues = append(issues, fmt.Sprintf("%s: portfolio 不应含 state=watching（见 Q2A）", pos.Code))
		}
		if pos.State == "closed" {
			if !pos.PositionPct.IsZero() {
				issues = append(issues, fmt.Sprintf("%s: closed 标的 position_pct 应为 0", pos.Code))
			}
			if pos.ClosedAt == "" {
				issues = append(issues, fmt.Sprintf("%s: closed 标的缺少 closed_at", pos.Code))
			}
		}
	}
	return issues
}
