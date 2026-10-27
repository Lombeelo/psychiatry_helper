package appservice_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/backend/appservice"
	"example.com/backend/domain"
	"example.com/backend/httpapi"
	sqlitestore "example.com/backend/sqlite"
)

func newService(t *testing.T) *appservice.SQLiteService {
	t.Helper()

	store, err := sqlitestore.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	return appservice.NewSQLiteService(store)
}

func TestDoctorPatientFlow(t *testing.T) {
	service := newService(t)
	request := httptest.NewRequest("POST", "/", nil)

	patient, err := service.CreatePatient(request, "patient-1", httpapi.CreatePatientRequest{
		DisplayName:   "Пациент",
		DateOfBirth:   "1990-01-01",
		PolicyVersion: "v1",
		Accepted:      true,
	})
	if err != nil {
		t.Fatalf("create patient: %v", err)
	}
	if !patient.Code.Valid() {
		t.Fatalf("patient code %q is invalid", patient.Code.Value)
	}

	doctor, err := service.CreateDoctor(request, "doctor-1", httpapi.CreateDoctorRequest{
		DisplayName: "Врач",
		DateOfBirth: "1985-05-05",
		Specialty:   "Психотерапевт",
	})
	if err != nil {
		t.Fatalf("create doctor: %v", err)
	}
	if !doctor.Code.Valid() {
		t.Fatalf("doctor code %q is invalid", doctor.Code.Value)
	}

	relationship, err := service.CreateRelationship(
		request,
		doctor.ID,
		httpapi.CreateRelationshipRequest{PatientCode: patient.Code.Value},
	)
	if err != nil {
		t.Fatalf("create relationship: %v", err)
	}

	if _, err := service.CreateRelationship(
		request,
		doctor.ID,
		httpapi.CreateRelationshipRequest{PatientCode: patient.Code.Value},
	); !errors.Is(err, domain.ErrDuplicateRelationship) {
		t.Fatalf("expected ErrDuplicateRelationship, got %v", err)
	}

	now := time.Now().UTC()
	assignment, err := service.CreateAssignment(
		request,
		doctor.ID,
		httpapi.CreateAssignmentRequest{
			PatientID:  patient.ID,
			Title:      "Ежедневная отметка",
			StartsAt:   now.Add(-time.Hour),
			EndsAt:     now.Add(time.Hour),
			Frequency:  "DAILY",
			DaysOfWeek: []time.Weekday{time.Monday},
			TimesOfDay: []string{"09:00"},
			Type:       domain.AssignmentAppCheckIn,
			Details:    json.RawMessage(`{"instructions":"Отметьте состояние"}`),
		},
	)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	assignments, err := service.ListAssignments(request, doctor.ID, domain.RoleDoctor, patient.ID)
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if len(assignments) != 1 || assignments[0].ID != assignment.ID {
		t.Fatalf("unexpected assignments: %+v", assignments)
	}

	if _, err := service.CompleteAssignment(
		request,
		patient.ID,
		assignment.ID,
		httpapi.CompleteAssignmentRequest{Comment: "Готово"},
	); err != nil {
		t.Fatalf("complete assignment: %v", err)
	}

	if err := service.ChangeAssignmentStatus(
		request,
		doctor.ID,
		assignment.ID,
		"close",
	); err != nil {
		t.Fatalf("close assignment: %v", err)
	}
	if err := service.ChangeAssignmentStatus(
		request,
		doctor.ID,
		assignment.ID,
		"close",
	); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	medication, err := service.CreateAssignment(
		request,
		doctor.ID,
		httpapi.CreateAssignmentRequest{
			PatientID:  patient.ID,
			Title:      "Приём препарата",
			StartsAt:   now.Add(-time.Hour),
			EndsAt:     now.Add(time.Hour),
			Frequency:  "DAILY",
			DaysOfWeek: []time.Weekday{time.Monday},
			TimesOfDay: []string{"09:00"},
			Type:       domain.AssignmentMedication,
			Details:    json.RawMessage(`{"medication_id":"medication-1","dosage":{"amount":1,"unit":"мг"},"instructions":"После еды"}`),
		},
	)
	if err != nil {
		t.Fatalf("create medication assignment: %v", err)
	}

	if _, err := service.CompleteAssignment(
		request,
		patient.ID,
		medication.ID,
		httpapi.CompleteAssignmentRequest{},
	); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for missing intake, got %v", err)
	}

	completion, err := service.CompleteAssignment(
		request,
		patient.ID,
		medication.ID,
		httpapi.CompleteAssignmentRequest{
			MedicationIntake: &httpapi.MedicationIntakeRequest{
				ReportedAsTaken: true,
				DosageTaken:     "1 мг",
			},
			SideEffects: &httpapi.SideEffectReportRequest{
				Effects: []string{"Тошнота"},
				Comment: "Немного тошнит",
			},
		},
	)
	if err != nil {
		t.Fatalf("complete medication assignment: %v", err)
	}
	if completion.Intake == nil || !completion.Intake.ReportedAsTaken {
		t.Fatalf("medication intake was not saved: %+v", completion)
	}
	if completion.SideEffects == nil || len(completion.SideEffects.Effects) != 1 {
		t.Fatalf("side effects were not saved: %+v", completion)
	}

	patientAssignments, err := service.ListAssignments(
		request,
		patient.ID,
		domain.RolePatient,
		"",
	)
	if err != nil {
		t.Fatalf("list patient assignments: %v", err)
	}
	if len(patientAssignments) != 2 {
		t.Fatalf("expected 2 patient assignments, got %d", len(patientAssignments))
	}

	if _, err := service.CreateDoctor(request, "doctor-2", httpapi.CreateDoctorRequest{
		DisplayName: "Другой врач",
		DateOfBirth: "1980-01-01",
	}); err != nil {
		t.Fatalf("create second doctor: %v", err)
	}

	if err := service.EndRelationship(request, "doctor-2", relationship.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for someone else's relationship, got %v", err)
	}
	if err := service.EndRelationship(request, doctor.ID, relationship.ID); err != nil {
		t.Fatalf("end relationship: %v", err)
	}
}
