package checklist

import "strconv"

// FieldType 表单字段类型（驱动 uni-app DynamicForm）。
type FieldType string

const (
	FieldText         FieldType = "text"
	FieldTextarea     FieldType = "textarea"
	FieldNumber       FieldType = "number"
	FieldEnum         FieldType = "enum"
	FieldMultiEnum    FieldType = "multi_enum"    // 多选枚举（如 expected_return_driver）
	FieldLibraryMulti FieldType = "library_multi" // 关联 L1 素材多选（按标的加载）
	FieldBool         FieldType = "bool"
	FieldArray        FieldType = "array"
	FieldDate         FieldType = "date"
)

// FieldOption 枚举选项。
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldSchema 单字段元数据（label/tip/default 供前端傻瓜式填表）。
type FieldSchema struct {
	Key      string        `json:"key"`
	Label    string        `json:"label"`
	Type     FieldType     `json:"type"`
	Required bool          `json:"required"`
	Default  any           `json:"default,omitempty"`
	Tip      string        `json:"tip,omitempty"`
	Group    string        `json:"group"`
	Options  []FieldOption `json:"options,omitempty"`
	Rows     int           `json:"rows,omitempty"` // textarea 行数
}

// FormSchema 某 checklist 类型的完整表单 schema。
type FormSchema struct {
	ChecklistType string        `json:"checklist_type"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Groups        []string      `json:"groups"`
	Fields        []FieldSchema `json:"fields"`
}

// GetFormSchema 返回 checklist 类型的结构化表单 schema（H8 前端动态表单单一数据源）。
func GetFormSchema(checklistType string) (*FormSchema, error) {
	if err := ValidateType(checklistType); err != nil {
		return nil, err
	}
	switch checklistType {
	case "buy":
		return buyFormSchema(), nil
	case "add":
		return addFormSchema(), nil
	case "sell":
		return sellFormSchema(), nil
	case "inspect":
		return inspectFormSchema(), nil
	case "review":
		return reviewFormSchema(), nil
	default:
		return &FormSchema{
			ChecklistType: checklistType,
			Title:         checklistType,
			Description:   "该类型首版请使用默认模板",
			Groups:        []string{"表单"},
			Fields:        []FieldSchema{{Key: "emotion_retrospect", Label: "情绪复盘预留", Type: FieldText, Group: "表单", Default: nil}},
		}, nil
	}
}

func buyFormSchema() *FormSchema {
	return &FormSchema{
		ChecklistType: "buy",
		Title:         "建仓买入",
		Description:   "首次买入某标的，建立持仓。请填写实际成交价与股数（不联动券商）。",
		Groups:        []string{"基本信息", "投资逻辑", "仓位计划", "交易执行", "风险情绪"},
		Fields: []FieldSchema{
			{Key: "source_entry", Label: "录入来源", Type: FieldEnum, Group: "基本信息", Default: "manual",
				Tip: "本次建仓表单的创建入口", Options: []FieldOption{
					{"manual", "手动录入"}, {"from_watchlist", "观察池"}, {"passive_discovery", "被动发现"}, {"from_inspection", "巡检转入"},
				}},
			{Key: "position_type", Label: "仓位类型", Type: FieldEnum, Required: true, Group: "基本信息", Default: "core",
				Tip: "core=主仓长线；swing=波段/修复仓", Options: []FieldOption{{"core", "主仓"}, {"swing", "波段"}}},
			{Key: "buy_reason_summary", Label: "买入理由摘要", Type: FieldTextarea, Required: true, Group: "投资逻辑", Rows: 2,
				Tip: "一两句话说明为什么现在买", Default: ""},
			{Key: "investment_thesis", Label: "持有逻辑", Type: FieldTextarea, Required: true, Group: "投资逻辑", Rows: 4,
				Tip: "approve 后写入 portfolio，巡检时对照此逻辑", Default: ""},
			{Key: "expected_return_driver", Label: "收益驱动", Type: FieldMultiEnum, Group: "投资逻辑",
				Tip: "至少选 1 项，说明本次收益主要来自哪里", Default: []string{"earnings_growth"},
				Options: []FieldOption{
					{"earnings_growth", "业绩增长"},
					{"valuation_repair", "估值修复"},
					{"roe_improvement", "ROE 中枢改善"},
					{"narrative_repricing", "叙事重估"},
					{"cycle_reversal", "周期反转"},
					{"capital_flow", "资金面改善"},
					{"event_catalyst", "事件催化"},
					{"other", "其他"},
				}},
			{Key: "related_library_ids", Label: "关联 L1 素材", Type: FieldLibraryMulti, Group: "投资逻辑",
				Tip: "选择支撑本次买入的研究素材；无素材时须填写下方说明", Default: []string{}},
			{Key: "opportunity_cost_benchmark", Label: "机会成本基准", Type: FieldEnum, Group: "投资逻辑", Default: "HS300",
				Tip: "持仓机会成本对比的指数基准", Options: []FieldOption{
					{"HS300", "沪深300"}, {"CSI_TECH", "科创50"}, {"sector_index", "行业指数"}, {"custom", "自定义"},
				}},
			{Key: "tier_acknowledgement", Label: "低可信度素材确认", Type: FieldBool, Group: "投资逻辑", Default: false,
				Tip: "主要依据素材最高 tier 为 C/D 时须确认「知悉可信度较低」"},
			{Key: "target_price", Label: "目标价（元）", Type: FieldNumber, Group: "投资逻辑", Default: 0,
				Tip: "预期卖出参考价，0 表示暂不设定"},
			{Key: "stop_loss", Label: "止损价（元）", Type: FieldNumber, Group: "投资逻辑", Default: 0,
				Tip: "跌破此价须重新评估 thesis"},
			{Key: "reversal_conditions", Label: "逻辑反转条件", Type: FieldArray, Required: true, Group: "投资逻辑",
				Tip: "出现哪些信号说明买入逻辑不再成立", Default: []string{"待填写"}},
			{Key: "position_size_plan.initial_pct", Label: "初始仓位 %", Type: FieldNumber, Required: true, Group: "仓位计划", Default: 5,
				Tip: "占总资产 %；页面会显示当前 M7 单票阈值，超出须填写超限说明"},
			{Key: "position_size_plan.override_reason", Label: "仓位超限说明", Type: FieldTextarea, Group: "仓位计划", Rows: 2, Default: "",
				Tip: "当初始仓位超过 M7 警告线时必填：说明为何仍可接受及后续约束"},
			{Key: "position_size_plan.max_pct", Label: "最大仓位 %", Type: FieldNumber, Group: "仓位计划", Default: 10,
				Tip: "该标的仓位上限"},
			{Key: "position_size_plan.add_condition", Label: "加仓条件", Type: FieldText, Group: "仓位计划", Default: "",
				Tip: "什么情况下会 add"},
			{Key: "position_size_plan.reduce_condition", Label: "减仓条件", Type: FieldText, Group: "仓位计划", Default: "",
				Tip: "什么情况下会 sell"},
			{Key: "execution_price", Label: "实际成交价（元/股）", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0,
				Tip: "手动填写券商实际成交价，写入 cost_basis"},
			{Key: "shares", Label: "买入股数", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0,
				Tip: "实际买入股数"},
			{Key: "confidence", Label: "信心程度", Type: FieldEnum, Group: "风险情绪", Default: "medium",
				Options: []FieldOption{{"high", "高"}, {"medium", "中"}, {"low", "低"}}},
			{Key: "emotion_tag", Label: "情绪标签", Type: FieldEnum, Group: "风险情绪", Default: "calm",
				Tip: "fomo/greedy/anxious 须额外情绪自检", Options: []FieldOption{{"calm", "冷静"}, {"fomo", "FOMO"}, {"greedy", "贪婪"}, {"anxious", "焦虑"}}},
			{Key: "identified_risks", Label: "已识别风险", Type: FieldArray, Group: "风险情绪", Default: []string{"待填写"}},
			{Key: "no_library_reason", Label: "无 L1 素材说明", Type: FieldTextarea, Group: "投资逻辑", Rows: 2, Default: "",
				Tip: "未选择任何 L1 素材时必填：说明本次买入依据来源（个人判断须写清逻辑）"},
		},
	}
}

func addFormSchema() *FormSchema {
	return &FormSchema{
		ChecklistType: "add",
		Title:         "加仓买入",
		Description:   "已有持仓，追加买入形成新 lot。系统会自动关联首次建仓 journal。",
		Groups:        []string{"关联", "加仓逻辑", "交易执行", "风险情绪"},
		Fields: []FieldSchema{
			{Key: "linked_buy_journal_id", Label: "首次建仓 journal", Type: FieldText, Required: true, Group: "关联", Default: "",
				Tip: "首次 buy approve 产生的 j_*；加仓向导会自动填入，一般无需手改"},
			{Key: "add_trigger", Label: "加仓触发", Type: FieldEnum, Required: true, Group: "加仓逻辑", Default: "thesis_strengthened",
				Options: []FieldOption{{"thesis_strengthened", "逻辑增强"}, {"price_dip", "价格回调"}, {"plan_execution", "计划执行"}}},
			{Key: "add_reason_summary", Label: "加仓理由", Type: FieldTextarea, Required: true, Group: "加仓逻辑", Rows: 2, Default: ""},
			{Key: "thesis_change", Label: "逻辑变化", Type: FieldEnum, Group: "加仓逻辑", Default: "strengthened",
				Options: []FieldOption{{"strengthened", "增强"}, {"unchanged", "不变"}, {"weakened", "减弱"}}},
			{Key: "add_pct", Label: "本次加仓 %", Type: FieldNumber, Required: true, Group: "加仓逻辑", Default: 3,
				Tip: "本次拟增加的仓位占总资产 %"},
			{Key: "max_pct_after_add", Label: "加仓后上限 %", Type: FieldNumber, Group: "加仓逻辑", Default: 10},
			{Key: "execution_price", Label: "实际成交价（元/股）", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0},
			{Key: "shares", Label: "加仓股数", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0},
			{Key: "emotion_tag", Label: "情绪标签", Type: FieldEnum, Group: "风险情绪", Default: "calm",
				Options: []FieldOption{{"calm", "冷静"}, {"fomo", "FOMO"}, {"greedy", "贪婪"}, {"anxious", "焦虑"}}},
		},
	}
}

func sellFormSchema() *FormSchema {
	return &FormSchema{
		ChecklistType: "sell",
		Title:         "卖出/减仓",
		Description:   "部分或全部卖出。系统按股数 FIFO 自动分配 lot，可先预览计划再确认。",
		Groups:        []string{"卖出原因", "交易执行", "复盘"},
		Fields: []FieldSchema{
			{Key: "sell_type", Label: "卖出类型", Type: FieldEnum, Required: true, Group: "卖出原因", Default: "reduce",
				Options: []FieldOption{{"reduce", "减仓"}, {"full", "清仓"}}},
			{Key: "sell_trigger", Label: "卖出触发", Type: FieldEnum, Required: true, Group: "卖出原因", Default: "target_reached",
				Options: []FieldOption{{"target_reached", "达目标价"}, {"thesis_broken", "逻辑破裂"}, {"risk_control", "风控"}, {"rebalance", "再平衡"}}},
			{Key: "sell_reason", Label: "卖出原因码", Type: FieldEnum, Required: true, Group: "卖出原因", Default: "target_achieved",
				Options: []FieldOption{{"target_achieved", "目标达成"}, {"thesis_invalid", "逻辑失效"}, {"stop_loss", "止损"}}},
			{Key: "sell_reason_detail", Label: "卖出说明", Type: FieldTextarea, Group: "卖出原因", Rows: 2, Default: ""},
			{Key: "sell_shares", Label: "卖出股数", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0,
				Tip: "须 ≤ 当前 open lot 可卖股数之和"},
			{Key: "execution_price", Label: "实际成交价（元/股）", Type: FieldNumber, Required: true, Group: "交易执行", Default: 0},
			{Key: "thesis_result", Label: "逻辑验证结果", Type: FieldEnum, Group: "复盘", Default: "partially_validated",
				Options: []FieldOption{{"validated", "验证"}, {"partially_validated", "部分验证"}, {"invalidated", "失效"}}},
			{Key: "thesis_result_explanation", Label: "逻辑结果说明", Type: FieldTextarea, Group: "复盘", Rows: 2, Default: ""},
			{Key: "lesson", Label: "教训总结", Type: FieldTextarea, Required: true, Group: "复盘", Rows: 2, Default: "",
				Tip: "这次卖出你学到了什么"},
			{Key: "is_panic_sell", Label: "是否恐慌卖出", Type: FieldBool, Group: "复盘", Default: false},
			{Key: "emotion_tag", Label: "情绪标签", Type: FieldEnum, Group: "复盘", Default: "calm",
				Options: []FieldOption{{"calm", "冷静"}, {"fomo", "FOMO"}, {"anxious", "焦虑"}}},
		},
	}
}

func reviewFormSchema() *FormSchema {
	return &FormSchema{
		ChecklistType: "review",
		Title:         "投资复盘",
		Description:   "单笔 lot 归因复盘或周期性复盘。从已卖出区进入时默认为单笔 lot 归因。",
		Groups:        []string{"复盘范围", "归因判断", "经验沉淀"},
		Fields: []FieldSchema{
			{Key: "review_type", Label: "复盘类型", Type: FieldEnum, Required: true, Group: "复盘范围", Default: "lot_attribution",
				Options: []FieldOption{
					{"lot_attribution", "单笔 lot 归因"},
					{"quarterly", "季度复盘"},
					{"monthly", "月度复盘"},
				}},
			{Key: "target_lot_id", Label: "目标 lot", Type: FieldText, Group: "复盘范围", Default: "",
				Tip: "单笔 lot 归因时由工作台自动填入，一般无需手改"},
			{Key: "target_code", Label: "标的代码", Type: FieldText, Group: "复盘范围", Default: "",
				Tip: "单笔 lot 归因时自动填入"},
			{Key: "period_start", Label: "区间开始", Type: FieldDate, Required: true, Group: "复盘范围", Default: "",
				Tip: "lot 归因时为 lot 开启日；周期复盘为统计区间起点"},
			{Key: "period_end", Label: "区间结束", Type: FieldDate, Required: true, Group: "复盘范围", Default: "",
				Tip: "lot 归因时为 lot 关闭日"},
			{Key: "attribution.result_category", Label: "结果分类", Type: FieldEnum, Required: true, Group: "归因判断", Default: "good_process_good_result",
				Tip: "过程与结果的四象限分类（02 §16.5）",
				Options: []FieldOption{
					{"good_process_good_result", "过程好，结果好"},
					{"good_process_bad_result", "过程好，结果不好"},
					{"bad_process_good_result", "过程差，结果好"},
					{"bad_process_bad_result", "过程差，结果差"},
				}},
			{Key: "attribution.thesis_contribution", Label: "逻辑贡献说明", Type: FieldTextarea, Group: "归因判断", Rows: 2, Default: "",
				Tip: "本次收益/亏损中，投资逻辑本身起了多大作用"},
			{Key: "attribution.execution_quality", Label: "执行质量", Type: FieldEnum, Group: "归因判断", Default: "as_planned",
				Options: []FieldOption{
					{"as_planned", "按计划执行"},
					{"early", "偏早"},
					{"late", "偏晚"},
					{"impulsive", "冲动执行"},
				}},
			{Key: "confirmed_patterns", Label: "已确认模式/教训", Type: FieldArray, Required: true, Group: "经验沉淀",
				Tip: "至少写 1 条可复用的经验或需避免的模式", Default: []string{"待填写"}},
			{Key: "notes", Label: "补充说明", Type: FieldTextarea, Group: "经验沉淀", Rows: 3, Default: ""},
		},
	}
}

func inspectFormSchema() *FormSchema {
	return &FormSchema{
		ChecklistType: "inspect",
		Title:         "持仓巡检",
		Description:   "定期检视持仓逻辑是否仍成立。须关联首次建仓 journal。",
		Groups:        []string{"关联", "巡检结论"},
		Fields: []FieldSchema{
			{Key: "inspection_type", Label: "巡检类型", Type: FieldEnum, Required: true, Group: "关联", Default: "scheduled",
				Options: []FieldOption{{"scheduled", "定期"}, {"event_driven", "事件驱动"}, {"ad_hoc", "临时"}}},
			{Key: "linked_buy_journal_id", Label: "关联建仓 journal", Type: FieldText, Required: true, Group: "关联", Default: "",
				Tip: "巡检向导会自动填入该标的的首次 buy journal"},
			{Key: "classification", Label: "逻辑分类", Type: FieldEnum, Required: true, Group: "巡检结论", Default: "thesis_intact",
				Options: []FieldOption{{"thesis_intact", "逻辑仍成立"}, {"wait_for_style_switch", "等风格切换"}, {"thesis_weakened", "逻辑减弱"}, {"thesis_broken", "逻辑破裂"}}},
			{Key: "planned_action", Label: "计划动作", Type: FieldEnum, Required: true, Group: "巡检结论", Default: "hold",
				Options: []FieldOption{{"hold", "持有"}, {"add", "加仓"}, {"reduce", "减仓"}, {"sell", "卖出"}, {"watch", "观察"}}},
			{Key: "thesis_still_valid", Label: "逻辑是否仍成立", Type: FieldBool, Group: "巡检结论", Default: true},
			{Key: "key_observations", Label: "关键观察", Type: FieldArray, Group: "巡检结论", Default: []string{},
				Tip: "本次巡检注意到的事实要点"},
		},
	}
}

// BuildPayloadFromFlat 将扁平 key（含 dot 路径）组装为 payload JSON 对象。
func BuildPayloadFromFlat(flat map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range flat {
		normalized, keep := normalizeFlatValue(v)
		if !keep {
			continue
		}
		setNested(out, k, normalized)
	}
	if _, ok := out["emotion_retrospect"]; !ok {
		out["emotion_retrospect"] = nil
	}
	if _, ok := out["related_library_ids"]; !ok {
		out["related_library_ids"] = []any{}
	}
	if _, ok := out["tier_acknowledgement"]; !ok {
		out["tier_acknowledgement"] = false
	}
	if _, ok := out["source_entry"]; !ok {
		out["source_entry"] = "manual"
	}
	return out
}

// normalizeFlatValue 扁平表单值规范化：空字符串跳过；数字字符串转 float64。
func normalizeFlatValue(v any) (any, bool) {
	s, ok := v.(string)
	if !ok {
		return v, true
	}
	if s == "" {
		return nil, false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return s, true
}

func setNested(root map[string]any, path string, value any) {
	parts := splitPath(path)
	if len(parts) == 1 {
		root[parts[0]] = value
		return
	}
	cur := root
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = value
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// DefaultFlatValues 从 schema 提取默认值扁平 map。
func DefaultFlatValues(schema *FormSchema) map[string]any {
	out := map[string]any{}
	for _, f := range schema.Fields {
		if f.Default != nil {
			out[f.Key] = f.Default
		}
	}
	return out
}
