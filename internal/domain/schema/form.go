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
	QuestionID int64         `json:"question_id"`
	Page       int           `json:"page"`
	Field      FormField     `json:"field"`
	Draft      QuestionDraft `json:"draft"`
}
