package yamlstore

import (
	"fmt"
)

const RiskRulesSchemaVersion = 1

// RiskRules M7 仓位护栏配置（03 §10B.5；SoT 在 state/risk_rules.yaml）。
type RiskRules struct {
	SchemaVersion   int             `yaml:"schema_version"`   // 文件 schema 版本，当前为 1
	Meta            YAMLMeta        `yaml:"meta"`             // 元信息（最后更新时间等）
	PositionLimits  PositionLimits  `yaml:"position_limits"`  // 四维集中度阈值（单标的/板块/总仓位/thesis）
	LegacyOverLimit LegacyOverLimit `yaml:"legacy_over_limit"` // 存量超限策略（02 §16.2 规则 5）
}

// PositionLimits 四维集中度阈值（02 §18.4）。
type PositionLimits struct {
	SingleStock  LimitPair `yaml:"single_stock"`  // 单标的占总资产 %
	SingleSector LimitPair `yaml:"single_sector"` // 单 sector_id 合计 %
	TotalEquity  LimitPair `yaml:"total_equity"`  // 全部 holding 合计 %
	SingleThesis LimitPair `yaml:"single_thesis"` // 同 thesis_id 多标的合计 %
}

// LimitPair warning / hard_block 双阈值（占总资产 %）。
type LimitPair struct {
	WarningPct   float64 `yaml:"warning_pct"`    // 超过则 warning，记入 risk_exceptions
	HardBlockPct float64 `yaml:"hard_block_pct"` // 超过则 hard_block，submit 须 exception_json
}

// LegacyOverLimit 存量导入超限后的扩大交易策略（02 §16.2）。
type LegacyOverLimit struct {
	AllowOnImport          bool `yaml:"allow_on_import"`           // import checklist 是否允许标记 legacy_over_limit
	BlockExpansionOnLegacy bool `yaml:"block_expansion_on_legacy"` // 已 legacy 标记时禁止 buy/add 继续扩大
}

// PersonalRedlines 个人禁区（03 §10B.6；SoT 在 state/personal_redlines.yaml）。
type PersonalRedlines struct {
	SchemaVersion int       `yaml:"schema_version"` // 文件 schema 版本
	Meta          YAMLMeta  `yaml:"meta"`           // 元信息
	Redlines      []Redline `yaml:"redlines"`       // 禁区规则列表（r001–r012 等）
}

// Redline 单条个人禁区（02 §18.6–§18.7）。
type Redline struct {
	ID               string   `yaml:"id"`                // 规则 id，如 r004
	Rule             string   `yaml:"rule"`              // 人类可读规则描述
	Severity         string   `yaml:"severity"`          // hard 强拦截 | soft 软警示
	Enabled          bool     `yaml:"enabled"`           // 是否启用（模板中部分默认 false）
	RelatedScenarios []string `yaml:"related_scenarios"` // 适用 checklist 类型：buy/add/sell 等
	ExceptionAllowed bool     `yaml:"exception_allowed"` // hard 时是否允许写 exception_json 后继续
	TriggerKind      string   `yaml:"trigger_kind"`      // builtin | keyword | field_check
}

// YAMLMeta 通用 meta 块（portfolio / risk_rules 等共用形状）。
type YAMLMeta struct {
	UpdatedAt string `yaml:"updated_at"` // 最后一次合法写入时间（ISO8601）
}

// LoadRiskRules 读取 risk_rules.yaml。
func LoadRiskRules(path string) (*RiskRules, error) {
	var r RiskRules
	if err := readYAML(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// LoadPersonalRedlines 读取 personal_redlines.yaml。
func LoadPersonalRedlines(path string) (*PersonalRedlines, error) {
	var r PersonalRedlines
	if err := readYAML(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ValidateRiskRules 基本结构校验。
func ValidateRiskRules(r *RiskRules) error {
	if r == nil {
		return fmt.Errorf("risk_rules 为空")
	}
	if r.SchemaVersion == 0 {
		r.SchemaVersion = RiskRulesSchemaVersion
	}
	return nil
}
