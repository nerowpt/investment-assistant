package yamlstore

import (
	"fmt"
	"regexp"
)

const ControlledTagsSchemaVersion = 2

// ControlledTags 对应 state/controlled_tags.yaml（03 §8.8、§十C.10）。
type ControlledTags struct {
	SchemaVersion int              `yaml:"schema_version"`
	Scheme        string           `yaml:"scheme"`
	Rules         ControlledRules  `yaml:"rules"`
	System        []ControlledTag  `yaml:"system"`
	User          []ControlledTag  `yaml:"user"`
	Suggested     []ControlledTag  `yaml:"suggested"`
}

// ControlledRules 词表规则块。
type ControlledRules struct {
	AllowFreeformTags bool   `yaml:"allow_freeform_tags"`
	FreeformField     string `yaml:"freeform_field"`
	MaxTagsPerItem    int    `yaml:"max_tags_per_item"`
	SuggestedTTLDays  int    `yaml:"suggested_ttl_days"`
}

// ControlledTag 单个受控标签条目。
type ControlledTag struct {
	ID              string `yaml:"id"`
	Label           string `yaml:"label"`
	Dimension       string `yaml:"dimension"`
	Enabled         *bool  `yaml:"enabled,omitempty"` // system 层可禁用
	SourceReviewID  string `yaml:"source_review_id,omitempty"`
}

var tagIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,48}$`)

// LoadControlledTags 读取 controlled_tags.yaml。
func LoadControlledTags(path string) (*ControlledTags, error) {
	var t ControlledTags
	if err := readYAML(path, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveControlledTags 原子写入 controlled_tags.yaml。
func SaveControlledTags(path string, t *ControlledTags) error {
	if t.SchemaVersion == 0 {
		t.SchemaVersion = ControlledTagsSchemaVersion
	}
	return writeYAML(path, t)
}

// EnabledIDs 返回可写入 library_items.tags_json 的 id 集合（enabled system ∪ user）。
func (t *ControlledTags) EnabledIDs() map[string]struct{} {
	out := map[string]struct{}{}
	for _, tag := range t.System {
		if tag.Enabled != nil && !*tag.Enabled {
			continue
		}
		out[tag.ID] = struct{}{}
	}
	for _, tag := range t.User {
		out[tag.ID] = struct{}{}
	}
	return out
}

// SuggestedIDs 返回 suggested 层 id 集合。
func (t *ControlledTags) SuggestedIDs() map[string]struct{} {
	out := map[string]struct{}{}
	for _, tag := range t.Suggested {
		out[tag.ID] = struct{}{}
	}
	return out
}

// ValidateTagIDs 校验 tags 是否均可写入 library_items。
func (t *ControlledTags) ValidateTagIDs(ids []string) error {
	enabled := t.EnabledIDs()
	suggested := t.SuggestedIDs()
	max := t.Rules.MaxTagsPerItem
	if max <= 0 {
		max = 12
	}
	if len(ids) > max {
		return fmt.Errorf("tags 超过上限 %d", max)
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("tags 含空 id")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("tags 重复: %s", id)
		}
		seen[id] = struct{}{}
		if _, ok := suggested[id]; ok {
			return fmt.Errorf("tag %s 仍在 suggested 层，须先 tags confirm", id)
		}
		if _, ok := enabled[id]; !ok {
			return fmt.Errorf("tag %s 不在 enabled(system)∪user", id)
		}
	}
	return nil
}

// ValidateNewUserTagID 校验 tags add 的新 id 格式与冲突。
func (t *ControlledTags) ValidateNewUserTagID(id string) error {
	if !tagIDPattern.MatchString(id) {
		return fmt.Errorf("id 格式须为 [a-z][a-z0-9_]{2,48}，当前: %s", id)
	}
	for _, tag := range t.System {
		if tag.ID == id {
			return fmt.Errorf("id 与 system 层冲突: %s", id)
		}
	}
	for _, tag := range t.User {
		if tag.ID == id {
			return fmt.Errorf("id 已存在于 user 层: %s", id)
		}
	}
	return nil
}

// MergeTags 并集去重合并 tags（tags:A supplement 语义）。
func MergeTags(base, extra []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, list := range [][]string{base, extra} {
		for _, id := range list {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
