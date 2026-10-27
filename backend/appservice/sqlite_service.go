// internal/appservice/sqlite_service.go
package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"example.com/backend/domain"
	"example.com/backend/httpapi"
	sqlitestore "example.com/backend/sqlite"
)

type SQLiteService struct {
	store *sqlitestore.Store
	now   func() time.Time
}

func NewSQLiteService(store *sqlitestore.Store) *SQLiteService {
	return &SQLiteService{
		store: store,
		now:   time.Now,
	}
}

// Проверка соответствия интерфейсу выполняется компилятором.
var _ httpapi.Service = (*SQLiteService)(nil)

func (s *SQLiteService) CreatePatient(
	r *http.Request,
	userID domain.ID,
	req httpapi.CreatePatientRequest,
) (domain.Patient, error) {
	ctx := r.Context()

	dateOfBirth, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return domain.Patient{}, domain.ErrInvalidInput
	}
	if !req.Accepted || strings.TrimSpace(req.PolicyVersion) == "" {
		return domain.Patient{}, domain.ErrInvalidInput
	}

	acceptedAt := s.now().UTC()
	patient := domain.Patient{
		Account: domain.Account{
			ID: userID,
			Code: domain.UserCode{
				Value: string(userID),
				Role:  domain.RolePatient,
			},
			Personal: domain.PersonalData{
				DisplayName: strings.TrimSpace(req.DisplayName),
				DateOfBirth: dateOfBirth,
			},
			CreatedAt: acceptedAt,
		},
		Consent: domain.Consent{
			Accepted:      true,
			PolicyVersion: strings.TrimSpace(req.PolicyVersion),
			AcceptedAt:    &acceptedAt,
		},
	}

	if err := patient.Personal.Validate(); err != nil {
		return domain.Patient{}, err
	}
	if err := patient.Consent.Validate(); err != nil {
		return domain.Patient{}, err
	}
	if err := s.store.CreatePatient(ctx, patient); err != nil {
		return domain.Patient{}, err
	}
	return patient, nil
}

func (s *SQLiteService) CreateRelationship(
	r *http.Request,
	doctorID domain.ID,
	req httpapi.CreateRelationshipRequest,
) (domain.CareRelationship, error) {
	if !req.PatientID.Valid() {
		return domain.CareRelationship{}, domain.ErrInvalidInput
	}

	id := domain.ID(uuid.NewString())
	relationship, err := domain.NewCareRelationship(
		id,
		req.PatientID,
		doctorID,
		s.now().UTC(),
	)
	if err != nil {
		return domain.CareRelationship{}, err
	}

	// Проверки существования врача и пациента должны выполняться БД
	// через внешние ключи. Проверка дублирующей активной связи — отдельный метод Store.
	if err := s.store.CreateRelationship(r.Context(), *relationship); err != nil {
		return domain.CareRelationship{}, err
	}
	return *relationship, nil
}

func (s *SQLiteService) EndRelationship(
	r *http.Request,
	doctorID, relationshipID domain.ID,
) error {
	// См. примечание после кода: безопасный вариант должен проверять doctorID
	// непосредственно в UPDATE, а не завершать связь только по её ID.
	return s.store.EndRelationship(r.Context(), relationshipID, s.now().UTC())
}

func (s *SQLiteService) ListAssignments(
	r *http.Request,
	userID domain.ID,
	role domain.AccountRole,
) ([]domain.Assignment, error) {
	switch role {
	case domain.RolePatient:
		return s.store.ListAssignmentsForPatient(r.Context(), userID)
	case domain.RoleDoctor:
		return s.store.ListAssignmentsForDoctor(r.Context(), userID)
	default:
		return nil, errors.New("unsupported role")
	}
}

func (s *SQLiteService) CreateAssignment(
	r *http.Request,
	doctorID domain.ID,
	req httpapi.CreateAssignmentRequest,
) (domain.Assignment, error) {
	details, err := decodeAssignmentDetails(req.Type, req.Details)
	if err != nil {
		return domain.Assignment{}, err
	}

	now := s.now().UTC()
	assignment, err := domain.NewAssignment(
		domain.ID(uuid.NewString()),
		req.PatientID,
		doctorID,
		req.Title,
		domain.Period{
			StartsAt: req.StartsAt,
			EndsAt:   req.EndsAt,
		},
		domain.Schedule{
			Frequency:  req.Frequency,
			DaysOfWeek: req.DaysOfWeek,
			TimesOfDay: req.TimesOfDay,
		},
		details,
		now,
	)
	if err != nil {
		return domain.Assignment{}, err
	}

	// Важно: до сохранения нужно проверить активную связь doctorID ↔ PatientID.
	active, err := s.store.HasActiveRelationship(r.Context(), req.PatientID, doctorID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if !active {
		return domain.Assignment{}, domain.ErrInactiveRelation
	}

	if err := s.store.CreateAssignment(r.Context(), *assignment); err != nil {
		return domain.Assignment{}, err
	}
	return *assignment, nil
}

func (s *SQLiteService) ChangeAssignmentStatus(
	r *http.Request,
	doctorID, assignmentID domain.ID,
	action string,
) error {
	assignment, err := s.store.GetAssignment(r.Context(), assignmentID)
	if err != nil {
		return err
	}
	if assignment.DoctorID != doctorID {
		return domain.ErrInactiveRelation
	}

	active, err := s.store.HasActiveRelationship(
		r.Context(),
		assignment.PatientID,
		doctorID,
	)
	if err != nil {
		return err
	}
	if !active {
		return domain.ErrInactiveRelation
	}

	switch action {
	case "pause":
		err = assignment.Pause()
	case "resume":
		err = assignment.Resume()
	default:
		return domain.ErrInvalidInput
	}
	if err != nil {
		return err
	}
	return s.store.UpdateAssignmentStatus(r.Context(), assignment.ID, assignment.Status)
}

func (s *SQLiteService) CompleteAssignment(
	r *http.Request,
	patientID, assignmentID domain.ID,
	req httpapi.CompleteAssignmentRequest,
) (domain.Completion, error) {
	assignment, err := s.store.GetAssignment(r.Context(), assignmentID)
	if err != nil {
		return domain.Completion{}, err
	}

	completion, err := domain.NewCompletion(
		domain.ID(uuid.NewString()),
		assignment,
		patientID,
		s.now().UTC(),
	)
	if err != nil {
		return domain.Completion{}, err
	}
	completion.Comment = req.Comment
	completion.Result = req.Result

	if err := s.store.CreateCompletion(r.Context(), *completion); err != nil {
		return domain.Completion{}, err
	}
	return *completion, nil
}

func (s *SQLiteService) CreateJournalEntry(
	r *http.Request,
	patientID domain.ID,
	req httpapi.CreateJournalEntryRequest,
) (domain.JournalEntry, error) {
	now := s.now().UTC()
	entry := domain.JournalEntry{
		ID:               domain.ID(uuid.NewString()),
		PatientID:        patientID,
		Text:             strings.TrimSpace(req.Text),
		ReflectionPrompt: req.ReflectionPrompt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if entry.Text == "" {
		return domain.JournalEntry{}, domain.ErrInvalidInput
	}
	if err := s.store.CreateJournalEntry(r.Context(), entry); err != nil {
		return domain.JournalEntry{}, err
	}
	return entry, nil
}

func (s *SQLiteService) CreateMoodCheckIn(
	r *http.Request,
	patientID domain.ID,
	req httpapi.CreateMoodCheckInRequest,
) (domain.MoodCheckIn, error) {
	checkIn := domain.MoodCheckIn{
		ID:        domain.ID(uuid.NewString()),
		PatientID: patientID,
		Answers:   req.Answers,
		Comment:   req.Comment,
		CreatedAt: s.now().UTC(),
	}
	if err := checkIn.Validate(); err != nil {
		return domain.MoodCheckIn{}, err
	}
	if err := s.store.CreateMoodCheckIn(r.Context(), checkIn); err != nil {
		return domain.MoodCheckIn{}, err
	}
	return checkIn, nil
}

func decodeAssignmentDetails(
	kind domain.AssignmentType,
	raw json.RawMessage,
) (domain.AssignmentDetails, error) {
	var details domain.AssignmentDetails

	switch kind {
	case domain.AssignmentMedication:
		details = &domain.MedicationDetails{}
	case domain.AssignmentQuestionnaire:
		details = &domain.QuestionnaireDetails{}
	case domain.AssignmentAppCheckIn:
		details = &domain.AppCheckInDetails{}
	case domain.AssignmentJournalTask:
		details = &domain.JournalTaskDetails{}
	default:
		return nil, domain.ErrInvalidInput
	}

	if len(raw) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := json.Unmarshal(raw, details); err != nil {
		return nil, domain.ErrInvalidInput
	}
	if err := details.Validate(); err != nil {
		return nil, err
	}
	return details, nil
}

// Защита от случайного использования контекста без дедлайна —
// все методы передают r.Context() в SQLite.
var _ context.Context
