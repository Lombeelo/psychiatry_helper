package appservice

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/backend/domain"
	"example.com/backend/httpapi"
	"example.com/backend/security"
)

// Authenticate проверяет логин и пароль.
func (s *SQLiteService) Authenticate(
	r *http.Request,
	login, password string,
) (domain.Account, bool, string, error) {
	account, err := s.store.LoginAccountByLogin(
		r.Context(),
		strings.TrimSpace(strings.ToLower(login)),
	)
	if err != nil {
		return domain.Account{}, false, "", domain.ErrInvalidCredentials
	}
	if !security.VerifyPassword(password, account.PasswordHash, account.PasswordSalt) {
		return domain.Account{}, false, "", domain.ErrInvalidCredentials
	}
	return account.Account, account.ConsentAccepted, account.Notes, nil
}

// AcceptConsent фиксирует согласие пациента с политикой.
func (s *SQLiteService) AcceptConsent(
	r *http.Request,
	patientID domain.ID,
	policyVersion string,
) error {
	if strings.TrimSpace(policyVersion) == "" {
		policyVersion = "v1"
	}
	return s.store.AcceptConsent(
		r.Context(),
		patientID,
		strings.TrimSpace(policyVersion),
		s.now().UTC(),
	)
}

// ListPatients возвращает пациентов врача с текущим статусом.
func (s *SQLiteService) ListPatients(
	r *http.Request,
	doctorID domain.ID,
) ([]httpapi.PatientSummary, error) {
	patients, err := s.store.ListPatientsForDoctor(r.Context(), doctorID)
	if err != nil {
		return nil, err
	}

	result := make([]httpapi.PatientSummary, 0, len(patients))
	for _, patient := range patients {
		status, err := s.patientStatus(r, patient.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, httpapi.PatientSummary{
			PatientID:   string(patient.ID),
			Code:        patient.Code.Value,
			DisplayName: patient.Personal.DisplayName,
			Notes:       patient.Profile.Notes,
			Status:      status,
		})
	}
	return result, nil
}

// GetPatientDetails собирает карточку пациента для врача.
func (s *SQLiteService) GetPatientDetails(
	r *http.Request,
	doctorID, patientID domain.ID,
) (httpapi.PatientDetails, error) {
	active, err := s.store.HasActiveRelationship(r.Context(), patientID, doctorID)
	if err != nil {
		return httpapi.PatientDetails{}, err
	}
	if !active {
		return httpapi.PatientDetails{}, domain.ErrInactiveRelation
	}

	patient, err := s.store.GetPatientByID(r.Context(), patientID)
	if err != nil {
		return httpapi.PatientDetails{}, notFoundAsDomainError(err)
	}
	status, err := s.patientStatus(r, patientID)
	if err != nil {
		return httpapi.PatientDetails{}, err
	}
	assignments, err := s.store.ListAssignmentsForDoctorAndPatient(
		r.Context(), doctorID, patientID,
	)
	if err != nil {
		return httpapi.PatientDetails{}, err
	}
	journal, err := s.store.ListJournalEntriesForPatient(r.Context(), patientID)
	if err != nil {
		return httpapi.PatientDetails{}, err
	}
	mood, err := s.store.ListMoodCheckInsForPatient(r.Context(), patientID)
	if err != nil {
		return httpapi.PatientDetails{}, err
	}
	if assignments == nil {
		assignments = []domain.Assignment{}
	}
	if journal == nil {
		journal = []domain.JournalEntry{}
	}
	if mood == nil {
		mood = []domain.MoodCheckIn{}
	}

	return httpapi.PatientDetails{
		Patient: httpapi.PatientSummary{
			PatientID:   string(patient.ID),
			Code:        patient.Code.Value,
			DisplayName: patient.Personal.DisplayName,
			Notes:       patient.Profile.Notes,
			Status:      status,
		},
		Assignments:    assignments,
		JournalEntries: journal,
		MoodCheckIns:   mood,
	}, nil
}

// MyDoctor возвращает врача, привязанного к пациенту.
func (s *SQLiteService) MyDoctor(
	r *http.Request,
	patientID domain.ID,
) (httpapi.DoctorSummary, error) {
	doctor, err := s.store.DoctorForPatient(r.Context(), patientID)
	if err != nil {
		return httpapi.DoctorSummary{}, notFoundAsDomainError(err)
	}
	return httpapi.DoctorSummary{
		DoctorID:    string(doctor.ID),
		Code:        doctor.Code.Value,
		DisplayName: doctor.Personal.DisplayName,
		Specialty:   doctor.Profile.Specialty,
	}, nil
}

func (s *SQLiteService) ListJournalEntries(
	r *http.Request,
	patientID domain.ID,
) ([]domain.JournalEntry, error) {
	entries, err := s.store.ListJournalEntriesForPatient(r.Context(), patientID)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []domain.JournalEntry{}
	}
	return entries, nil
}

// UpdateAnamnesis сохраняет первичный сбор анамнеза.
func (s *SQLiteService) UpdateAnamnesis(
	r *http.Request,
	patientID domain.ID,
	notes string,
) error {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return fmt.Errorf("%w: anamnesis must not be empty", domain.ErrInvalidInput)
	}
	return s.store.UpdatePatientNotes(r.Context(), patientID, notes)
}

// GetForest рассчитывает состояние личного леса по действиям пациента.
func (s *SQLiteService) GetForest(
	r *http.Request,
	patientID domain.ID,
) (domain.ForestState, error) {
	entries, err := s.store.ListJournalEntriesForPatient(r.Context(), patientID)
	if err != nil {
		return domain.ForestState{}, err
	}
	completions, err := s.store.ListCompletionsForPatient(r.Context(), patientID)
	if err != nil {
		return domain.ForestState{}, err
	}

	now := s.now().UTC()
	forest := domain.ForestState{
		ID:          domain.ID("forest-" + string(patientID)),
		PatientID:   patientID,
		GrowthLevel: len(entries)*2 + len(completions)*5,
		UpdatedAt:   now,
	}
	for _, entry := range entries {
		forest.Activities = append(forest.Activities, domain.ForestActivity{
			ActivityType: "JOURNAL_ENTRY",
			OccurredAt:   entry.CreatedAt,
			Points:       2,
		})
	}
	for _, completion := range completions {
		forest.Activities = append(forest.Activities, domain.ForestActivity{
			ActivityType: "ASSIGNMENT_COMPLETED",
			OccurredAt:   completion.CompletedAt,
			Points:       5,
		})
	}
	sort.Slice(forest.Activities, func(i, j int) bool {
		return forest.Activities[i].OccurredAt.After(forest.Activities[j].OccurredAt)
	})
	if len(forest.Activities) > 20 {
		forest.Activities = forest.Activities[:20]
	}
	if len(forest.Activities) > 0 {
		forest.UpdatedAt = forest.Activities[0].OccurredAt
	}
	return forest, nil
}

func (s *SQLiteService) ListNotifications(
	r *http.Request,
	recipientID domain.ID,
) ([]domain.Notification, error) {
	notifications, err := s.store.ListNotificationsForRecipient(
		r.Context(), recipientID,
	)
	if err != nil {
		return nil, err
	}
	if notifications == nil {
		notifications = []domain.Notification{}
	}
	return notifications, nil
}

func (s *SQLiteService) MarkNotificationRead(
	r *http.Request,
	recipientID, notificationID domain.ID,
) error {
	return s.store.MarkNotificationRead(
		r.Context(), recipientID, notificationID, s.now().UTC(),
	)
}

func (s *SQLiteService) ListMedicationGroups(
	r *http.Request,
) ([]httpapi.MedicationGroup, error) {
	groups, err := s.store.ListMedicationGroups(r.Context())
	if err != nil {
		return nil, err
	}

	result := make([]httpapi.MedicationGroup, 0, len(groups))
	for _, group := range groups {
		medications := make([]httpapi.Medication, 0, len(group.Medications))
		for _, medication := range group.Medications {
			medications = append(medications, httpapi.Medication{
				MedicationID:    string(medication.ID),
				Name:            medication.Name,
				InstructionText: medication.InstructionText,
			})
		}
		result = append(result, httpapi.MedicationGroup{
			GroupID:     string(group.Group.ID),
			Name:        group.Group.Name,
			Description: group.Group.Description,
			Medications: medications,
		})
	}
	return result, nil
}

func (s *SQLiteService) ListSupportContents(
	r *http.Request,
) ([]domain.SupportContent, error) {
	contents, err := s.store.ListSupportContents(r.Context())
	if err != nil {
		return nil, err
	}
	if contents == nil {
		contents = []domain.SupportContent{}
	}
	return contents, nil
}

// patientStatus рассчитывает светофор по пропускам приёма лекарств.
func (s *SQLiteService) patientStatus(
	r *http.Request,
	patientID domain.ID,
) (httpapi.StatusSummary, error) {
	now := s.now().UTC()
	lookbackStart := now.Add(-7 * 24 * time.Hour)

	assignments, err := s.store.ListAssignmentsForPatient(r.Context(), patientID)
	if err != nil {
		return httpapi.StatusSummary{}, err
	}
	completions, err := s.store.ListCompletionsForPatient(r.Context(), patientID)
	if err != nil {
		return httpapi.StatusSummary{}, err
	}

	completionsByAssignment := make(map[domain.ID]int)
	for _, completion := range completions {
		if completion.Intake == nil {
			continue
		}
		if completion.CompletedAt.Before(lookbackStart) || completion.CompletedAt.After(now) {
			continue
		}
		completionsByAssignment[completion.AssignmentID]++
	}

	expected := 0
	missed := 0
	for _, assignment := range assignments {
		if assignment.Type != domain.AssignmentMedication ||
			assignment.Status != domain.AssignmentActive {
			continue
		}
		planned := expectedDoses(assignment, lookbackStart, now)
		if planned == 0 {
			continue
		}
		actual := completionsByAssignment[assignment.ID]
		if actual > planned {
			actual = planned
		}
		expected += planned
		missed += planned - actual
	}

	status := httpapi.StatusSummary{
		CalculatedAt:   now,
		Reasons:        []httpapi.StatusReason{},
		DataSufficient: expected > 0,
		Color:          string(domain.StatusGreen),
	}
	if expected == 0 {
		return status, nil
	}

	switch {
	case missed >= 5:
		status.Color = string(domain.StatusRed)
	case missed >= 2:
		status.Color = string(domain.StatusYellow)
	default:
		status.Color = string(domain.StatusGreen)
	}

	description := "Пропусков приёма за последние 7 дней нет"
	if missed > 0 {
		description = fmt.Sprintf(
			"Пропущено приёмов: %d за последние 7 дней",
			missed,
		)
	}
	status.Reasons = append(status.Reasons, httpapi.StatusReason{
		Kind:          string(domain.ReasonMissedMedication),
		Description:   description,
		ObservedValue: strconv.Itoa(missed),
	})
	return status, nil
}

// expectedDoses считает запланированные приёмы по расписанию за период.
func expectedDoses(assignment domain.Assignment, start, now time.Time) int {
	periodStart := assignment.Period.StartsAt
	if periodStart.Before(start) {
		periodStart = start
	}
	if assignment.Period.EndsAt.Before(periodStart) || periodStart.After(now) {
		return 0
	}

	perDay := len(assignment.Schedule.TimesOfDay)
	if perDay == 0 {
		perDay = 1
	}
	day := time.Date(
		periodStart.Year(), periodStart.Month(), periodStart.Day(),
		0, 0, 0, 0, periodStart.Location(),
	)
	lastDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	total := 0
	for !day.After(lastDay) {
		if !day.Before(assignment.Period.StartsAt) &&
			!day.After(assignment.Period.EndsAt) &&
			weekdayMatches(day.Weekday(), assignment.Schedule.DaysOfWeek) {
			total += perDay
		}
		day = day.AddDate(0, 0, 1)
	}
	return total
}

func weekdayMatches(day time.Weekday, days []time.Weekday) bool {
	if len(days) == 0 {
		return true
	}
	for _, value := range days {
		if value == day {
			return true
		}
	}
	return false
}
