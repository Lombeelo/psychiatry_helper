package seed_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"example.com/backend/appservice"
	"example.com/backend/domain"
	"example.com/backend/security"
	"example.com/backend/seed"
	sqlitestore "example.com/backend/sqlite"
)

func TestEnsureDemoData(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	credentials, err := seed.EnsureDemoData(ctx, store)
	if err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if credentials.DoctorLogin != seed.DoctorLogin ||
		credentials.PatientLogin != seed.PatientLogin {
		t.Fatalf("unexpected credentials: %+v", credentials)
	}

	doctor, err := store.LoginAccountByLogin(ctx, seed.DoctorLogin)
	if err != nil {
		t.Fatalf("find doctor: %v", err)
	}
	if doctor.Account.Code.Role != domain.RoleDoctor {
		t.Fatalf("expected doctor role, got %q", doctor.Account.Code.Role)
	}
	if !security.VerifyPassword(
		seed.DoctorPassword,
		doctor.PasswordHash,
		doctor.PasswordSalt,
	) {
		t.Fatal("doctor password does not match")
	}

	patient, err := store.LoginAccountByLogin(ctx, seed.PatientLogin)
	if err != nil {
		t.Fatalf("find patient: %v", err)
	}
	if patient.Account.Code.Role != domain.RolePatient {
		t.Fatalf("expected patient role, got %q", patient.Account.Code.Role)
	}
	if patient.ConsentAccepted {
		t.Fatal("seeded patient should accept the policy during the demo")
	}

	request := httptest.NewRequest("GET", "/", nil)
	service := appservice.NewSQLiteService(store)

	authenticated, consent, _, err := service.Authenticate(
		request,
		seed.PatientLogin,
		seed.PatientPassword,
	)
	if err != nil {
		t.Fatalf("authenticate patient: %v", err)
	}
	if authenticated.ID != patient.Account.ID || consent {
		t.Fatalf("unexpected authentication result: %+v", authenticated)
	}

	patients, err := service.ListPatients(request, doctor.Account.ID)
	if err != nil {
		t.Fatalf("list patients: %v", err)
	}
	if len(patients) != 1 {
		t.Fatalf("expected one linked patient, got %d", len(patients))
	}
	if patients[0].Status.Color != string(domain.StatusYellow) {
		t.Fatalf("expected yellow status, got %+v", patients[0].Status)
	}
	if !patients[0].Status.DataSufficient {
		t.Fatal("expected enough data for the status")
	}

	doctorInfo, err := service.MyDoctor(request, patient.Account.ID)
	if err != nil {
		t.Fatalf("my doctor: %v", err)
	}
	if doctorInfo.Code != doctor.Account.Code.Value {
		t.Fatalf("unexpected doctor code: %q", doctorInfo.Code)
	}
}
