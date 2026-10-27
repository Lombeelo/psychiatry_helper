// internal/domain/model.go
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrInactiveRelation   = errors.New("active care relationship required")
	ErrAssignmentMismatch = errors.New("assignment does not belong to patient")
	ErrAlreadyEnded       = errors.New("care relationship already ended")
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

type UserCode struct {
	Value string
	Role  AccountRole
}

func (c UserCode) Validate() error {
	if strings.TrimSpace(c.Value) == "" {
		return fmt.Errorf("%w: empty user code", ErrInvalidInput)
	}
	switch c.Role {
	case RolePatient, RoleDoctor, RoleAdmin:
		return nil
	default:
		return fmt.Errorf("%w: unknown account role", ErrInvalidInput)
	}
}

type PersonalData struct {
	DisplayName string
	DateOfBirth time.Time
}

func (p PersonalData) Validate() error {
	if strings.TrimSpace(p.DisplayName) == "" {
		return fmt.Errorf("%w: display name is required", ErrInvalidInput)
	}
	if p.DateOfBirth.After(time.Now()) {
		return fmt.Errorf("%w: date of birth is in the future", ErrInvalidInput)
	}
	return nil
}

// ---------- Учётные записи и связь ----------

type Account struct {
	ID        ID
	Code      UserCode
	Personal  PersonalData
	CreatedAt time.Time
}

type Consent struct {
	Accepted      bool
	PolicyVersion string
	AcceptedAt    *time.Time
}

func (c Consent) Validate() error {
	if c.Accepted && (c.AcceptedAt == nil || strings.TrimSpace(c.PolicyVersion) == "") {
		return fmt.Errorf("%w: accepted consent requires version and timestamp", ErrInvalidInput)
	}
	return nil
}

type Patient struct {
	Account
	Notes   string
	Consent Consent
}

type Doctor struct {
	Account
	Specialty string
}

type RelationshipStatus string

const (
	RelationshipActive RelationshipStatus = "ACTIVE"
	RelationshipEnded  RelationshipStatus = "ENDED"
)

// CareRelationship — агрегатная граница связи врача и пациента.
type CareRelationship struct {
	ID        ID
	PatientID ID
	DoctorID  ID
	CreatedAt time.Time
	EndedAt   *time.Time
	Status    RelationshipStatus
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
	StartsAt time.Time
	EndsAt   time.Time
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
	Frequency  string
	DaysOfWeek []time.Weekday
	TimesOfDay []string // формат HH:MM; часовой пояс — пациента
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
	Amount float64
	Unit   string
}

func (d Dosage) Validate() error {
	if d.Amount <= 0 || strings.TrimSpace(d.Unit) == "" {
		return fmt.Errorf("%w: invalid dosage", ErrInvalidInput)
	}
	return nil
}

type MedicationDetails struct {
	MedicationID ID
	Dosage       Dosage
	Instructions string
}

func (d MedicationDetails) Kind() AssignmentType { return AssignmentMedication }
func (d MedicationDetails) Validate() error {
	if !d.MedicationID.Valid() {
		return fmt.Errorf("%w: medication ID is required", ErrInvalidInput)
	}
	return d.Dosage.Validate()
}

type QuestionnaireDetails struct {
	QuestionnaireID        ID
	QuestionnaireVersionID ID
}

func (d QuestionnaireDetails) Kind() AssignmentType { return AssignmentQuestionnaire }
func (d QuestionnaireDetails) Validate() error {
	if !d.QuestionnaireID.Valid() || !d.QuestionnaireVersionID.Valid() {
		return fmt.Errorf("%w: questionnaire and version IDs are required", ErrInvalidInput)
	}
	return nil
}

type AppCheckInDetails struct{ Instructions string }

func (d AppCheckInDetails) Kind() AssignmentType { return AssignmentAppCheckIn }
func (d AppCheckInDetails) Validate() error      { return nil }

type JournalTaskDetails struct{ Prompt string }

func (d JournalTaskDetails) Kind() AssignmentType { return AssignmentJournalTask }
func (d JournalTaskDetails) Validate() error      { return nil }

// Assignment — агрегат. Детали задаются типизированным интерфейсом.
type Assignment struct {
	ID        ID
	PatientID ID
	DoctorID  ID
	Type      AssignmentType
	Title     string
	Period    Period
	Schedule  Schedule
	Status    AssignmentStatus
	Details   AssignmentDetails
	CreatedAt time.Time
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
	TakenAt         time.Time
	ReportedAsTaken bool
	DosageTaken     string
}

type SideEffectReport struct {
	Effects    []string
	Comment    string
	ReportedAt time.Time
}

// Completion — отдельный агрегат, чтобы выполнения можно было хранить независимо.
type Completion struct {
	ID           ID
	AssignmentID ID
	PatientID    ID
	CompletedAt  time.Time
	Comment      string
	Result       string
	Intake       *MedicationIntake
	SideEffects  *SideEffectReport
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
	ID           ID
	Text         string
	AnswerType   AnswerType
	Options      []string
	ScoreMapping map[string]float64
}

type QuestionnaireVersion struct {
	ID                ID
	Version           string
	Questions         []Question
	LowScoreThreshold float64
	CreatedAt         time.Time
	IsActive          bool
}

// ScoringRule должен быть версионированным и фиксированным после публикации.
// Конкретные правила подсчёта зависят от выбранного опросника.
type Answer struct {
	QuestionID ID
	Value      string
	Score      float64
}

type QuestionnaireAttempt struct {
	ID                     ID
	QuestionnaireID        ID
	QuestionnaireVersionID ID
	PatientID              ID
	AssignmentID           ID
	Answers                []Answer
	Score                  float64
	Category               ResultCategory
	CompletedAt            time.Time
}

func (a QuestionnaireAttempt) ValidateAgainst(version QuestionnaireVersion) error {
	if a.QuestionnaireVersionID != version.ID {
		return fmt.Errorf("%w: attempt version mismatch", ErrInvalidInput)
	}
	if len(a.Answers) != len(version.Questions) {
		return fmt.Errorf("%w: incomplete questionnaire", ErrInvalidInput)
	}
	return nil
}

// ---------- Дневник и отметки состояния ----------

type JournalEntry struct {
	ID               ID
	PatientID        ID
	Text             string
	ReflectionPrompt string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MoodAnswer struct {
	Question string
	Value    string
}

type MoodCheckIn struct {
	ID        ID
	PatientID ID
	Answers   []MoodAnswer
	Comment   string
	CreatedAt time.Time
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
	Kind          StatusReasonKind
	Description   string
	ObservedValue string
	Period        Period
}

type PatientStatus struct {
	Color        StatusColor
	Reasons      []StatusReason
	CalculatedAt time.Time
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
	ID          ID
	RecipientID ID
	PatientID   ID
	Type        NotificationType
	Message     string
	CreatedAt   time.Time
	ReadAt      *time.Time
}

// ---------- Поддержка и геймификация ----------

type SupportContentType string

const (
	ContentEncouragement SupportContentType = "ENCOURAGEMENT"
	ContentReflection    SupportContentType = "REFLECTION"
	ContentContactDoctor SupportContentType = "CONTACT_DOCTOR_PROMPT"
)

type SupportContent struct {
	ID          ID
	Trigger     string
	Message     string
	ContentType SupportContentType
}

type ForestActivity struct {
	ActivityType string
	OccurredAt   time.Time
	Points       int
}

type ForestState struct {
	ID          ID
	PatientID   ID
	GrowthLevel int
	Activities  []ForestActivity
	UpdatedAt   time.Time
}

func (f *ForestState) ApplyActivity(activity ForestActivity) error {
	if !f.PatientID.Valid() || activity.Points < 0 {
		return fmt.Errorf("%w: invalid forest activity", ErrInvalidInput)
	}
	f.Activities = append(f.Activities, activity)
	f.GrowthLevel += activity.Points
	f.UpdatedAt = activity.OccurredAt
	return nil
}
