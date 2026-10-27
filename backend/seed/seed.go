// Package seed наполняет пустую базу демонстрационными данными.
package seed

import (
	"context"
	"fmt"
	"time"

	"example.com/backend/domain"
	"example.com/backend/security"
	sqlitestore "example.com/backend/sqlite"
)

const (
	DoctorLogin     = "doctor"
	DoctorPassword  = "doctor123"
	PatientLogin    = "patient"
	PatientPassword = "patient123"

	demoDoctorID  domain.ID = "demo-doctor"
	demoPatientID domain.ID = "demo-patient"
)

// Credentials — данные для входа, которые печатаются при старте сервера.
type Credentials struct {
	DoctorLogin     string
	DoctorPassword  string
	PatientLogin    string
	PatientPassword string
}

// EnsureDemoData создаёт демонстрационных врача и пациента, если база пуста.
func EnsureDemoData(ctx context.Context, store *sqlitestore.Store) (Credentials, error) {
	credentials := Credentials{
		DoctorLogin:     DoctorLogin,
		DoctorPassword:  DoctorPassword,
		PatientLogin:    PatientLogin,
		PatientPassword: PatientPassword,
	}

	exists, err := store.AccountExists(ctx, demoDoctorID)
	if err != nil {
		return credentials, err
	}
	if exists {
		return credentials, nil
	}

	now := time.Now().UTC()
	doctorHash, doctorSalt, err := security.HashPassword(DoctorPassword)
	if err != nil {
		return credentials, err
	}
	patientHash, patientSalt, err := security.HashPassword(PatientPassword)
	if err != nil {
		return credentials, err
	}

	doctor := domain.Doctor{
		Account: domain.Account{
			ID:   demoDoctorID,
			Code: domain.UserCode{Value: "VDR0001", Role: domain.RoleDoctor},
			Personal: domain.PersonalData{
				DisplayName: "Иванов Иван Иванович",
				DateOfBirth: time.Date(1980, time.May, 15, 0, 0, 0, 0, time.UTC),
			},
			CreatedAt: now.AddDate(0, 0, -30),
		},
		Profile: domain.DoctorProfile{Specialty: "Психотерапевт"},
	}
	if err := store.CreateDoctor(ctx, doctor); err != nil {
		return credentials, fmt.Errorf("seed doctor: %w", err)
	}
	if err := store.SetCredentials(
		ctx, doctor.ID, DoctorLogin, doctorHash, doctorSalt,
	); err != nil {
		return credentials, err
	}

	patient := domain.Patient{
		Account: domain.Account{
			ID:   demoPatientID,
			Code: domain.UserCode{Value: "PPAT001", Role: domain.RolePatient},
			Personal: domain.PersonalData{
				DisplayName: "Петрова Анна Сергеевна",
				DateOfBirth: time.Date(1995, time.March, 22, 0, 0, 0, 0, time.UTC),
			},
			CreatedAt: now.AddDate(0, 0, -30),
		},
		Profile: domain.PatientProfile{},
		Consent: domain.Consent{
			Accepted:      false,
			PolicyVersion: "",
		},
	}
	if err := store.CreatePatient(ctx, patient); err != nil {
		return credentials, fmt.Errorf("seed patient: %w", err)
	}
	if err := store.SetCredentials(
		ctx, patient.ID, PatientLogin, patientHash, patientSalt,
	); err != nil {
		return credentials, err
	}

	relationship, err := domain.NewCareRelationship(
		"demo-relationship",
		patient.ID,
		doctor.ID,
		now.AddDate(0, 0, -10),
	)
	if err != nil {
		return credentials, fmt.Errorf("seed relationship: %w", err)
	}
	if err := store.CreateRelationship(ctx, *relationship); err != nil {
		return credentials, fmt.Errorf("seed relationship: %w", err)
	}

	if err := seedMedications(ctx, store); err != nil {
		return credentials, err
	}
	if err := seedAssignments(ctx, store, now); err != nil {
		return credentials, err
	}
	if err := seedJournalAndMood(ctx, store, now); err != nil {
		return credentials, err
	}
	if err := seedNotifications(ctx, store, now); err != nil {
		return credentials, err
	}
	if err := seedSupportContents(ctx, store); err != nil {
		return credentials, err
	}

	return credentials, nil
}

func seedMedications(ctx context.Context, store *sqlitestore.Store) error {
	groups := []domain.MedicationGroup{
		{
			ID:          "group-ssri",
			Name:        "Антидепрессанты (СИОЗС)",
			Description: "Препараты первой линии при депрессивной симптоматике.",
		},
		{
			ID:          "group-anxiolytic",
			Name:        "Анксиолитики",
			Description: "Препараты для кратковременного снижения тревоги.",
		},
		{
			ID:          "group-mood",
			Name:        "Нормотимики",
			Description: "Стабилизаторы настроения.",
		},
	}
	for _, group := range groups {
		if err := store.CreateMedicationGroup(ctx, group); err != nil {
			return err
		}
	}

	medications := []domain.Medication{
		{
			ID:              "med-sertraline",
			Name:            "Сертралин",
			GroupID:         "group-ssri",
			InstructionText: "Принимать утром после еды. Дозу подбирает врач.",
		},
		{
			ID:              "med-escitalopram",
			Name:            "Эсциталопрам",
			GroupID:         "group-ssri",
			InstructionText: "Принимать один раз в сутки в одно и то же время.",
		},
		{
			ID:              "med-lorazepam",
			Name:            "Лоразепам",
			GroupID:         "group-anxiolytic",
			InstructionText: "Кратковременный приём. Не превышать назначенную дозу.",
		},
		{
			ID:              "med-lithium",
			Name:            "Лития карбонат",
			GroupID:         "group-mood",
			InstructionText: "Требует контроля концентрации в крови.",
		},
	}
	for _, medication := range medications {
		if err := store.CreateMedication(ctx, medication); err != nil {
			return err
		}
	}
	return nil
}

func seedAssignments(
	ctx context.Context,
	store *sqlitestore.Store,
	now time.Time,
) error {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startsAt := day.AddDate(0, 0, -7)
	endsAt := day.AddDate(0, 0, 23).Add(23 * time.Hour)

	medicationAssignment, err := domain.NewAssignment(
		"demo-assignment-medication",
		demoPatientID,
		demoDoctorID,
		"Сертралин 50 мг",
		domain.Period{StartsAt: startsAt, EndsAt: endsAt},
		domain.Schedule{Frequency: "DAILY", TimesOfDay: []string{"09:00"}},
		&domain.MedicationDetails{
			MedicationID: "med-sertraline",
			Dosage: domain.Dosage{
				Amount:       50,
				Unit:         "мг",
				Instructions: "После завтрака",
			},
			Instructions: "Принимать утром после еды",
		},
		now.AddDate(0, 0, -7),
	)
	if err != nil {
		return fmt.Errorf("seed medication assignment: %w", err)
	}
	if err := store.CreateAssignment(ctx, *medicationAssignment); err != nil {
		return err
	}

	checkInAssignment, err := domain.NewAssignment(
		"demo-assignment-checkin",
		demoPatientID,
		demoDoctorID,
		"Вечерняя отметка состояния",
		domain.Period{StartsAt: startsAt, EndsAt: endsAt},
		domain.Schedule{Frequency: "DAILY", TimesOfDay: []string{"20:00"}},
		&domain.AppCheckInDetails{Instructions: "Отметьте настроение и тревожность"},
		now.AddDate(0, 0, -7),
	)
	if err != nil {
		return fmt.Errorf("seed check-in assignment: %w", err)
	}
	if err := store.CreateAssignment(ctx, *checkInAssignment); err != nil {
		return err
	}

	journalAssignment, err := domain.NewAssignment(
		"demo-assignment-journal",
		demoPatientID,
		demoDoctorID,
		"Дневниковая практика",
		domain.Period{StartsAt: startsAt, EndsAt: endsAt},
		domain.Schedule{Frequency: "DAILY", TimesOfDay: []string{"21:00"}},
		&domain.JournalTaskDetails{Prompt: "Что сегодня помогло вам почувствовать себя лучше?"},
		now.AddDate(0, 0, -7),
	)
	if err != nil {
		return fmt.Errorf("seed journal assignment: %w", err)
	}
	if err := store.CreateAssignment(ctx, *journalAssignment); err != nil {
		return err
	}

	for index := 6; index >= 2; index-- {
		takenAt := day.AddDate(0, 0, -index).Add(9 * time.Hour)
		completion := domain.Completion{
			ID:           domain.ID(fmt.Sprintf("demo-completion-med-%d", index)),
			AssignmentID: medicationAssignment.ID,
			PatientID:    demoPatientID,
			CompletedAt:  takenAt,
			Comment:      "Приняла",
			Result:       "",
			Intake: &domain.MedicationIntake{
				TakenAt:         takenAt,
				ReportedAsTaken: true,
				DosageTaken:     "1 таблетка",
			},
		}
		if err := store.CreateCompletion(ctx, completion); err != nil {
			return err
		}
	}

	return nil
}

func seedJournalAndMood(
	ctx context.Context,
	store *sqlitestore.Store,
	now time.Time,
) error {
	entries := []domain.JournalEntry{
		{
			ID:               "demo-journal-1",
			PatientID:        demoPatientID,
			Text:             "Сегодня получилось выйти на прогулку, стало немного спокойнее.",
			ReflectionPrompt: "Что сегодня помогло вам почувствовать себя лучше?",
			CreatedAt:        now.AddDate(0, 0, -2),
			UpdatedAt:        now.AddDate(0, 0, -2),
		},
		{
			ID:               "demo-journal-2",
			PatientID:        demoPatientID,
			Text:             "Тревога была сильнее обычного, но я справилась с делами.",
			ReflectionPrompt: "Что тревожило вас сегодня?",
			CreatedAt:        now.AddDate(0, 0, -1),
			UpdatedAt:        now.AddDate(0, 0, -1),
		},
	}
	for _, entry := range entries {
		if err := store.CreateJournalEntry(ctx, entry); err != nil {
			return err
		}
	}

	checkIns := []domain.MoodCheckIn{
		{
			ID:        "demo-mood-1",
			PatientID: demoPatientID,
			Answers: []domain.MoodAnswer{
				{Question: "Настроение", Value: "Нейтральное"},
				{Question: "Тревожность", Value: "Умеренная"},
			},
			Comment:   "Было много работы",
			CreatedAt: now.AddDate(0, 0, -2),
		},
		{
			ID:        "demo-mood-2",
			PatientID: demoPatientID,
			Answers: []domain.MoodAnswer{
				{Question: "Настроение", Value: "Плохое"},
				{Question: "Тревожность", Value: "Высокая"},
			},
			Comment:   "Плохо спала",
			CreatedAt: now.AddDate(0, 0, -1),
		},
	}
	for _, checkIn := range checkIns {
		if err := store.CreateMoodCheckIn(ctx, checkIn); err != nil {
			return err
		}
	}
	return nil
}

func seedNotifications(
	ctx context.Context,
	store *sqlitestore.Store,
	now time.Time,
) error {
	notifications := []domain.Notification{
		{
			ID:          "demo-notification-1",
			RecipientID: demoDoctorID,
			PatientID:   demoPatientID,
			Type:        domain.NotificationStatusChanged,
			Message:     "Статус пациентки изменился: есть пропуски приёма лекарства.",
			CreatedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:          "demo-notification-2",
			RecipientID: demoDoctorID,
			PatientID:   demoPatientID,
			Type:        domain.NotificationSideEffect,
			Message:     "Пациентка сообщила о побочных эффектах: тошнота.",
			CreatedAt:   now.Add(-1 * time.Hour),
		},
	}
	for _, notification := range notifications {
		if err := store.CreateNotification(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func seedSupportContents(ctx context.Context, store *sqlitestore.Store) error {
	contents := []domain.SupportContent{
		{
			ID:          "demo-support-1",
			Trigger:     "after_journal",
			Message:     "Спасибо, что нашли время записать свои мысли. Это важный шаг.",
			ContentType: domain.ContentEncouragement,
		},
		{
			ID:          "demo-support-2",
			Trigger:     "after_mood",
			Message:     "Что сегодня помогло вам хотя бы немного? Можно записать это в дневник.",
			ContentType: domain.ContentReflection,
		},
		{
			ID:          "demo-support-3",
			Trigger:     "red_status",
			Message:     "Я вижу, тебе нужна помощь. Пожалуйста, свяжись со своим врачом.",
			ContentType: domain.ContentContactDoctor,
		},
		{
			ID:          "demo-support-4",
			Trigger:     "after_medication",
			Message:     "Спасибо, что сообщили о приёме. Если самочувствие ухудшится, обсудите это с врачом.",
			ContentType: domain.ContentEncouragement,
		},
	}
	for _, content := range contents {
		if err := store.CreateSupportContent(ctx, content); err != nil {
			return err
		}
	}
	return nil
}
