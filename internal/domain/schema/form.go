package schema

type FormMode string

type FormStep string

type FormField string

const (
	FormModeCreate FormMode = "create"
	FormModeEdit   FormMode = "edit"
)

const (
	FormStepQuestion    FormStep = "question"
	FormStepAnswer      FormStep = "answer"
	FormStepPreview     FormStep = "preview"
	FormStepChooseField FormStep = "choose_field"
	FormStepEditInput   FormStep = "edit_input"
	FormStepPoolInput   FormStep = "pool_input"
	FormStepPoolPreview FormStep = "pool_preview"
	FormStepPoolEdit    FormStep = "pool_edit"
)

const (
	FormFieldQuestion FormField = "question"
	FormFieldAnswer   FormField = "answer"
)

type QuestionDraft struct {
	QuestionText string `json:"question_text"`
	AnswerText   string `json:"answer_text"`
}

type FormState struct {
	Mode       FormMode      `json:"mode"`
	Step       FormStep      `json:"step"`
	QuestionID string        `json:"question_id"`
	Page       int           `json:"page"`
	Field      FormField     `json:"field"`
	Draft      QuestionDraft `json:"draft"`
	PoolItems  []QuestionDraft `json:"pool_items,omitempty"`
	PoolIndex  int             `json:"pool_index,omitempty"`
	PoolSaved  int             `json:"pool_saved,omitempty"`
}
