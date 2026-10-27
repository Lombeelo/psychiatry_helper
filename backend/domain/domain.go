// internal/domain/model.go
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrInvalidTransition     = errors.New("invalid status transition")
	ErrInactiveRelation      = errors.New("active care relationship required")
	ErrAssignmentMismatch    = errors.New("assignment does not belong to patient")
	ErrAlreadyEnded          = errors.New("care relationship already ended")
	ErrNotFound              = errors.New("not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrDuplicateRelationship = errors.New("active care relationship already exists")
	ErrAccountExists         = errors.New("account already exists")
)

type ID string

func (id ID) Valid() bool {
	return strings.TrimSpace(string(id)) != ""
}

// ---------- Общие типы ----------

type AccountRole string

const (
	RolePatient AccountRole = "PATIENT"
	RoleDoctor  AccountRole = "DOCTOR"
	RoleAdmin   AccountRole = "ADMIN"
)

// UserCode — код, который отображается вместо персональных данных:
// PXXXXXX у пациента и VXXXXXX у врача. Администратор — заглушка
// без рабочего кода.
type UserCode struct {
	Value string      `json:"value"`
	Role  AccountRole `json:"role"`
}

const userCodeSuffixLength = 6

func (c UserCode) Validate() error {
	value := strings.TrimSpace(c.Value)
	if len(value) != userCodeSuffixLength+1 {
		return fmt.Errorf("%w: user code must match PXXXXXX or VXXXXXX", ErrInvalidInput)
	}

	var prefix byte
	switch c.Role {
	case RolePatient:
		prefix = 'P'
	case RoleDoctor:
		prefix = 'V'
	case RoleAdmin:
		return fmt.Errorf("%w: administrator accounts are a stub and have no user code", ErrInvalidInput)
	default:
		return fmt.Errorf("%w: unknown account role", ErrInvalidInput)
	}

	if value[0] != prefix {
		return fmt.Errorf("%w: code %q must start with %q", ErrInvalidInput, value, string(prefix))
	}
	for i := 1; i < len(value); i++ {
		if !isUpperAlphaNumeric(value[i]) {
			return fmt.Errorf("%w: user code allows only A-Z and 0-9", ErrInvalidInput)
		}
	}
	return nil
}

func isUpperAlphaNumeric(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

// Valid сообщает, соответствует ли код ожидаемому формату.
func (c UserCode) Valid() bool {
	return c.Validate() == nil
}

type PersonalData struct {
	DisplayName string    `json:"display_name"`
	DateOfBirth time.Time `json:"date_of_birth"`
}

func (p PersonalData) Validate() error {
	if strings.TrimSpace(p.DisplayName) == "" {
		return fmt.Errorf("%w: display name is required", ErrInvalidInput)
	}
	if p.DateOfBirth.IsZero() {
		return fmt.Errorf("%w: date of birth is required", ErrInvalidInput)
	}
	if p.DateOfBirth.After(time.Now()) {
		return fmt.Errorf("%w: date of birth is in the future", ErrInvalidInput)
	}
	return nil
}

// ---------- Учётные записи и связь ----------

type Account struct {
	ID        ID           `json:"account_id"`
	Code      UserCode     `json:"code"`
	Personal  PersonalData `json:"personal_data"`
	CreatedAt time.Time    `json:"created_at"`
}

type PatientProfile struct {
	Notes string `json:"notes"`
}

type DoctorProfile struct {
	Specialty string `json:"specialty"`
}

type Consent struct {
	Accepted      bool       `json:"accepted"`
	PolicyVersion string     `json:"policy_version"`
	AcceptedAt    *time.Time `json:"accepted_at"`
}

func (c Consent) Validate() error {
	if c.Accepted && (c.AcceptedAt == nil || strings.TrimSpace(c.PolicyVersion) == "") {
		return fmt.Errorf("%w: accepted consent requires version and timestamp", ErrInvalidInput)
	}
	return nil
}

type Patient struct {
	Account
	Profile PatientProfile `json:"profile"`
	Consent Consent        `json:"consent"`
}

type Doctor struct {
	Account
	Profile DoctorProfile `json:"profile"`
}

// AdminStub — обозначение роли администратора в MVP.
// Административные функции не реализуются.
type AdminStub struct {
	Account
	Available bool `json:"available"`
}

type RelationshipStatus string

const (
	RelationshipActive RelationshipStatus = "ACTIVE"
	RelationshipEnded  RelationshipStatus = "ENDED"
)

// CareRelationship — агрегатная граница связи врача и пациента.
type CareRelationship struct {
	ID        ID                 `json:"relationship_id"`
	PatientID ID                 `json:"patient_id"`
	DoctorID  ID                 `json:"doctor_id"`
	CreatedAt time.Time          `json:"created_at"`
	EndedAt   *time.Time         `json:"ended_at"`
	Status    RelationshipStatus `json:"status"`
}

func NewCareRelationship(id, patientID, doctorID ID, now time.Time) (*CareRelationship, error) {
	if !id.Valid() || !patientID.Valid() || !doctorID.Valid() || patientID == doctorID {
		return nil, fmt.Errorf("%w: invalid relationship identifiers", ErrInvalidInput)
	}
	return &CareRelationship{
		ID: id, PatientID: patientID, DoctorID: doctorID,
		CreatedAt: now, Status: RelationshipActive,
	}, nil
}

func (r *CareRelationship) End(now time.Time) error {
	if r.Status != RelationshipActive {
		return ErrAlreadyEnded
	}
	r.Status = RelationshipEnded
	r.EndedAt = &now
	return nil
}

// ---------- Назначения ----------

type AssignmentType string

const (
	AssignmentMedication    AssignmentType = "MEDICATION"
	AssignmentQuestionnaire AssignmentType = "QUESTIONNAIRE"
	AssignmentAppCheckIn    AssignmentType = "APP_CHECK_IN"
	AssignmentJournalTask   AssignmentType = "JOURNAL_TASK"
)

type AssignmentStatus string

const (
	AssignmentActive    AssignmentStatus = "ACTIVE"
	AssignmentCompleted AssignmentStatus = "COMPLETED"
	AssignmentPaused    AssignmentStatus = "PAUSED"
	AssignmentExpired   AssignmentStatus = "EXPIRED"
	AssignmentCancelled AssignmentStatus = "CANCELLED"
)

type Period struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func (p Period) Validate() error {
	if p.StartsAt.IsZero() || p.EndsAt.IsZero() || !p.EndsAt.After(p.StartsAt) {
		return fmt.Errorf("%w: invalid period", ErrInvalidInput)
	}
	return nil
}

func (p Period) Contains(t time.Time) bool {
	return !t.Before(p.StartsAt) && !t.After(p.EndsAt)
}

type Schedule struct {
	Frequency  string         `json:"frequency"`
	DaysOfWeek []time.Weekday `json:"days_of_week"`
	TimesOfDay []string       `json:"times_of_day"` // формат HH:MM; часовой пояс — пациента
}

func (s Schedule) Validate() error {
	if strings.TrimSpace(s.Frequency) == "" {
		return fmt.Errorf("%w: schedule frequency is required", ErrInvalidInput)
	}
	for _, value := range s.TimesOfDay {
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Errorf("%w: invalid time %q; expected HH:MM", ErrInvalidInput, value)
		}
	}
	return nil
}

type AssignmentDetails interface {
	Validate() error
	Kind() AssignmentType
}

type Dosage struct {
	Amount       float64 `json:"amount"`
	Unit         string  `json:"unit"`
	Instructions string  `json:"instructions"`
}

func (d Dosage) Validate() error {
	if d.Amount <= 0 || strings.TrimSpace(d.Unit) == "" {
		return fmt.Errorf("%w: invalid dosage", ErrInvalidInput)
	}
	return nil
}

type MedicationDetails struct {
	MedicationID ID     `json:"medication_id"`
	Dosage       Dosage `json:"dosage"`
	Instructions string `json:"instructions"`
}

func (d MedicationDetails) Kind() AssignmentType { return AssignmentMedication }
func (d MedicationDetails) Validate() error {
	if !d.MedicationID.Valid() {
		return fmt.Errorf("%w: medication ID is required", ErrInvalidInput)
	}
	return d.Dosage.Validate()
}

type QuestionnaireDetails struct {
	QuestionnaireID        ID `json:"questionnaire_id"`
	QuestionnaireVersionID ID `json:"questionnaire_version_id"`
}

func (d QuestionnaireDetails) Kind() AssignmentType { return AssignmentQuestionnaire }
func (d QuestionnaireDetails) Validate() error {
	if !d.QuestionnaireID.Valid() || !d.QuestionnaireVersionID.Valid() {
		return fmt.Errorf("%w: questionnaire and version IDs are required", ErrInvalidInput)
	}
	return nil
}

type AppCheckInDetails struct {
	Instructions string `json:"instructions"`
}

func (d AppCheckInDetails) Kind() AssignmentType { return AssignmentAppCheckIn }
func (d AppCheckInDetails) Validate() error      { return nil }

type JournalTaskDetails struct {
	Prompt string `json:"prompt"`
}

func (d JournalTaskDetails) Kind() AssignmentType { return AssignmentJournalTask }
func (d JournalTaskDetails) Validate() error      { return nil }

// Assignment — агрегат. Детали задаются типизированным интерфейсом.
type Assignment struct {
	ID        ID                `json:"assignment_id"`
	PatientID ID                `json:"patient_id"`
	DoctorID  ID                `json:"doctor_id"`
	Type      AssignmentType    `json:"type"`
	Title     string            `json:"title"`
	Period    Period            `json:"period"`
	Schedule  Schedule          `json:"schedule"`
	Status    AssignmentStatus  `json:"status"`
	Details   AssignmentDetails `json:"details"`
	CreatedAt time.Time         `json:"created_at"`
}

func NewAssignment(
	id, patientID, doctorID ID,
	title string,
	period Period,
	schedule Schedule,
	details AssignmentDetails,
	now time.Time,
) (*Assignment, error) {
	if !id.Valid() || !patientID.Valid() || !doctorID.Valid() || details == nil {
		return nil, fmt.Errorf("%w: missing assignment data", ErrInvalidInput)
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if err := period.Validate(); err != nil {
		return nil, err
	}
	if err := schedule.Validate(); err != nil {
		return nil, err
	}
	if err := details.Validate(); err != nil {
		return nil, err
	}

	return &Assignment{
		ID: id, PatientID: patientID, DoctorID: doctorID,
		Type: details.Kind(), Title: title, Period: period,
		Schedule: schedule, Status: AssignmentActive,
		Details: details, CreatedAt: now,
	}, nil
}

func (a *Assignment) Pause() error {
	if a.Status != AssignmentActive {
		return ErrInvalidTransition
	}
	a.Status = AssignmentPaused
	return nil
}

func (a *Assignment) Resume() error {
	if a.Status != AssignmentPaused {
		return ErrInvalidTransition
	}
	a.Status = AssignmentActive
	return nil
}

func (a *Assignment) Close() error {
	if a.Status != AssignmentActive && a.Status != AssignmentPaused {
		return ErrInvalidTransition
	}
	a.Status = AssignmentCompleted
	return nil
}

func (a *Assignment) Cancel() error {
	if a.Status != AssignmentActive && a.Status != AssignmentPaused {
		return ErrInvalidTransition
	}
	a.Status = AssignmentCancelled
	return nil
}

// ---------- Выполнение назначения ----------

type MedicationIntake struct {
	TakenAt         time.Time `json:"taken_at"`
	ReportedAsTaken bool      `json:"reported_as_taken"`
	DosageTaken     string    `json:"dosage_taken"`
}

func (m MedicationIntake) Validate() error {
	if m.TakenAt.IsZero() {
		return fmt.Errorf("%w: intake time is required", ErrInvalidInput)
	}
	return nil
}

type SideEffectReport struct {
	Effects    []string  `json:"effects"`
	Comment    string    `json:"comment"`
	ReportedAt time.Time `json:"reported_at"`
}

func (s SideEffectReport) Validate() error {
	if s.ReportedAt.IsZero() {
		return fmt.Errorf("%w: side effect report time is required", ErrInvalidInput)
	}
	if len(s.Effects) == 0 && strings.TrimSpace(s.Comment) == "" {
		return fmt.Errorf("%w: side effect report requires effects or a comment", ErrInvalidInput)
	}
	return nil
}

// Completion — отдельный агрегат, чтобы выполнения можно было хранить независимо.
type Completion struct {
	ID           ID                `json:"completion_id"`
	AssignmentID ID                `json:"assignment_id"`
	PatientID    ID                `json:"patient_id"`
	CompletedAt  time.Time         `json:"completed_at"`
	Comment      string            `json:"comment"`
	Result       string            `json:"result"`
	Intake       *MedicationIntake `json:"intake,omitempty"`
	SideEffects  *SideEffectReport `json:"side_effects,omitempty"`
}

func NewCompletion(
	id ID,
	assignment Assignment,
	patientID ID,
	now time.Time,
) (*Completion, error) {
	if !id.Valid() || !patientID.Valid() || assignment.PatientID != patientID {
		return nil, ErrAssignmentMismatch
	}
	if assignment.Status != AssignmentActive || !assignment.Period.Contains(now) {
		return nil, fmt.Errorf("%w: assignment is not currently active", ErrInvalidInput)
	}
	return &Completion{
		ID: id, AssignmentID: assignment.ID,
		PatientID: patientID, CompletedAt: now,
	}, nil
}

// ---------- Опросники ----------

type AnswerType string

const (
	AnswerSingleChoice AnswerType = "SINGLE_CHOICE"
	AnswerMultiple     AnswerType = "MULTIPLE_CHOICE"
	AnswerScale        AnswerType = "SCALE"
	AnswerFreeText     AnswerType = "FREE_TEXT"
)

type ResultCategory string

const (
	ResultLow      ResultCategory = "LOW"
	ResultModerate ResultCategory = "MODERATE"
	ResultHigh     ResultCategory = "HIGH"
	ResultOther    ResultCategory = "OTHER"
)

type Question struct {
	ID           ID                 `json:"question_id"`
	Text         string             `json:"text"`
	AnswerType   AnswerType         `json:"answer_type"`
	Options      []string           `json:"options"`
	ScoreMapping map[string]float64 `json:"score_mapping"`
}

// ScoringRule — правило подсчёта конкретной версии опросника.
// Порог низкого результата хранится в QuestionnaireVersion, потому что
// результаты разных опросников не сравниваются по общей шкале.
type ScoringRule struct {
	Description string `json:"description"`
}

func (r ScoringRule) Calculate(answers []Answer) float64 {
	var total float64
	for _, answer := range answers {
		total += answer.Score
	}
	return total
}

func (r ScoringRule) Classify(score, lowScoreThreshold float64) ResultCategory {
	if score <= lowScoreThreshold {
		return ResultLow
	}
	return ResultOther
}

type QuestionnaireVersion struct {
	ID                ID          `json:"questionnaire_version_id"`
	Version           string      `json:"version"`
	ScoringRule       ScoringRule `json:"scoring_rule"`
	Questions         []Question  `json:"questions"`
	LowScoreThreshold float64     `json:"low_score_threshold"`
	CreatedAt         time.Time   `json:"created_at"`
	IsActive          bool        `json:"is_active"`
}

// Questionnaire — агрегат: опросник и его версии.
type Questionnaire struct {
	ID       ID                     `json:"questionnaire_id"`
	Name     string                 `json:"name"`
	IsActive bool                   `json:"is_active"`
	Versions []QuestionnaireVersion `json:"versions"`
}

// ScoreMapping фиксируется вместе с версией и после публикации не меняется.
type Answer struct {
	QuestionID ID      `json:"question_id"`
	Value      string  `json:"value"`
	Score      float64 `json:"score"`
}

type QuestionnaireAttempt struct {
	ID                     ID             `json:"attempt_id"`
	QuestionnaireID        ID             `json:"questionnaire_id"`
	QuestionnaireVersionID ID             `json:"questionnaire_version_id"`
	PatientID              ID             `json:"patient_id"`
	AssignmentID           ID             `json:"assignment_id,omitempty"`
	Answers                []Answer       `json:"answers"`
	Score                  float64        `json:"score"`
	Category               ResultCategory `json:"result_category"`
	CompletedAt            time.Time      `json:"completed_at"`
}

func (a QuestionnaireAttempt) ValidateAgainst(version QuestionnaireVersion) error {
	if a.QuestionnaireVersionID != version.ID {
		return fmt.Errorf("%w: attempt version mismatch", ErrInvalidInput)
	}
	if len(a.Answers) != len(version.Questions) {
		return fmt.Errorf("%w: incomplete questionnaire", ErrInvalidInput)
	}

	expected := make(map[ID]struct{}, len(version.Questions))
	for _, question := range version.Questions {
		expected[question.ID] = struct{}{}
	}
	for _, answer := range a.Answers {
		if _, ok := expected[answer.QuestionID]; !ok {
			return fmt.Errorf("%w: answer for unknown question %q", ErrInvalidInput, answer.QuestionID)
		}
		delete(expected, answer.QuestionID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("%w: not every question is answered", ErrInvalidInput)
	}
	return nil
}

// ---------- Дневник и отметки состояния ----------

type JournalEntry struct {
	ID               ID        `json:"entry_id"`
	PatientID        ID        `json:"patient_id"`
	Text             string    `json:"text"`
	ReflectionPrompt string    `json:"reflection_prompt"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MoodAnswer struct {
	Question string `json:"question"`
	Value    string `json:"value"`
}

type MoodCheckIn struct {
	ID        ID           `json:"check_in_id"`
	PatientID ID           `json:"patient_id"`
	Answers   []MoodAnswer `json:"answers"`
	Comment   string       `json:"comment"`
	CreatedAt time.Time    `json:"created_at"`
}

func (m MoodCheckIn) Validate() error {
	if !m.ID.Valid() || !m.PatientID.Valid() || len(m.Answers) == 0 {
		return fmt.Errorf("%w: check-in requires an ID, patient, and answers", ErrInvalidInput)
	}
	return nil
}

// ---------- Статус и уведомления ----------

type StatusColor string

const (
	StatusGreen  StatusColor = "GREEN"
	StatusYellow StatusColor = "YELLOW"
	StatusRed    StatusColor = "RED"
)

type StatusReasonKind string

const (
	ReasonMissedMedication  StatusReasonKind = "MISSED_MEDICATIONS"
	ReasonLowQuestionnaires StatusReasonKind = "STABLY_LOW_QUESTIONNAIRE_RESULTS"
)

type StatusReason struct {
	Kind          StatusReasonKind `json:"kind"`
	Description   string           `json:"description"`
	ObservedValue string           `json:"observed_value"`
	Period        Period           `json:"period"`
}

type PatientStatus struct {
	Color        StatusColor    `json:"color"`
	Reasons      []StatusReason `json:"reasons"`
	CalculatedAt time.Time      `json:"calculated_at"`
}

// MedicationAdherence — результат анализа пропусков приёма лекарств.
type MedicationAdherence struct {
	MissedCount int    `json:"missed_count"`
	Period      Period `json:"period"`
}

// QuestionnaireTrend — результат анализа последовательных низких результатов
// в рамках одного опросника и его версии.
type QuestionnaireTrend struct {
	QuestionnaireID        ID      `json:"questionnaire_id"`
	QuestionnaireVersionID ID      `json:"questionnaire_version_id"`
	ConsecutiveLowResults  int     `json:"consecutive_low_results"`
	Threshold              float64 `json:"threshold"`
}

// StatusPolicy — доменный сервис. Здесь намеренно нет медицинского диагноза
// или автоматической оценки экстренности.
type StatusPolicy interface {
	Evaluate(patientID ID, now time.Time) (PatientStatus, error)
}

type NotificationType string

const (
	NotificationStatusChanged NotificationType = "PATIENT_STATUS_CHANGED"
	NotificationSideEffect    NotificationType = "MEDICATION_SIDE_EFFECT_REPORTED"
	NotificationReminder      NotificationType = "ASSIGNMENT_REMINDER"
	NotificationQuestionnaire NotificationType = "QUESTIONNAIRE_COMPLETED"
)

type Notification struct {
	ID          ID               `json:"notification_id"`
	RecipientID ID               `json:"recipient_id"`
	PatientID   ID               `json:"patient_id"`
	Type        NotificationType `json:"type"`
	Message     string           `json:"message"`
	CreatedAt   time.Time        `json:"created_at"`
	ReadAt      *time.Time       `json:"read_at"`
}

// ---------- Поддержка и геймификация ----------

type SupportContentType string

const (
	ContentEncouragement SupportContentType = "ENCOURAGEMENT"
	ContentReflection    SupportContentType = "REFLECTION"
	ContentContactDoctor SupportContentType = "CONTACT_DOCTOR_PROMPT"
)

type SupportContent struct {
	ID          ID                 `json:"content_id"`
	Trigger     string             `json:"trigger"`
	Message     string             `json:"message"`
	ContentType SupportContentType `json:"content_type"`
}

type ForestActivity struct {
	ActivityType string    `json:"activity_type"`
	OccurredAt   time.Time `json:"occurred_at"`
	Points       int       `json:"points"`
}

type ForestState struct {
	ID          ID               `json:"forest_id"`
	PatientID   ID               `json:"patient_id"`
	GrowthLevel int              `json:"growth_level"`
	Activities  []ForestActivity `json:"activities"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func (f *ForestState) ApplyActivity(activity ForestActivity) error {
	if !f.PatientID.Valid() || activity.Points < 0 {
		return fmt.Errorf("%w: invalid forest activity", ErrInvalidInput)
	}
	// Повторное открытие той же записи не начисляет прогресс снова.
	for _, existing := range f.Activities {
		if existing.ActivityType == activity.ActivityType &&
			existing.OccurredAt.Equal(activity.OccurredAt) {
			return nil
		}
	}
	f.Activities = append(f.Activities, activity)
	f.GrowthLevel += activity.Points
	f.UpdatedAt = activity.OccurredAt
	return nil
}

// ---------- Справочники ----------

// MedicationGroup — группа препаратов из справочника.
type MedicationGroup struct {
	ID          ID     `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Medication — препарат из справочника. Приложение не назначает
// препарат само: врач выбирает его из справочника.
type Medication struct {
	ID              ID     `json:"medication_id"`
	Name            string `json:"name"`
	GroupID         ID     `json:"group_id"`
	InstructionText string `json:"instruction_text"`
}
